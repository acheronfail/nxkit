package nsz

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/acheronfail/nxkit/lib/keys"
	"github.com/acheronfail/nxkit/lib/nca"
	"github.com/acheronfail/nxkit/lib/pfs0"
	"github.com/klauspost/compress/zstd"
)

const (
	ncaHeaderSize = 0x4000

	nczSectionMagic = "NCZSECTN"
	nczBlockMagic   = "NCZBLOCK"
)

type ProgressFunc func(done, total int64)

type progressCounter struct {
	done     int64
	total    int64
	progress ProgressFunc
}

func newProgressCounter(total int64, progress ProgressFunc) *progressCounter {
	if progress != nil {
		progress(0, total)
	}
	return &progressCounter{total: total, progress: progress}
}

func (p *progressCounter) Advance(n int64) {
	if n <= 0 {
		return
	}
	p.done += n
	if p.progress != nil {
		p.progress(min(p.done, p.total), p.total)
	}
}

func (p *progressCounter) Complete() {
	p.done = p.total
	if p.progress != nil {
		p.progress(p.total, p.total)
	}
}

type countingReader struct {
	reader   io.Reader
	progress *progressCounter
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.progress.Advance(int64(n))
	return n, err
}

func CompressedPath(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".nsp") {
		return "", fmt.Errorf("expected an .nsp file, got %s", filepath.Base(path))
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".nsz", nil
}

func DecompressedPath(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".nsz") {
		return "", fmt.Errorf("expected an .nsz file, got %s", filepath.Base(path))
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".nsp", nil
}

func CompressNSPWithProgressContext(ctx context.Context, inPath, outPath string, keyset keys.Keys, progress ProgressFunc) error {
	var titleKeys map[[0x10]byte][]byte
	return transformPFS0(ctx, inPath, outPath, progress, func(fs *pfs0.Pfs0Fs) error {
		var err error
		titleKeys, err = titleKeysFromPFS0(fs, keyset)
		return err
	}, func(ctx context.Context, entry pfs0.Pfs0Entry, out io.Writer, p *progressCounter) (string, error) {
		reader, err := entry.Open()
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".nca") {
			_, err := copyWithProgress(ctx, out, reader, int64(entry.Size()), p)
			return entry.Name(), err
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) + ".ncz"
		err = compressNCZWithProgressCounter(ctx, reader, int64(entry.Size()), out, keyset, titleKeys, p)
		return name, err
	})
}

func DecompressNSZWithProgressContext(ctx context.Context, inPath, outPath string, progress ProgressFunc) error {
	return transformPFS0(ctx, inPath, outPath, progress, nil, func(ctx context.Context, entry pfs0.Pfs0Entry, out io.Writer, p *progressCounter) (string, error) {
		reader, err := entry.Open()
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".ncz") {
			_, err := copyWithProgress(ctx, out, reader, int64(entry.Size()), p)
			return entry.Name(), err
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) + ".nca"
		err = decompressNCZWithProgressCounter(ctx, reader, out, p)
		return name, err
	})
}

func transformPFS0(ctx context.Context, inPath, outPath string, progress ProgressFunc, prepare func(*pfs0.Pfs0Fs) error, transform func(context.Context, pfs0.Pfs0Entry, io.Writer, *progressCounter) (string, error)) error {
	if filepath.Clean(inPath) == filepath.Clean(outPath) {
		return errors.New("input and output paths must be different")
	}

	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()

	stat, err := in.Stat()
	if err != nil {
		return err
	}

	fs, err := pfs0.NewPfs0(in)
	if err != nil {
		return err
	}
	if prepare != nil {
		if err := prepare(fs); err != nil {
			return err
		}
	}

	tmpDir, err := os.MkdirTemp(filepath.Dir(outPath), ".nxkit-nsz-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	counter := newProgressCounter(stat.Size(), progress)
	entries := make([]pfs0.BuildEntry, 0, len(fs.Entries()))
	for i, entry := range fs.Entries() {
		if err := ctx.Err(); err != nil {
			return err
		}

		tmpPath := filepath.Join(tmpDir, fmt.Sprintf("%04d-%s", i, filepath.Base(entry.Name())))
		tmp, err := os.Create(tmpPath)
		if err != nil {
			return err
		}
		name, transformErr := transform(ctx, entry, tmp, counter)
		closeErr := tmp.Close()
		if transformErr != nil {
			return transformErr
		}
		if closeErr != nil {
			return closeErr
		}
		entries = append(entries, pfs0.BuildEntry{Name: name, Path: tmpPath})
	}

	if _, err := pfs0.Build(outPath, entries); err != nil {
		return err
	}
	counter.Complete()
	return nil
}

func titleKeysFromPFS0(fs *pfs0.Pfs0Fs, keyset keys.Keys) (map[[0x10]byte][]byte, error) {
	titleKeys := make(map[[0x10]byte][]byte)
	for _, entry := range fs.Entries() {
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".tik") {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		rightsID, encryptedTitleKey, masterKeyIndex, err := parseTicket(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ticket %s: %w", entry.Name(), err)
		}
		if isZero16(rightsID) || isZero(encryptedTitleKey) {
			continue
		}
		if masterKeyIndex >= len(keyset.Titlekek) || len(keyset.Titlekek[masterKeyIndex]) != aes.BlockSize {
			return nil, fmt.Errorf("missing titlekek_%02x required by ticket %s", masterKeyIndex, entry.Name())
		}
		block, err := aes.NewCipher(keyset.Titlekek[masterKeyIndex])
		if err != nil {
			return nil, err
		}
		titleKey := make([]byte, aes.BlockSize)
		block.Decrypt(titleKey, encryptedTitleKey)
		titleKeys[rightsID] = titleKey
	}
	return titleKeys, nil
}

func parseTicket(data []byte) ([0x10]byte, []byte, int, error) {
	var rightsID [0x10]byte
	if len(data) < 4 {
		return rightsID, nil, 0, io.ErrUnexpectedEOF
	}

	signatureSizes := map[uint32]int{
		0x010000: 0x200,
		0x010001: 0x100,
		0x010002: 0x3c,
		0x010003: 0x200,
		0x010004: 0x100,
		0x010005: 0x3c,
	}
	signatureType := binary.LittleEndian.Uint32(data[:4])
	signatureSize, ok := signatureSizes[signatureType]
	if !ok {
		return rightsID, nil, 0, fmt.Errorf("invalid signature type 0x%x", signatureType)
	}
	padding := 0x40 - ((signatureSize + 4) % 0x40)
	base := 4 + signatureSize + padding
	if len(data) < base+0x170 {
		return rightsID, nil, 0, io.ErrUnexpectedEOF
	}

	encryptedTitleKey := bytes.Clone(data[base+0x40 : base+0x50])
	copy(rightsID[:], data[base+0x160:base+0x170])

	revision := int(data[base+0x145])
	if revision == 0 {
		revision = int(data[base+0x146])
	}
	if revision > 0 {
		revision--
	}
	return rightsID, encryptedTitleKey, revision, nil
}

func isZero16(value [0x10]byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

func isZero(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

type nczSection struct {
	offset        int64
	size          int64
	cryptType     nca.SectionCryptType
	key           []byte
	counter       [0x10]byte
	compressedKey bool
}

func CompressNCZWithProgressContext(ctx context.Context, src interface {
	io.Reader
	io.ReaderAt
}, srcSize int64, dst io.Writer, keyset keys.Keys, progress ProgressFunc) error {
	return compressNCZWithProgressCounter(ctx, src, srcSize, dst, keyset, nil, newProgressCounter(srcSize, progress))
}

func compressNCZWithProgressCounter(ctx context.Context, src interface {
	io.Reader
	io.ReaderAt
}, srcSize int64, dst io.Writer, keyset keys.Keys, titleKeys map[[0x10]byte][]byte, progress *progressCounter) error {
	if srcSize < ncaHeaderSize {
		return fmt.Errorf("NCA is too small: %d bytes", srcSize)
	}

	archive, err := nca.NewNcaWithTitleKeys(src, keyset, titleKeys)
	if err != nil {
		return fmt.Errorf("failed to parse NCA: %w", err)
	}

	sections := make([]nczSection, 0)
	for _, section := range archive.Sections() {
		info := nczSection{
			offset:    section.Offset(),
			size:      section.Size(),
			cryptType: section.CryptType(),
			key:       section.Key(),
			counter:   section.Counter(),
		}
		switch info.cryptType {
		case nca.CryptNone:
		case nca.CryptCtr, nca.CryptBktr:
			info.compressedKey = true
			if archive.HasRightsID() && !archive.HasTitleKey() {
				return fmt.Errorf("missing title key for rights ID %x", archive.RightsID())
			}
			if len(info.key) != aes.BlockSize {
				return fmt.Errorf("section at 0x%x has an invalid crypto key", info.offset)
			}
		default:
			return fmt.Errorf("section at 0x%x uses unsupported crypto type %s", info.offset, info.cryptType)
		}
		sections = append(sections, info)
	}

	if len(sections) == 0 {
		return errors.New("NCA contains no compressible sections")
	}

	slices.SortFunc(sections, func(a, b nczSection) int {
		if a.offset < b.offset {
			return -1
		}
		if a.offset > b.offset {
			return 1
		}
		return 0
	})

	header := make([]byte, ncaHeaderSize)
	if _, err := src.ReadAt(header, 0); err != nil {
		return err
	}
	progress.Advance(ncaHeaderSize)
	if _, err := dst.Write(header); err != nil {
		return err
	}
	if err := writeSectionHeader(dst, sections); err != nil {
		return err
	}

	encoder, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return err
	}

	pos := int64(ncaHeaderSize)
	for _, section := range sections {
		if err := ctx.Err(); err != nil {
			return err
		}
		if section.offset < pos {
			return fmt.Errorf("NCA sections overlap around 0x%x", section.offset)
		}
		if section.offset > pos {
			if _, err := copyRangeWithProgress(ctx, encoder, src, pos, section.offset-pos, nil, progress); err != nil {
				return err
			}
			pos = section.offset
		}
		if _, err := copyRangeWithProgress(ctx, encoder, src, section.offset, section.size, &section, progress); err != nil {
			return err
		}
		pos = section.offset + section.size
	}

	return encoder.Close()
}

func DecompressNCZWithProgressContext(ctx context.Context, src io.Reader, srcSize int64, dst io.Writer, progress ProgressFunc) error {
	return decompressNCZWithProgressCounter(ctx, src, dst, newProgressCounter(srcSize, progress))
}

func decompressNCZWithProgressCounter(ctx context.Context, src io.Reader, dst io.Writer, progress *progressCounter) error {
	reader := &countingReader{reader: src, progress: progress}

	header := make([]byte, ncaHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	if _, err := dst.Write(header); err != nil {
		return err
	}

	sections, err := readSectionHeader(reader)
	if err != nil {
		return err
	}
	if len(sections) == 0 {
		return errors.New("NCZ contains no sections")
	}

	peek := make([]byte, len(nczBlockMagic))
	if _, err := io.ReadFull(reader, peek); err != nil {
		return err
	}
	if string(peek) == nczBlockMagic {
		return decompressBlockNCZ(ctx, reader, dst, sections)
	}
	return decompressSolidNCZ(ctx, io.MultiReader(bytes.NewReader(peek), reader), dst, sections)
}

func writeSectionHeader(dst io.Writer, sections []nczSection) error {
	if _, err := dst.Write([]byte(nczSectionMagic)); err != nil {
		return err
	}
	if err := binary.Write(dst, binary.LittleEndian, uint64(len(sections))); err != nil {
		return err
	}
	for _, section := range sections {
		values := []uint64{
			uint64(section.offset),
			uint64(section.size),
			uint64(section.cryptType),
			0,
		}
		for _, value := range values {
			if err := binary.Write(dst, binary.LittleEndian, value); err != nil {
				return err
			}
		}
		var key [0x10]byte
		if section.compressedKey {
			copy(key[:], section.key)
		}
		if _, err := dst.Write(key[:]); err != nil {
			return err
		}
		if _, err := dst.Write(section.counter[:]); err != nil {
			return err
		}
	}
	return nil
}

func readSectionHeader(src io.Reader) ([]nczSection, error) {
	magic := make([]byte, len(nczSectionMagic))
	if _, err := io.ReadFull(src, magic); err != nil {
		return nil, err
	}
	if string(magic) != nczSectionMagic {
		return nil, fmt.Errorf("missing %s header", nczSectionMagic)
	}

	var sectionCount uint64
	if err := binary.Read(src, binary.LittleEndian, &sectionCount); err != nil {
		return nil, err
	}
	if sectionCount > 4096 {
		return nil, fmt.Errorf("unreasonable NCZ section count: %d", sectionCount)
	}

	sections := make([]nczSection, 0, sectionCount)
	for range sectionCount {
		var offset, size, cryptType, padding uint64
		for _, target := range []*uint64{&offset, &size, &cryptType, &padding} {
			if err := binary.Read(src, binary.LittleEndian, target); err != nil {
				return nil, err
			}
		}
		section := nczSection{
			offset:    int64(offset),
			size:      int64(size),
			cryptType: nca.SectionCryptType(cryptType),
			key:       make([]byte, 0x10),
		}
		if _, err := io.ReadFull(src, section.key); err != nil {
			return nil, err
		}
		if _, err := io.ReadFull(src, section.counter[:]); err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}

	slices.SortFunc(sections, func(a, b nczSection) int {
		if a.offset < b.offset {
			return -1
		}
		if a.offset > b.offset {
			return 1
		}
		return 0
	})
	return sections, nil
}

func decompressSolidNCZ(ctx context.Context, compressed io.Reader, dst io.Writer, sections []nczSection) error {
	decoder, err := zstd.NewReader(compressed)
	if err != nil {
		return err
	}
	defer decoder.Close()

	for _, section := range logicalSections(sections) {
		if err := ctx.Err(); err != nil {
			return err
		}
		limited := io.LimitReader(decoder, section.size)
		if err := copyDecodedSection(ctx, dst, limited, section); err != nil {
			return err
		}
	}
	return nil
}

type blockHeader struct {
	version             byte
	blockType           byte
	blockSizeExponent   byte
	numberOfBlocks      uint32
	decompressedSize    uint64
	compressedBlockSize []uint32
}

func decompressBlockNCZ(ctx context.Context, src io.Reader, dst io.Writer, sections []nczSection) error {
	block, err := readBlockHeader(src)
	if err != nil {
		return err
	}
	if block.blockSizeExponent < 14 || block.blockSizeExponent > 32 {
		return fmt.Errorf("invalid NCZ block size exponent: %d", block.blockSizeExponent)
	}
	blockSize := int64(1) << block.blockSizeExponent
	if block.version != 2 {
		return fmt.Errorf("unsupported NCZ block version: %d", block.version)
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return err
	}
	defer decoder.Close()

	sections = logicalSections(sections)
	sectionIndex := 0
	pos := sections[0].offset
	for i, compressedSize := range block.compressedBlockSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		data := make([]byte, compressedSize)
		if _, err := io.ReadFull(src, data); err != nil {
			return err
		}
		decompressedBlockSize := blockSize
		if i == len(block.compressedBlockSize)-1 && block.decompressedSize%uint64(blockSize) != 0 {
			decompressedBlockSize = int64(block.decompressedSize % uint64(blockSize))
		}
		if int64(compressedSize) < decompressedBlockSize {
			data, err = decoder.DecodeAll(data, nil)
			if err != nil {
				return err
			}
		}
		if int64(len(data)) != decompressedBlockSize {
			return fmt.Errorf("NCZ block %d decompressed to %d bytes, expected %d", i, len(data), decompressedBlockSize)
		}

		remaining := data
		for len(remaining) > 0 {
			if sectionIndex >= len(sections) {
				return errors.New("NCZ block data exceeds declared sections")
			}
			section := sections[sectionIndex]
			if pos >= section.offset+section.size {
				sectionIndex++
				continue
			}
			partSize := min(int64(len(remaining)), section.offset+section.size-pos)
			part := remaining[:partSize]
			if err := writeDecodedPart(dst, part, section, pos); err != nil {
				return err
			}
			remaining = remaining[partSize:]
			pos += partSize
		}
	}
	return nil
}

func readBlockHeader(src io.Reader) (blockHeader, error) {
	var fixed [16]byte
	if _, err := io.ReadFull(src, fixed[:]); err != nil {
		return blockHeader{}, err
	}
	block := blockHeader{
		version:           fixed[0],
		blockType:         fixed[1],
		blockSizeExponent: fixed[3],
		numberOfBlocks:    binary.LittleEndian.Uint32(fixed[4:8]),
		decompressedSize:  binary.LittleEndian.Uint64(fixed[8:16]),
	}
	if block.numberOfBlocks > 1<<24 {
		return blockHeader{}, fmt.Errorf("unreasonable NCZ block count: %d", block.numberOfBlocks)
	}
	block.compressedBlockSize = make([]uint32, block.numberOfBlocks)
	for i := range block.compressedBlockSize {
		if err := binary.Read(src, binary.LittleEndian, &block.compressedBlockSize[i]); err != nil {
			return blockHeader{}, err
		}
	}
	return block, nil
}

func logicalSections(sections []nczSection) []nczSection {
	out := make([]nczSection, 0, len(sections)+1)
	if sections[0].offset > ncaHeaderSize {
		out = append(out, nczSection{
			offset:    ncaHeaderSize,
			size:      sections[0].offset - ncaHeaderSize,
			cryptType: nca.CryptNone,
		})
	}
	out = append(out, sections...)
	return out
}

func copyDecodedSection(ctx context.Context, dst io.Writer, src io.Reader, section nczSection) error {
	buf := make([]byte, 1024*1024)
	pos := section.offset
	remaining := section.size
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		readSize := min(int64(len(buf)), remaining)
		n, err := io.ReadFull(src, buf[:readSize])
		if err != nil {
			return err
		}
		if err := writeDecodedPart(dst, buf[:n], section, pos); err != nil {
			return err
		}
		pos += int64(n)
		remaining -= int64(n)
	}
	return nil
}

func writeDecodedPart(dst io.Writer, data []byte, section nczSection, pos int64) error {
	if section.cryptType == nca.CryptCtr || section.cryptType == nca.CryptBktr {
		var err error
		data, err = cryptCTR(data, section.key, section.counter, pos)
		if err != nil {
			return err
		}
	}
	_, err := dst.Write(data)
	return err
}

func copyRangeWithProgress(ctx context.Context, dst io.Writer, src io.ReaderAt, off, size int64, section *nczSection, progress *progressCounter) (int64, error) {
	var reader io.Reader = io.NewSectionReader(src, off, size)
	if section != nil && (section.cryptType == nca.CryptCtr || section.cryptType == nca.CryptBktr) {
		block, err := aes.NewCipher(section.key)
		if err != nil {
			return 0, err
		}
		iv := ctrForOffset(section.counter, off)
		reader = &streamSectionReader{
			reader: io.NewSectionReader(src, off, size),
			stream: cipher.NewCTR(block, iv),
		}
	}
	return copyWithProgress(ctx, dst, reader, size, progress)
}

type streamSectionReader struct {
	reader io.Reader
	stream cipher.Stream
}

func (r *streamSectionReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.stream.XORKeyStream(p[:n], p[:n])
	}
	return n, err
}

func copyWithProgress(ctx context.Context, dst io.Writer, src io.Reader, size int64, progress *progressCounter) (int64, error) {
	buf := make([]byte, 1024*1024)
	var copied int64
	for copied < size {
		if err := ctx.Err(); err != nil {
			return copied, err
		}
		readSize := min(int64(len(buf)), size-copied)
		n, readErr := src.Read(buf[:readSize])
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return copied, writeErr
			}
			if written != n {
				return copied, io.ErrShortWrite
			}
			copied += int64(written)
			progress.Advance(int64(written))
		}
		if readErr != nil {
			if readErr == io.EOF && copied == size {
				break
			}
			return copied, readErr
		}
	}
	return copied, nil
}

func cryptCTR(data []byte, key []byte, counter [0x10]byte, offset int64) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := bytes.Clone(data)
	cipher.NewCTR(block, ctrForOffset(counter, offset)).XORKeyStream(out, out)
	return out, nil
}

func ctrForOffset(counter [0x10]byte, offset int64) []byte {
	iv := bytes.Clone(counter[:])
	binary.BigEndian.PutUint64(iv[8:], uint64(offset)>>4)
	return iv
}

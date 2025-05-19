package pfs0

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	Psf0Magic     = 0x30534650
	headerSize    = 16
	entryListSize = 24
)

type Pfs0Reader interface {
	io.Reader
	io.ReaderAt
}

type pfs0Header struct {
	magic           uint32
	numFiles        uint32
	stringTableSize uint32
	_               uint32
}

type pfs0EntryListing struct {
	offset            uint64
	size              uint64
	stringTableOffset uint32
	_                 uint32
}

type Pfs0Entry struct {
	name  string
	size  uint64
	fs    *Pfs0Fs
	index int
}

func (entry *Pfs0Entry) Open() (*io.SectionReader, error) {
	if entry.fs == nil || entry.index >= len(entry.fs.entries) {
		return nil, os.ErrInvalid
	}

	e := entry.fs.entries[entry.index]
	return io.NewSectionReader(entry.fs.reader, entry.fs.dataOffset+int64(e.offset), int64(e.size)), nil
}

func (entry *Pfs0Entry) Name() string {
	return entry.name
}

func (entry *Pfs0Entry) Size() uint64 {
	return entry.size
}

type Pfs0Fs struct {
	reader      Pfs0Reader
	header      pfs0Header
	entries     []pfs0EntryListing
	stringTable []string
	dataOffset  int64
}

func (fs *Pfs0Fs) Entries() []Pfs0Entry {
	entries := make([]Pfs0Entry, 0)
	for i, e := range fs.entries {
		entries = append(entries, Pfs0Entry{
			name:  fs.stringTable[i],
			size:  e.size,
			fs:    fs,
			index: i,
		})
	}

	return entries
}

func NewPfs0(reader Pfs0Reader) (*Pfs0Fs, error) {
	headerBytes := make([]byte, headerSize)
	if _, err := reader.Read(headerBytes); err != nil {
		return nil, fmt.Errorf("failed to read header %s", err)
	}

	header, err := parseHeader(headerBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header %s", err)
	}

	entryListingSize := int(entryListSize * header.numFiles)
	entryListingBytes := make([]byte, entryListingSize)
	if _, err = reader.Read(entryListingBytes); err != nil {
		return nil, fmt.Errorf("failed to read entry listing %s", err)
	}

	entries, err := parseEntryList(entryListingBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse entry listing %s", err)
	}

	stringTableBytes := make([]byte, header.stringTableSize)
	if _, err = reader.Read(stringTableBytes); err != nil {
		return nil, fmt.Errorf("failed to read string table %s", err)
	}

	stringTable, err := parseStringTable(entries, stringTableBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse string table %s", err)
	}

	return &Pfs0Fs{
		reader:      reader,
		header:      *header,
		entries:     entries,
		stringTable: stringTable,
		dataOffset:  int64(len(headerBytes) + len(entryListingBytes) + len(stringTableBytes)),
	}, nil
}

// data starts from beginning of file
func parseHeader(data []byte) (*pfs0Header, error) {
	header := pfs0Header{
		magic:           binary.LittleEndian.Uint32(data[0:4]),
		numFiles:        binary.LittleEndian.Uint32(data[4:8]),
		stringTableSize: binary.LittleEndian.Uint32(data[8:12]),
	}

	if header.magic != Psf0Magic {
		return nil, fmt.Errorf("unrecognised magic value; expected %x got %x", Psf0Magic, header.magic)
	}

	return &header, nil
}

// data starts right after header
func parseEntryList(data []byte) ([]pfs0EntryListing, error) {
	entries := make([]pfs0EntryListing, 0, len(data)/entryListSize)
	for i := 0; i < len(data); i += entryListSize {
		entry := pfs0EntryListing{
			offset:            binary.LittleEndian.Uint64(data[i : i+0x8]),
			size:              binary.LittleEndian.Uint64(data[i+0x8 : i+0x10]),
			stringTableOffset: binary.LittleEndian.Uint32(data[i+0x10 : i+0x14]),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// data starts right after file entries
func parseStringTable(entries []pfs0EntryListing, data []byte) ([]string, error) {
	table := make([]string, 0)
	for _, entry := range entries {
		sub := data[entry.stringTableOffset:]
		end := bytes.IndexByte(sub, 0x00)
		if end == -1 {
			return nil, fmt.Errorf("failed to parse string table - unterminated string found")
		}

		s := string(sub[:end])
		table = append(table, s)
	}

	return table, nil
}

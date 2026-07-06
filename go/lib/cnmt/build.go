package cnmt

import (
	"encoding/binary"
	"fmt"
)

const (
	TypeApplication          byte = 0x80
	ContentTypeProgram       byte = 0x01
	ContentTypeMeta          byte = 0x02
	ContentTypeControl       byte = 0x03
	ContentTypeHtmlDocument  byte = 0x04
	ContentTypeLegalInfo     byte = 0x05
	applicationExtHeaderSize      = 0x10
	contentRecordSize             = 0x38
)

type ContentRecord struct {
	Hash  [32]byte
	NCAID [16]byte
	Size  [6]byte
	Type  byte
	ID    byte
}

func BuildApplication(titleID uint64, records []ContentRecord) ([]byte, error) {
	if len(records) > 0xffff {
		return nil, fmt.Errorf("too many CNMT content records: %d", len(records))
	}

	out := make([]byte, 0, 0x20+applicationExtHeaderSize+len(records)*contentRecordSize+0x20)
	header := make([]byte, 0x20)
	binary.LittleEndian.PutUint64(header[0x0:], titleID)
	header[0xc] = TypeApplication
	binary.LittleEndian.PutUint16(header[0xe:], uint16(applicationExtHeaderSize))
	binary.LittleEndian.PutUint16(header[0x10:], uint16(len(records)))
	out = append(out, header...)

	extHeader := make([]byte, applicationExtHeaderSize)
	binary.LittleEndian.PutUint64(extHeader[0x0:], titleID+0x800)
	out = append(out, extHeader...)

	for _, record := range records {
		out = appendContentRecord(out, record)
	}

	out = append(out, make([]byte, 0x20)...)
	return out, nil
}

func appendContentRecord(out []byte, record ContentRecord) []byte {
	out = append(out, record.Hash[:]...)
	out = append(out, record.NCAID[:]...)
	out = append(out, record.Size[:]...)
	out = append(out, record.Type, record.ID)
	return out
}

package cnmt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildApplication(t *testing.T) {
	var program ContentRecord
	var control ContentRecord
	for i := range program.Hash {
		program.Hash[i] = byte(i)
		control.Hash[i] = byte(0x80 + i)
	}
	for i := range program.NCAID {
		program.NCAID[i] = byte(0x20 + i)
		control.NCAID[i] = byte(0xa0 + i)
	}
	copy(program.Size[:], []byte{1, 2, 3, 4, 5, 6})
	program.Type = ContentTypeProgram
	copy(control.Size[:], []byte{7, 8, 9, 10, 11, 12})
	control.Type = ContentTypeControl

	titleID := uint64(0x0162696bc58e0000)
	data, err := BuildApplication(titleID, []ContentRecord{program, control})
	if err != nil {
		t.Fatalf("BuildApplication failed: %v", err)
	}

	if got, want := len(data), 0x20+0x10+2*contentRecordSize+0x20; got != want {
		t.Fatalf("size = %d, want %d", got, want)
	}
	if got := binary.LittleEndian.Uint64(data[0x0:]); got != titleID {
		t.Fatalf("title ID = 0x%016x, want 0x%016x", got, titleID)
	}
	if got := data[0xc]; got != TypeApplication {
		t.Fatalf("type = 0x%02x, want 0x%02x", got, TypeApplication)
	}
	if got := binary.LittleEndian.Uint16(data[0xe:]); got != applicationExtHeaderSize {
		t.Fatalf("extended header size = %d, want %d", got, applicationExtHeaderSize)
	}
	if got := binary.LittleEndian.Uint16(data[0x10:]); got != 2 {
		t.Fatalf("content record count = %d, want 2", got)
	}
	if got := binary.LittleEndian.Uint16(data[0x12:]); got != 0 {
		t.Fatalf("meta entry count = %d, want 0", got)
	}

	extOffset := 0x20
	if got, want := binary.LittleEndian.Uint64(data[extOffset:]), titleID+0x800; got != want {
		t.Fatalf("patch title ID = 0x%016x, want 0x%016x", got, want)
	}
	if !bytes.Equal(data[extOffset+0x8:extOffset+0x10], make([]byte, 0x8)) {
		t.Fatal("extended application header trailing bytes are not zero")
	}

	assertContentRecord(t, data[0x30:0x30+contentRecordSize], program)
	assertContentRecord(t, data[0x30+contentRecordSize:0x30+2*contentRecordSize], control)
	if !bytes.Equal(data[len(data)-0x20:], make([]byte, 0x20)) {
		t.Fatal("digest placeholder is not zero-filled")
	}
}

func TestBuildApplicationAllowsNoContentRecords(t *testing.T) {
	data, err := BuildApplication(0x0100000000000000, nil)
	if err != nil {
		t.Fatalf("BuildApplication failed: %v", err)
	}
	if got, want := len(data), 0x20+0x10+0x20; got != want {
		t.Fatalf("size = %d, want %d", got, want)
	}
	if got := binary.LittleEndian.Uint16(data[0x10:]); got != 0 {
		t.Fatalf("content record count = %d, want 0", got)
	}
}

func TestBuildApplicationRejectsTooManyContentRecords(t *testing.T) {
	_, err := BuildApplication(0x0100000000000000, make([]ContentRecord, 0x10000))
	if err == nil {
		t.Fatal("BuildApplication succeeded with too many content records")
	}
}

func assertContentRecord(t *testing.T, data []byte, record ContentRecord) {
	t.Helper()
	if !bytes.Equal(data[0x00:0x20], record.Hash[:]) {
		t.Fatalf("hash = %x, want %x", data[0x00:0x20], record.Hash)
	}
	if !bytes.Equal(data[0x20:0x30], record.NCAID[:]) {
		t.Fatalf("NCA ID = %x, want %x", data[0x20:0x30], record.NCAID)
	}
	if !bytes.Equal(data[0x30:0x36], record.Size[:]) {
		t.Fatalf("size = %x, want %x", data[0x30:0x36], record.Size)
	}
	if data[0x36] != record.Type {
		t.Fatalf("type = 0x%02x, want 0x%02x", data[0x36], record.Type)
	}
	if data[0x37] != record.ID {
		t.Fatalf("ID = 0x%02x, want 0x%02x", data[0x37], record.ID)
	}
}

package fat12

import "testing"

func TestCreateFilesystem(t *testing.T) {
	fs, err := NewFromPath("../fixtures/fat16/disk.img")
	if err != nil {
		t.Fatalf("failed to create filesystem: %v", err)
	}
	defer fs.Close()
}

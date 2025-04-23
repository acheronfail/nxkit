package nand

import (
	"bytes"
	"testing"
)

func TestLayer_NoCrypto(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)

		writeAt(disk, 0, []byte{1, 2, 3, 4, 5})
		assertBytes(t, readAt(layer, 0, 5), []byte{1, 2, 3, 4, 5})
		assertBytes(t, readAt(layer, 5, 5), []byte{0, 0, 0, 0, 0})
	})

	t.Run("write", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)

		writeAt(layer, 0, []byte{1, 2, 3, 4, 5})
		assertBytes(t, readAt(disk, 0, 5), []byte{1, 2, 3, 4, 5})
		assertBytes(t, readAt(layer, 5, 5), []byte{0, 0, 0, 0, 0})
	})
}

func TestLayer_Crypto(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		writeAt(disk, 0, []byte{42, 42, 42, 42, 42})
		writeAt(disk, CLUSTER_SIZE, []byte{42, 42, 42, 42, 42})

		assertBytes(t, readAt(layer, 0, 5), []byte{42, 42, 42, 42, 42})
		assertBytes(t, readAt(layer, CLUSTER_SIZE, 5), []byte{43, 43, 43, 43, 43})
	})

	t.Run("read over cluster bounds", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		// 1st cluster: all `0`
		// 2nd cluster: all `42` (XOR'd value is 43)
		// 3rd cluster: all `0`
		// (rest all `1`)
		writeAt(disk, CLUSTER_SIZE, bytes.Repeat([]byte{43}, int(CLUSTER_SIZE)))

		// read 1st cluster + first byte from 2nd cluster
		assertBytes(t, readAt(layer, 0, int(CLUSTER_SIZE)+1), expand([]Rle{{0, 16}, {42, 1}}))
		// read 2nd cluster + last byte from 1st cluster
		assertBytes(t, readAt(layer, CLUSTER_SIZE-1, int(CLUSTER_SIZE)+1), expand([]Rle{{0, 1}, {42, 16}}))
		// read last byte from 1st cluster, entire 2nd cluster, first byte from 3rd cluster
		assertBytes(t, readAt(layer, CLUSTER_SIZE-1, int(CLUSTER_SIZE)+2), expand([]Rle{{0, 1}, {42, 16}, {0, 1}}))
	})

	t.Run("write", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		// set first 5 bytes of 1st and 2nd cluster to `42`
		assertLength(t, writeAt(layer, 0, []byte{42, 42, 42, 42, 42}), int(CLUSTER_SIZE))
		assertLength(t, writeAt(layer, CLUSTER_SIZE, []byte{42, 42, 42, 42, 42}), int(CLUSTER_SIZE))

		assertBytes(t, readAt(disk, 0, int(CLUSTER_SIZE)), expand([]Rle{{42, 5}, {0, int(CLUSTER_SIZE) - 5}}))
		assertBytes(t, readAt(disk, CLUSTER_SIZE, int(CLUSTER_SIZE)), expand([]Rle{{43, 5}, {1, int(CLUSTER_SIZE) - 5}}))
	})

	t.Run("write: entire 1st cluster + first byte of 2nd cluster", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		assertLength(t, writeAt(layer, 0, expand([]Rle{{42, int((CLUSTER_SIZE) + 1)}})), int(CLUSTER_SIZE)*2)
		assertBytes(t, readAt(disk, 0, int(CLUSTER_SIZE)+1), expand([]Rle{{42, int(CLUSTER_SIZE)}, {43, 1}}))
	})

	t.Run("write: last byte of 1st cluster + entire 2nd cluster", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		assertLength(t, writeAt(layer, CLUSTER_SIZE-1, expand([]Rle{{42, int((CLUSTER_SIZE) + 1)}})), int(CLUSTER_SIZE)*2)
		assertBytes(t, readAt(disk, 0, int(CLUSTER_SIZE)*2), expand([]Rle{{0, int(CLUSTER_SIZE) - 1}, {42, 1}, {43, int(CLUSTER_SIZE)}}))
	})

	t.Run("write: last byte of 1st cluster + entire 2nd cluster + first byte of 3rd cluster", func(t *testing.T) {
		layer, disk := getLayer(DISK_SIZE)
		disk.zeroWithXor()
		layer.SetCrypto(NewXorCrypto())

		assertLength(t, writeAt(layer, CLUSTER_SIZE-1, expand([]Rle{{42, int((CLUSTER_SIZE) + 2)}})), int(CLUSTER_SIZE)*3)
		assertBytes(t, readAt(disk, 0, int(CLUSTER_SIZE)*3), expand([]Rle{
			{0, int(CLUSTER_SIZE) - 1},
			{42, 1},
			{43, int(CLUSTER_SIZE)},
			{40, 1},
			{2, int(CLUSTER_SIZE) - 1},
		}))
	})
}

package nacp

// Reference: https://switchbrew.org/wiki/NACP
// Adapted from: https://github.com/TooTallNate/switch-tools/
// More:
//   - https://github.com/switchbrew/switch-tools/blob/master/src/nacptool.c
//   - https://switchbrew.github.io/libnx/nacp_8h_source.html
//   - https://www.retroreversing.com/SwitchFileFormats

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf8"
)

const (
	VideoCaptureDisabled = iota
	VideoCaptureEnabled
	VideoCaptureAutomatic
)

type Nacp struct {
	buffer []byte
}

func NewNacp(buffer []byte) *Nacp {
	if buffer == nil {
		buffer = make([]byte, 0x4000)

		binary.LittleEndian.PutUint32(buffer[0x3024:], 0x100)
		binary.LittleEndian.PutUint32(buffer[0x302c:], 0xbff)
		binary.LittleEndian.PutUint32(buffer[0x3034:], 0x10000)

		unkData := []byte{
			0x0c, 0xff, 0xff, 0x0a, 0xff, 0x0c, 0x0c, 0x0c, 0x0c, 0x0c,
			0x0d, 0x0d, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff,
		}
		copy(buffer[0x3040:], unkData)

		binary.LittleEndian.PutUint32(buffer[0x3080:], 0x3e00000)
		binary.LittleEndian.PutUint32(buffer[0x3088:], 0x180000)
		binary.LittleEndian.PutUint32(buffer[0x30f0:], 0x102)
	}
	return &Nacp{buffer: buffer}
}

func (n *Nacp) Buffer() []byte {
	return slices.Clone(n.buffer)
}

func encodeWithSize(v string, size int, name string) []byte {
	if utf8.RuneCountInString(v) >= size {
		panic(fmt.Sprintf("%s length must be <= %d bytes, got %d", name, size-1, len(v)))
	}
	buf := make([]byte, size)
	copy(buf, []byte(v))
	return buf
}

func parseU64(v any) uint64 {
	switch val := v.(type) {
	case string:
		if len(val) > 16 {
			panic(fmt.Sprintf("id length must be 16, got %d", len(val)))
		}
		var result uint64
		fmt.Sscanf(val, "%x", &result)
		return result
	case uint64:
		return val
	default:
		panic("invalid type for parseU64")
	}
}

func (n *Nacp) GetTitle() string {
	return string(bytes.TrimRight(n.buffer[:0x200], "\x00"))
}

func (n *Nacp) SetTitle(v string) {
	buf := encodeWithSize(v, 0x200, "title")
	for i := range 12 {
		copy(n.buffer[i*0x300:], buf)
	}
}

func (n *Nacp) GetAuthor() string {
	return string(bytes.TrimRight(n.buffer[0x200:0x300], "\x00"))
}

func (n *Nacp) SetAuthor(v string) {
	buf := encodeWithSize(v, 0x100, "author")
	for i := range 12 {
		copy(n.buffer[i*0x300+0x200:], buf)
	}
}

func (n *Nacp) SetID(v any) {
	val := parseU64(v)
	// PresenceGroupId
	binary.LittleEndian.PutUint64(n.buffer[0x3038:], val)
	// SaveDataOwnerId
	binary.LittleEndian.PutUint64(n.buffer[0x3078:], val)
	// AddOnContentBaseId (dlcbase)
	binary.LittleEndian.PutUint64(n.buffer[0x3070:], val+0x1000)
	// LocalCommunicationId
	for x := range 8 {
		binary.LittleEndian.PutUint64(n.buffer[0x30b0+x*8:], val)
	}
}

func (n *Nacp) GetID() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3038:])
}

/**
 * Whether or not to display the user account picker
 * when booting up the application.
 */

func (n *Nacp) SetStartupUserAccount(v uint8) {
	n.buffer[0x3025] = v
}
func (n *Nacp) GetStartupUserAccount() uint8 {
	return n.buffer[0x3025]
}

func (n *Nacp) SetUserAccountSwitchLock(v uint8) {
	n.buffer[0x3026] = v
}
func (n *Nacp) GetUserAccountSwitchLock() uint8 {
	return n.buffer[0x3026]
}

func (n *Nacp) SetAddOnContentRegistrationType(v uint8) {
	n.buffer[0x3027] = v
}
func (n *Nacp) GetAddOnContentRegistrationType() uint8 {
	return n.buffer[0x3027]
}

func (n *Nacp) SetAttributeFlag(v uint32) {
	binary.LittleEndian.PutUint32(n.buffer[0x3028:], v)
}
func (n *Nacp) GetAttributeFlag() uint32 {
	return binary.LittleEndian.Uint32(n.buffer[0x3028:])
}

func (n *Nacp) SetSupportedLanguageFlag(v uint32) {
	binary.LittleEndian.PutUint32(n.buffer[0x302c:], v)
}
func (n *Nacp) GetSupportedLanguageFlag() uint32 {
	return binary.LittleEndian.Uint32(n.buffer[0x302c:])
}

func (n *Nacp) SetParentalControlFlag(v uint32) {
	binary.LittleEndian.PutUint32(n.buffer[0x3030:], v)
}
func (n *Nacp) GetParentalControlFlag() uint32 {
	return binary.LittleEndian.Uint32(n.buffer[0x3030:])
}

/**
 * Whether or not pressing the screenshot button will capture a screenshot.
 *   - Value of `0`: Enabled
 *   - Value of `1`: Disabled
 */
func (n *Nacp) SetScreenshot(v uint8) {
	n.buffer[0x3034] = v
}
func (n *Nacp) GetScreenshot() uint8 {
	return n.buffer[0x3034]
}

/**
 * Whether or not holding the screenshot button will capture a video.
 *   - Value of `0`: Disabled
 *   - Value of `1`: Only enabled if app invokes `appletInitializeGamePlayRecording()`
 *   - Value of `2`: Always enabled
 */
func (n *Nacp) SetVideoCapture(v uint8) {
	n.buffer[0x3035] = v
}
func (n *Nacp) GetVideoCapture() uint8 {
	return n.buffer[0x3035]
}

func (n *Nacp) SetDataLossConfirmation(v uint8) {
	n.buffer[0x3036] = v
}
func (n *Nacp) GetDataLlossConfirmation() uint8 {
	return n.buffer[0x3036]
}

func (n *Nacp) SetPlayLogPolicy(v uint8) {
	n.buffer[0x3037] = v
}
func (n *Nacp) GetPlayLogPolicy() uint8 {
	return n.buffer[0x3037]
}

func (n *Nacp) SetPresenceGroupId(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3038:], parseU64(v))
}
func (n *Nacp) GetPresenceGroupId() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3038:])
}

func (n *Nacp) GetVersion() string {
	return string(bytes.TrimRight(n.buffer[0x3060:0x3070], "\x00"))
}
func (n *Nacp) SetVersion(v string) {
	buf := encodeWithSize(v, 0x10, "version")
	copy(n.buffer[0x3060:], buf)
}

func (n *Nacp) SetAddOnContentBaseId(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3070:], parseU64(v))
}
func (n *Nacp) GetAddOnContentBaseId() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3070:])
}

func (n *Nacp) SetSaveDataOwnerId(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3078:], parseU64(v))
}
func (n *Nacp) GetSaveDataOwnerId() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3078:])
}

func (n *Nacp) SetUserAccountSaveDataSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3080:], parseU64(v))
}
func (n *Nacp) GetUserAccountSaveDataSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3080:])
}

func (n *Nacp) SetUserAccountSaveDataJournalSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3088:], parseU64(v))
}
func (n *Nacp) GetUserAccountSaveDataJournalSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3088:])
}

func (n *Nacp) SetDeviceSaveDataSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3090:], parseU64(v))
}
func (n *Nacp) GetDeviceSaveDataSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3090:])
}

func (n *Nacp) SetDeviceSaveDataJournalSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3098:], parseU64(v))
}
func (n *Nacp) GetDeviceSaveDataJournalSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3098:])
}

func (n *Nacp) SetBcatDeliveryCacheStorageSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x30a0:], parseU64(v))
}
func (n *Nacp) GetBcatDeliveryCacheStorageSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x30a0:])
}

func (n *Nacp) SetApplicationErrorCodeCategory(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x30a8:], parseU64(v))
}
func (n *Nacp) GetApplicationErrorCodeCategory() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x30a8:])
}

/**
 * Text shown above logo during boot-up.
 *   - Value of 0: "Licensed by"
 *   - Value of 1: "Distributed by"
 *   - Anything else: no text shown
 */
func (n *Nacp) SetLogoType(v uint8) {
	n.buffer[0x30f0] = v
}
func (n *Nacp) GetLogoType() uint8 {
	return n.buffer[0x30f0]
}

func (n *Nacp) SetLogoHandling(v uint8) {
	n.buffer[0x30f1] = v
}
func (n *Nacp) GetLogoHandling() uint8 {
	return n.buffer[0x30f1]
}

func (n *Nacp) SetRuntimeAddOnContentInstall(v uint8) {
	n.buffer[0x30f2] = v
}
func (n *Nacp) GetRuntimeAddOnContentInstall() uint8 {
	return n.buffer[0x30f2]
}

func (n *Nacp) SetRuntimeParameterDelivery(v uint8) {
	n.buffer[0x30f3] = v
}
func (n *Nacp) GetRuntimeParameterDelivery() uint8 {
	return n.buffer[0x30f3]
}

func (n *Nacp) SetCrashReport(v uint8) {
	n.buffer[0x30f6] = v
}
func (n *Nacp) GetCrashReport() uint8 {
	return n.buffer[0x30f6]
}

func (n *Nacp) SetHdcp(v uint8) {
	n.buffer[0x30f7] = v
}
func (n *Nacp) GetHdcp() uint8 {
	return n.buffer[0x30f7]
}

func (n *Nacp) SetPseudoDeviceIdSeed(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x30a8:], parseU64(v))
}
func (n *Nacp) GetPseudoDeviceIdSeed() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x30a8:])
}

func (n *Nacp) SetStartupUserAccountOption(v uint8) {
	n.buffer[0x3141] = v
}
func (n *Nacp) GetStartupUserAccountOption() uint8 {
	return n.buffer[0x3141]
}

func (n *Nacp) SetUserAccountSaveDataSizeMax(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3148:], parseU64(v))
}
func (n *Nacp) GetUserAccountSaveDataSizeMax() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3148:])
}

func (n *Nacp) SetUserAccountSaveDataJournalSizeMax(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3150:], parseU64(v))
}
func (n *Nacp) GetUserAccountSaveDataJournalSizeMax() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3150:])
}

func (n *Nacp) SetDeviceSaveDataSizeMax(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3158:], parseU64(v))
}
func (n *Nacp) GetDeviceSaveDataSizeMax() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3158:])
}

func (n *Nacp) SetDeviceSaveDataJournalSizeMax(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3160:], parseU64(v))
}
func (n *Nacp) GetDeviceSaveDataJournalSizeMax() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3160:])
}

func (n *Nacp) SetTemporaryStorageSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3168:], parseU64(v))
}
func (n *Nacp) GetTemporaryStorageSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3168:])
}

func (n *Nacp) SetCacheStorageSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3170:], parseU64(v))
}
func (n *Nacp) GetCacheStorageSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3170:])
}

func (n *Nacp) SetCacheStorageJournalSize(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3178:], parseU64(v))
}
func (n *Nacp) GetCacheStorageJournalSize() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3178:])
}

func (n *Nacp) SetCacheStorageDataAndJournalSizeMax(v any) {
	binary.LittleEndian.PutUint64(n.buffer[0x3180:], parseU64(v))
}
func (n *Nacp) GetCacheStorageDataAndJournalSizeMax() uint64 {
	return binary.LittleEndian.Uint64(n.buffer[0x3180:])
}

func (n *Nacp) SetCacheStorageIndexMax(v uint16) {
	binary.LittleEndian.PutUint16(n.buffer[0x3188:], v)
}
func (n *Nacp) GetCacheStorageIndexMax() uint16 {
	return binary.LittleEndian.Uint16(n.buffer[0x3188:])
}

func (n *Nacp) SetPlayLogQueryCapability(v uint8) {
	n.buffer[0x3210] = v
}
func (n *Nacp) GetPlayLogQueryCapability() uint8 {
	return n.buffer[0x3210]
}

func (n *Nacp) SetRepairFlag(v uint8) {
	n.buffer[0x3211] = v
}
func (n *Nacp) GetRepairFlag() uint8 {
	return n.buffer[0x3211]
}

func (n *Nacp) SetProgramIndex(v uint8) {
	n.buffer[0x3212] = v
}
func (n *Nacp) GetProgramIndex() uint8 {
	return n.buffer[0x3212]
}

func (n *Nacp) SetRequiredNetworkServiceLicenseOnLaunch(v uint8) {
	n.buffer[0x3213] = v
}
func (n *Nacp) GetRequiredNetworkServiceLicenseOnLaunch() uint8 {
	return n.buffer[0x3213]
}

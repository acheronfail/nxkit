package keys

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
)

type Keys struct {
	AesKekGenerationSource       []byte   `name:"aes_kek_generation_source"`
	AesKeyGenerationSource       []byte   `name:"aes_key_generation_source"`
	BisKekSource                 []byte   `name:"bis_kek_source"`
	BisKey                       [][]byte `prefix:"bis_key_"`
	BisKeySource                 [][]byte `prefix:"bis_key_source_"`
	DeviceKey                    []byte   `name:"device_key"`
	DeviceKey4X                  []byte   `name:"device_key_4x"`
	EticketRsaKek                []byte   `name:"eticket_rsa_kek"`
	EticketRsaKekSource          []byte   `name:"eticket_rsa_kek_source"`
	EticketRsaKekekSource        []byte   `name:"eticket_rsa_kekek_source"`
	EticketRsaKeypair            []byte   `name:"eticket_rsa_keypair"`
	HeaderKekSource              []byte   `name:"header_kek_source"`
	HeaderKey                    []byte   `name:"header_key"`
	HeaderKeySource              []byte   `name:"header_key_source"`
	KeyAreaKeyApplication        [][]byte `prefix:"key_area_key_application_"`
	KeyAreaKeyApplicationSource  []byte   `name:"key_area_key_application_source"`
	KeyAreaKeyOcean              [][]byte `prefix:"key_area_key_ocean_"`
	KeyAreaKeyOceanSource        []byte   `name:"key_area_key_ocean_source"`
	KeyAreaKeySystem             [][]byte `prefix:"key_area_key_system_"`
	KeyAreaKeySystemSource       []byte   `name:"key_area_key_system_source"`
	Keyblob                      [][]byte `prefix:"keyblob_"`
	KeyblobKey                   [][]byte `prefix:"keyblob_key_"`
	KeyblobKeySource             [][]byte `prefix:"keyblob_key_source_"`
	KeyblobMacKey                [][]byte `prefix:"keyblob_mac_key_"`
	KeyblobMacKeySource          []byte   `name:"keyblob_mac_key_source"`
	MarikoMasterKekSource        [][]byte `prefix:"mariko_master_kek_source_"` // NOTE starts at 5?
	MasterKek                    [][]byte `prefix:"master_kek_"`
	MasterKekSource              [][]byte `prefix:"master_kek_source_"` // NOTE starts at 6?
	MasterKey                    [][]byte `prefix:"master_key_"`
	MasterKeySource              []byte   `name:"master_key_source"`
	Package1Key                  [][]byte `prefix:"package1_key_"`
	Package2Key                  [][]byte `prefix:"package2_key_"`
	Package2KeySource            []byte   `name:"package2_key_source"`
	PerConsoleKeySource          []byte   `name:"per_console_key_source"`
	RetailSpecificAesKeySource   []byte   `name:"retail_specific_aes_key_source"`
	SaveMacKekSource             []byte   `name:"save_mac_kek_source"`
	SaveMacKey                   []byte   `name:"save_mac_key"`
	SaveMacKeySource             []byte   `name:"save_mac_key_source"`
	SaveMacSdCardKekSource       []byte   `name:"save_mac_sd_card_kek_source"`
	SaveMacSdCardKeySource       []byte   `name:"save_mac_sd_card_key_source"`
	SdCardCustomStorageKeySource []byte   `name:"sd_card_custom_storage_key_source"`
	SdCardKekSource              []byte   `name:"sd_card_kek_source"`
	SdCardNcaKeySource           []byte   `name:"sd_card_nca_key_source"`
	SdCardSaveKeySource          []byte   `name:"sd_card_save_key_source"`
	SdSeed                       []byte   `name:"sd_seed"`
	SecureBootKey                []byte   `name:"secure_boot_key"`
	SslRsaKek                    []byte   `name:"ssl_rsa_kek"`
	SslRsaKekSource              []byte   `name:"ssl_rsa_kek_source"`
	SslRsaKekekSource            []byte   `name:"ssl_rsa_kekek_source"`
	SslRsaKey                    []byte   `name:"ssl_rsa_key"`
	Titlekek                     [][]byte `prefix:"titlekek_"`
	TitlekekSource               []byte   `name:"titlekek_source"`
	TsecKey                      []byte   `name:"tsec_key"`
	TsecRootKey02                []byte   `name:"tsec_root_key_02"`
}

func (keys *Keys) GetKeyAreaKey(cryptoType, kakIndex int) ([]byte, error) {
	switch cryptoType {
	case 0:
		return keys.KeyAreaKeyApplication[kakIndex], nil
	case 1:
		return keys.KeyAreaKeyOcean[kakIndex], nil
	case 2:
		return keys.KeyAreaKeySystem[kakIndex], nil
	}

	return nil, fmt.Errorf("failed to find key area key for cryptoType=%d, index=%d", cryptoType, kakIndex)
}

func NewFromPath(path string) (*Keys, error) {
	m, err := parseFileToMap(path)
	if err != nil {
		return nil, err
	}

	return mapToKeys(m)
}

func mapToKeys(m map[string][]byte) (*Keys, error) {
	keys := &Keys{}
	v := reflect.ValueOf(keys).Elem()
	t := v.Type()

	for i := range v.NumField() {
		field := t.Field(i)
		name := field.Tag.Get("name")
		prefix := field.Tag.Get("prefix")

		// single keys by name
		if name != "" {
			if val, ok := m[name]; ok {
				v.Field(i).SetBytes(val)
			} else {
				return nil, fmt.Errorf("failed to find %s in keys", field.Name)
			}
		} else if prefix != "" {
			// collect matchingKeys keys by prefix
			matchingKeys := make([]string, 0)
			for key := range m {
				if strings.HasPrefix(key, prefix) && len(key)-len(prefix) == 2 {
					matchingKeys = append(matchingKeys, key)
				}
			}

			if len(matchingKeys) == 0 {
				return nil, fmt.Errorf("failed to find keys starting with %s", prefix)
			}

			// sort keys to maintain order
			slices.Sort(matchingKeys)
			values := make([][]byte, len(matchingKeys))
			for i, key := range matchingKeys {
				values[i] = m[key]
			}

			v.Field(i).Set(reflect.ValueOf(values))
		} else {
			return nil, fmt.Errorf("field %s must be given a 'name' or 'prefix' tag", field.Name)
		}
	}

	return keys, nil
}

func parseFileToMap(path string) (map[string][]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	m := make(map[string][]byte)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("unrecognised line: %s\n", line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		valBytes, err := hex.DecodeString(val)
		if err != nil {
			fmt.Printf("failed to decode value: %s\n", val)
			continue
		}
		m[key] = valBytes
	}

	return m, scanner.Err()
}

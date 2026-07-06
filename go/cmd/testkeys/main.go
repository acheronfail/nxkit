package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type keySpec struct {
	name    string
	hexSize int
}

func main() {
	paths := os.Args[1:]
	if len(paths) == 0 {
		paths = []string{filepath.Join(".data", "prod.keys")}
	}

	data := []byte(strings.Join(lines(), "\n") + "\n")
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fail("failed to create parent directory for %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fail("failed to write %s: %v", path, err)
		}
	}
}

func lines() []string {
	var specs []keySpec
	for _, name := range singleKeyNames {
		specs = append(specs, keySpec{name: name, hexSize: hexSize(name)})
	}
	for prefix, indexes := range prefixedKeyIndexes {
		for _, index := range indexes {
			name := fmt.Sprintf("%s%02x", prefix, index)
			specs = append(specs, keySpec{name: name, hexSize: hexSize(name)})
		}
	}

	sort.Slice(specs, func(i, j int) bool {
		return specs[i].name < specs[j].name
	})

	lines := make([]string, 0, len(specs))
	for _, spec := range specs {
		lines = append(lines, fmt.Sprintf("%s = %s", spec.name, strings.Repeat("0", spec.hexSize)))
	}
	return lines
}

func hexSize(name string) int {
	switch {
	case name == "eticket_rsa_keypair":
		return 1056
	case name == "ssl_rsa_key":
		return 512
	case strings.HasPrefix(name, "keyblob_") && !strings.HasPrefix(name, "keyblob_key_") && !strings.HasPrefix(name, "keyblob_mac_key_"):
		return 288
	case strings.HasPrefix(name, "bis_key_"),
		strings.HasPrefix(name, "bis_key_source_"),
		name == "header_key",
		name == "header_key_source",
		name == "sd_card_custom_storage_key_source",
		name == "sd_card_nca_key_source",
		name == "sd_card_save_key_source":
		return 64
	default:
		return 32
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

var singleKeyNames = []string{
	"aes_kek_generation_source",
	"aes_key_generation_source",
	"bis_kek_source",
	"device_key",
	"device_key_4x",
	"eticket_rsa_kek",
	"eticket_rsa_kek_source",
	"eticket_rsa_kekek_source",
	"eticket_rsa_keypair",
	"header_kek_source",
	"header_key",
	"header_key_source",
	"key_area_key_application_source",
	"key_area_key_ocean_source",
	"key_area_key_system_source",
	"keyblob_mac_key_source",
	"master_key_source",
	"package2_key_source",
	"per_console_key_source",
	"retail_specific_aes_key_source",
	"save_mac_kek_source",
	"save_mac_key",
	"save_mac_key_source",
	"save_mac_sd_card_kek_source",
	"save_mac_sd_card_key_source",
	"sd_card_custom_storage_key_source",
	"sd_card_kek_source",
	"sd_card_nca_key_source",
	"sd_card_save_key_source",
	"sd_seed",
	"secure_boot_key",
	"ssl_rsa_kek",
	"ssl_rsa_kek_source",
	"ssl_rsa_kekek_source",
	"ssl_rsa_key",
	"titlekek_source",
	"tsec_key",
	"tsec_root_key_02",
}

var prefixedKeyIndexes = map[string][]int{
	"bis_key_":                  rangeFromTo(0, 3),
	"bis_key_source_":           rangeFromTo(0, 2),
	"key_area_key_application_": rangeFromTo(0, 0x11),
	"key_area_key_ocean_":       rangeFromTo(0, 0x11),
	"key_area_key_system_":      rangeFromTo(0, 0x11),
	"keyblob_":                  rangeFromTo(0, 5),
	"keyblob_key_":              rangeFromTo(0, 5),
	"keyblob_key_source_":       rangeFromTo(0, 5),
	"keyblob_mac_key_":          rangeFromTo(0, 5),
	"mariko_master_kek_source_": rangeFromTo(5, 0x11),
	"master_kek_":               rangeFromTo(0, 0x11),
	"master_kek_source_":        rangeFromTo(6, 0x11),
	"master_key_":               rangeFromTo(0, 0x11),
	"package1_key_":             rangeFromTo(0, 5),
	"package2_key_":             rangeFromTo(0, 0x11),
	"titlekek_":                 rangeFromTo(0, 0x11),
}

func rangeFromTo(start, end int) []int {
	values := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		values = append(values, i)
	}
	return values
}

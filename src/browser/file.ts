export function downloadFile(file: File) {
  const url = window.URL.createObjectURL(file);
  const a = document.createElement('a');
  a.href = url;
  a.download = file.name; // Set the file name
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}

export async function readFile(file: File): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = reject;
    reader.onload = (event) => resolve(new Uint8Array(event.target.result as ArrayBuffer));

    reader.readAsArrayBuffer(file);
  });
}

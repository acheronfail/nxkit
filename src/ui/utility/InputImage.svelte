<script lang="ts" context="module">
  export interface Image {
    toBlob: () => Promise<Blob>;
  }

  export interface Props {
    onCropComplete?: (image: Image | null) => void;
  }
</script>

<script lang="ts">
  import ImageCropper from 'svelte-easy-crop';
  import type { CropArea } from 'svelte-easy-crop/dist/types';
  import { extractIcon, isNRO } from '@tootallnate/nro';
  import Container from './Container.svelte';
  import Button from './Button.svelte';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let imageDataUrl = $state<string | null>(null);
  let { onCropComplete }: Props = $props();

  $effect(() => {
    const image = files?.[0];
    if (image) {
      handleImageChange(image);
    }
  });

  async function handleImageChange(file: File) {
    if (await isNRO(file)) {
      const icon = await extractIcon(file);
      if (icon) {
        imageDataUrl = URL.createObjectURL(new Blob([icon], { type: 'image/jpeg' }));
      } else {
        reset();
      }
    } else {
      imageDataUrl = URL.createObjectURL(file);
    }
  }

  // TODO: svelte-easy-crop logs heaps of errors on zoom

  function createImageElement(src: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
      const image = new Image();
      image.addEventListener('load', () => resolve(image));
      image.addEventListener('error', (error) => reject(error));
      image.src = src;
    });
  }

  async function extractCroppedImage(crop: CropArea): Promise<Blob> {
    if (!imageDataUrl) throw new Error('Cannot extract cropped image without an image!');

    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = 256;

    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('Failed to get canvas 2d context');

    ctx.imageSmoothingEnabled = true;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const image = await createImageElement(imageDataUrl);
    ctx.drawImage(image, crop.x, crop.y, crop.width, crop.height, 0, 0, canvas.width, canvas.height);

    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob(
        (blob) => {
          if (blob) {
            resolve(blob);
          } else {
            reject(new Error('An unknown error occurred when producing image'));
          }
        },
        `image/jpeg`,
        0.95,
      );
    });
  }

  function reset() {
    if (input) input.value = '';
    files = null;
    imageDataUrl = null;
    onCropComplete?.(null);
  }

  const imageBoxClass = 'border dark:border-slate-600 dark:bg-slate-900';
</script>

<Container>
  <div class="m-auto">
    {#if imageDataUrl}
      <div class="nro256 {imageBoxClass} relative flex flex-col justify-center align-center">
        <ImageCropper
          image={imageDataUrl}
          aspect={1}
          showGrid={false}
          on:cropcomplete={(event) =>
            onCropComplete?.(imageDataUrl ? { toBlob: () => extractCroppedImage(event.detail.pixels) } : null)}
        />
      </div>
    {:else}
      <label
        for="image-input"
        class="nro256 {imageBoxClass} border-dashed h-full flex flex-col p-6 text-center justify-center items-center cursor-pointer hover:dark:bg-slate-700"
      >
        Please select an NRO file or an image
      </label>
    {/if}
    <div class="flex justify-center items-center p-2">
      <Button size="inline" disabled={!imageDataUrl} onclick={reset}>reset</Button>
    </div>
  </div>

  <input hidden type="file" accept="image/*,.nro" id="image-input" bind:this={input} bind:files />
</Container>

<style>
  .nro256 {
    height: 256px;
    width: 256px;
  }
</style>

<script lang="ts" context="module">
  import type { Snippet } from 'svelte';

  export interface Props {}

  interface ItemProps {
    id: string;
    title: string;
    class?: string;
    snippet: Snippet;
  }
</script>

<script lang="ts">
  import InputFile from './utility/InputFile.svelte';
  import Container from './utility/Container.svelte';
  import Code from './utility/Code.svelte';
  import Button from './utility/Button.svelte';
  import Tooltip from './utility/Tooltip.svelte';
  import InputCheckbox from './utility/InputCheckbox.svelte';
  import Markdown from './utility/Markdown.svelte';
  import splitAsFileMd from './markdown/split-as-archive.md?raw';
  import splitAsDirMd from './markdown/split-as-file.md?raw';
  import type { SplitMergeResult } from '../node/split';

  let merging = $state(false);
  let mergeSelected = $state(false);
  let mergeFileList = $state<FileList | null>(null);
  let mergeCopy = $state(true);

  let splitting = $state(false);
  let splitSelected = $state(false);
  let splitFileList = $state<FileList | null>(null);
  let splitAsArchive = $state(true);
  let splitCopy = $state(true);

  function handleSplitMergeResult(result: SplitMergeResult) {
    if (result.type === 'failure') {
      alert(result.description);
    } else {
      window.nxkit.call('openPath', result.outputPath);
    }
  }
</script>

{#snippet item({ id, title, class: propClass, snippet }: ItemProps)}
  <div data-testid={id} class="flex flex-col gap-2">
    <h2 class="font-bold">{title}</h2>
    <div class={propClass}>{@render snippet()}</div>
  </div>
{/snippet}

{#snippet splitter()}
  <p>
    The max size a file can be on FAT32 filesystems is 4 GB. Use this tool to split files into chunks so they can be
    copied to and from FAT32 filesystems.
  </p>
  <InputFile label="File to split" disabled={splitting} bind:selected={splitSelected} bind:files={splitFileList} />
  <div class="flex gap-2 items-center">
    <Tooltip class="grow" disabled={splitSelected}>
      <Button
        class="block w-full"
        appearance="primary"
        disabled={!splitSelected || splitting}
        onclick={() => {
          if (splitFileList) {
            splitting = true;
            window.nxkit
              .call('splitFile', splitFileList[0].path, splitAsArchive, !splitCopy)
              .then(handleSplitMergeResult)
              .finally(() => (splitting = false));
          }
        }}
      >
        {#if splitting}
          Splitting file...
        {:else}
          Split!
        {/if}
      </Button>
      {#snippet tooltip()}
        Select a file first
      {/snippet}
    </Tooltip>
    <Tooltip>
      <InputCheckbox id="splitAsArchive" label="Split as archive" disabled={splitting} bind:checked={splitAsArchive} />
      {#snippet tooltip()}
        <Markdown class="w-96" content={splitAsArchive ? splitAsFileMd : splitAsDirMd} />
      {/snippet}
    </Tooltip>
    <Tooltip>
      <InputCheckbox id="splitCopy" label="Make Copy" disabled={splitting} bind:checked={splitCopy} />
      {#snippet tooltip()}
        <Markdown
          class="w-48 text-center"
          content={splitCopy
            ? 'Creates a new split file from the original.'
            : 'Splits the file without creating a copy, requires less free disk space but uses up the original file.'}
        />
      {/snippet}
    </Tooltip>
  </div>
{/snippet}

{#snippet merger()}
  <p>With this tool you can merge split files into a single file (this does the reverse of the file splitter).</p>
  <Tooltip>
    <InputFile label="File to merge" disabled={merging} bind:selected={mergeSelected} bind:files={mergeFileList} />
    {#snippet tooltip()}
      <p>Select the first part, e.g.: <Code>rawnand.bin.00</Code> or <Code>my_file.nsp/00</Code></p>
    {/snippet}
  </Tooltip>
  <div class="flex gap-2 items-center">
    <Tooltip class="grow" disabled={mergeSelected}>
      <Button
        class="block w-full"
        appearance="primary"
        disabled={!mergeSelected || merging}
        onclick={() => {
          if (mergeFileList) {
            merging = true;
            window.nxkit
              .call('mergeFile', mergeFileList[0].path, !mergeCopy)
              .then(handleSplitMergeResult)
              .finally(() => (merging = false));
          }
        }}
      >
        {#if merging}
          Merging file...
        {:else}
          Merge!
        {/if}
      </Button>
      {#snippet tooltip()}
        Select a file first
      {/snippet}
    </Tooltip>
    <Tooltip>
      <InputCheckbox id="mergeCopy" label="Make Copy" disabled={merging} bind:checked={mergeCopy} />
      {#snippet tooltip()}
        <Markdown
          class="w-48 text-center"
          content={mergeCopy
            ? 'Creates a new merged file from the original parts.'
            : 'Merges the file and deletes the split parts during the process, requires less free disk space.'}
        />
      {/snippet}
    </Tooltip>
  </div>
{/snippet}

<Container data-testid="tools">
  {@render item({ id: 'splitter', title: 'File Splitter', class: 'flex flex-col gap-1', snippet: splitter })}
  <hr class="border-slate-500" />
  {@render item({ id: 'merger', title: 'File Merger', class: 'flex flex-col gap-1', snippet: merger })}
</Container>

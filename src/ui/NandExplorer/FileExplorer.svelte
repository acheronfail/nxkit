<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../../nand/fatfs/fs';
  import type { Node } from '../utility/FileTreeNode.svelte';

  function entryToNode(entry: FSEntry): Node<FSDirectory, FSFile> {
    return {
      id: entry.path,
      name: entry.name,
      isDirectory: entry.type === 'd',
      data: entry,
    };
  }

  export interface Props {
    rootEntries: FSEntry[];
  }
</script>

<script lang="ts">
  import { Tooltip } from '@svelte-plugins/tooltips';

  import { ArrowDownTrayIcon, EllipsisHorizontalCircleIcon } from 'heroicons-svelte/24/outline';
  import FileTreeRoot from '../utility/FileTreeRoot.svelte';
  import ActionButton from './ActionButton.svelte';
  import ActionButtons from './ActionButtons.svelte';

  let { rootEntries }: Props = $props();

  const handlers = {
    onDirActions: (dir: FSDirectory) => {
      alert('TODO dir actions');
      console.log(dir);
    },
    onFileActions: (file: FSFile) => {
      alert('TODO file actions');
      console.log(file);
    },
    downloadFile: async (file: FSFile) => {
      await window.nxkit.nandCopyFile(file.path);
    },
    openNandDirectory: async (dir: FSDirectory): Promise<Node<FSDirectory, FSFile>[]> => {
      return window.nxkit.nandReaddir(dir.path).then((entries) => entries.map(entryToNode));
    },
  };
</script>

<FileTreeRoot nodes={rootEntries.map(entryToNode)} openDirectory={handlers.openNandDirectory}>
  <ActionButtons slot="dir-extra" let:dir>
    <ActionButton onclick={() => handlers.onDirActions(dir)}>
      <EllipsisHorizontalCircleIcon class="h-4 cursor-pointer hover:text-black" />
    </ActionButton>
  </ActionButtons>
  <ActionButtons slot="file-extra" let:file>
    <span class="font-mono">{file.sizeHuman}</span>
    <Tooltip content="Download {file.name}" position="left">
      <ActionButton onclick={() => handlers.downloadFile(file)}>
        <ArrowDownTrayIcon class="h-4 cursor-pointer hover:text-black" />
      </ActionButton>
    </Tooltip>
    <ActionButton onclick={() => handlers.onFileActions(file)}>
      <EllipsisHorizontalCircleIcon class="h-4 cursor-pointer hover:text-black" />
    </ActionButton>
  </ActionButtons>
</FileTreeRoot>

<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../../nand/fatfs/fs';
  import type { Node } from '../utility/FileTree/FileTreeNode.svelte';

  export function entryToNode(entry: FSEntry): Node<FSDirectory, FSFile> {
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
  import Tooltip from '../utility/Tooltip.svelte';

  import { ArrowDownTrayIcon, EllipsisHorizontalCircleIcon } from 'heroicons-svelte/24/outline';
  import FileTreeRoot from '../utility/FileTree/FileTreeRoot.svelte';
  import ActionButton from '../utility/FileTree/ActionButton.svelte';
  import ActionButtons from '../utility/FileTree/ActionButtons.svelte';
  import { NandError } from 'src/channels';

  let { rootEntries }: Props = $props();

  // TODO: actions: rename, delete, download entire directory
  // TODO:  native right click menu?
  // TODO:  copy/paste?
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
      return window.nxkit.nandReaddir(dir.path).then((result) => {
        if (result.error === NandError.None) {
          return result.data.map(entryToNode);
        }

        console.error(`Failed to readdir: ${result.error}`);
        return [];
      });
    },
  };
</script>

<FileTreeRoot nodes={rootEntries.map(entryToNode)} openDirectory={handlers.openNandDirectory}>
  <ActionButtons slot="dir-extra" let:dir>
    <ActionButton onclick={() => handlers.onDirActions(dir)}>
      <EllipsisHorizontalCircleIcon class="h-4 cursor-pointer hover:stroke-slate-900 hover:stroke-2" />
    </ActionButton>
  </ActionButtons>
  <ActionButtons slot="file-extra" let:file>
    <span class="font-mono">{file.sizeHuman}</span>
    <Tooltip placement="left">
      <ActionButton slot="content" onclick={() => handlers.downloadFile(file)}>
        <ArrowDownTrayIcon class="h-4 cursor-pointer hover:stroke-slate-900 hover:stroke-2" />
      </ActionButton>
      <span slot="tooltip">Download {file.name}</span>
    </Tooltip>
    <ActionButton onclick={() => handlers.onFileActions(file)}>
      <EllipsisHorizontalCircleIcon class="h-4 cursor-pointer hover:stroke-slate-900 hover:stroke-2" />
    </ActionButton>
  </ActionButtons>
</FileTreeRoot>

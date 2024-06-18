<script context="module" lang="ts">
  import FileTreeNode, { type Node } from './FileTreeNode.svelte';

  interface CommonProps<FileData = any, DirData = any> {
    nodes: Node<FileData, DirData>[];
    onFileClick?: (data: FileData) => void;
  }

  export type Props<FileData = any, DirData = any> = DirData extends never
    ? CommonProps & { openDirectory?: undefined }
    : CommonProps & { openDirectory: (dir: DirData) => Promise<Node<FileData, DirData>[]> };
</script>

<script lang="ts">
  let { nodes, onFileClick, openDirectory }: Props = $props();
</script>

<ul class="select-none font-mono overflow-hidden m-2 border border-slate-900">
  {#each nodes as node}
    <FileTreeNode {onFileClick} {openDirectory} {node} iconSlotPresent={$$slots.icon} depth={1}>
      <!-- svelte-ignore slot_element_deprecated -->
      <span slot="icon" let:iconClass let:node><slot name="icon" {iconClass} {node} /></span>
      <!-- svelte-ignore slot_element_deprecated -->
      <span slot="dir-extra" let:dir><slot name="dir-extra" {dir} /></span>
      <!-- svelte-ignore slot_element_deprecated -->
      <span slot="file-extra" let:file><slot name="file-extra" {file} /></span>
    </FileTreeNode>
  {/each}
</ul>

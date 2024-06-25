<script context="module" lang="ts">
  import FileTreeNode, { type Snippets, type Node } from './FileTreeNode.svelte';

  interface CommonProps<FileData = any, DirData = any> extends Snippets<FileData, DirData> {
    nodes: Node<FileData, DirData>[];
    onFileClick?: (data: FileData) => void;
    class?: string;
  }

  export type Props<FileData = any, DirData = any> = DirData extends never
    ? CommonProps & { openDirectory?: undefined }
    : CommonProps & {
        openDirectory: (dir: DirData) => Promise<Node<FileData, DirData>[]>;
      };
</script>

<script lang="ts">
  let { nodes, onFileClick, openDirectory, icon, name, dirExtra, fileExtra, class: cls = '' }: Props = $props();
</script>

<ul class="select-none font-mono m-2 border border-slate-900 bg-slate-900 {cls}">
  {#each nodes as node}
    <FileTreeNode {onFileClick} {openDirectory} {name} {node} {icon} {dirExtra} {fileExtra} depth={1} />
  {/each}
</ul>

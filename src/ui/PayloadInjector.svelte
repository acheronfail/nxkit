<script lang="ts">
  import { findRCMDevices, injectPayload } from '../browser/inject';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import { onMount } from 'svelte';
  import type { FSDirectory, FSFile } from '../node/nand/fatfs/fs';
  import FileTree from './utility/FileTree/FileTree.svelte';
  import { entryToNode, type FileNode, ROOT_NODE } from './NandExplorer/FileExplorer.svelte';
  import ActionButtons from './utility/FileTree/ActionButtons.svelte';
  import Tooltip from './utility/Tooltip.svelte';
  import DownloadPayloads from './PayloadInjector/DownloadPayloads.svelte';
  import LogOutput from './utility/LogOutput.svelte';
  import Markdown from './utility/Markdown.svelte';
  import rcmHelpMd from './markdown/rcm-help.md?raw';

  let showHelp = $state(true);
  let output = $state('');
  let payloads = $state<FSFile[]>([]);
  let fileTree = $state<FileTree<FSFile, FSDirectory> | undefined>();

  async function updatePayloads() {
    payloads = await window.nxkit.call('payloadsFind');
    await fileTree?.reloadDir(ROOT_NODE.id);
  }

  onMount(() => {
    updatePayloads();
    window.addEventListener('focus', updatePayloads);
    return () => window.removeEventListener('focus', updatePayloads);
  });

  const handlers = {
    openPayloadDir: () => window.nxkit.call('payloadsOpenDirectory'),
    injectPayload: async (payloadPath: string) => {
      output = '';

      if (window.nxkit.isWindows) {
        const result = await window.nxkit.call('tegraRcmSmash', payloadPath);
        output = result.stdout;
        if (result.stderr) output += ' -- -- -- \n' + result.stderr;
      } else {
        const [dev] = await findRCMDevices();
        if (!dev) return alert('No Switch found in RCM mode!');

        try {
          const payloadBytes = await window.nxkit.call('payloadsReadFile', payloadPath);
          await injectPayload(dev, payloadBytes, (log) => (output += `${log}\n`));
        } catch (err) {
          output += err instanceof Error ? err.stack : String(err);
        }
      }
    },
    onDragDrop: (_: FileNode<true>, fileList: FileNode | FileList) => {
      // only support incoming files
      if (!(fileList instanceof FileList)) return;

      // only copy in paths ending in `.bin`
      const filePaths: string[] = [];
      for (const file of fileList) {
        if (file.path.endsWith('.bin')) {
          filePaths.push(file.path);
        }
      }

      if (filePaths.length !== fileList.length) {
        return alert('You can only copy files ending with ".bin" to the payloads folder!');
      }

      window.nxkit
        .call('payloadsCopyIn', filePaths)
        .catch((err) => {
          console.error(err);
          alert(`An error occurred copying files: ${String(err)}`);
        })
        .finally(() => updatePayloads());
    },
  };
</script>

<Container fillParent data-testid="injector">
  <div class="grow flex flex-col gap-2 h-full">
    {#if payloads?.length}
      <p class="text-center">Choose a payload to inject to a Switch in RCM mode</p>
      <FileTree
        bind:this={fileTree}
        class="overflow-auto"
        root={ROOT_NODE}
        loadDirectory={async () => payloads.map(entryToNode)}
        onDragDrop={handlers.onDragDrop}
      >
        {#snippet fileExtra(node)}
          <ActionButtons>
            <span>{node.data.sizeHuman}</span>
            <Tooltip placement="left">
              {#snippet tooltip()}
                <div>Inject {node.data.name}</div>
              {/snippet}
              <div>
                <Button size="small" appearance="primary" onclick={() => handlers.injectPayload(node.data.path)}>
                  inject
                </Button>
              </div>
            </Tooltip>
          </ActionButtons>
        {/snippet}
      </FileTree>
      <div class:grow={!showHelp} class="flex flex-col gap-1">
        <DownloadPayloads />
        <div class="text-center">
          <Button onclick={handlers.openPayloadDir}>Open Payload Folder</Button>
        </div>
      </div>
    {:else}
      <div class:grow={!showHelp} class="flex flex-col gap-2 justify-center items-center">
        <h3 class="font-bold">No payloads found!</h3>
        <DownloadPayloads />
        <Button appearance="primary" onclick={handlers.openPayloadDir}>Open Payload Folder</Button>
      </div>
    {/if}

    {#if output}
      <LogOutput title="Payload Injection Log" bind:output />
    {/if}

    {#if showHelp}
      <div class="grow">
        <Markdown content={rcmHelpMd} />
      </div>
    {/if}

    <div>
      <Button onclick={() => (showHelp = !showHelp)}>
        {#if showHelp}
          Hide help
        {:else}
          Show help
        {/if}
      </Button>
    </div>
  </div>
</Container>

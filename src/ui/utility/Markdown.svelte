<script lang="ts" context="module">
  export interface Props {
    content: string;
    class?: string;
  }
</script>

<script lang="ts">
  import { lexer } from 'marked';
  import Code from './Code.svelte';

  let { content, class: propClass = '' }: Props = $props();

  // TODO: support images (ensure vite bundles them in, etc)
  // TODO: fix 'any' types here
  const tokens = $derived(lexer(content));

  const codeClass = `font-mono break-words align-text-bottom text-sm border rounded p-1 border-slate-400 dark:border-slate-900 bg-slate-200 dark:bg-slate-700`;
</script>

{#snippet renderText({ token }: any)}
  {#if token.type === 'text'}
    {#if token.tokens}
      {#each token.tokens as t}
        {@render renderText({ token: t })}
      {/each}
    {:else}
      {@html token.text}
    {/if}
  {:else if token.type === 'strong'}
    <strong>{token.text}</strong>
  {:else if token.type === 'em'}
    <em>{token.text}</em>
  {:else if token.type === 'image'}
    <img src={token.href} alt={token.text} />
  {:else if token.type === 'link'}
    <a class="text-blue-400 underline" target="_blank" href={token.href}>{token.text}</a>
  {:else if token.type === 'codespan'}
    <Code>{@html token.text}</Code>
  {:else if token.type === 'list'}
    {@render renderList({ token })}
  {/if}
{/snippet}

{#snippet renderList({ token }: any)}
  {#if token.ordered}
    <ol class="text-left list-decimal pl-4">
      {@render renderListItems({ listItems: token.items })}
    </ol>
  {:else}
    <ul class="text-left list-disc pl-4">
      {@render renderListItems({ listItems: token.items })}
    </ul>
  {/if}
{/snippet}

{#snippet renderListItems({ listItems }: any)}
  {#each listItems as item}
    <li class="m-1">
      {#each item.tokens as t}
        {@render renderText({ token: t })}
      {/each}
    </li>
  {/each}
{/snippet}

{#snippet code({ token }: any)}
  <pre class={codeClass}>{token.text}</pre>
{/snippet}

<div class={propClass}>
  {#each tokens as token}
    {#if token.type === 'heading'}
      {#if token.depth === 1}
        <h1 class="font-bold text-2xl">{token.text}</h1>
      {:else if token.depth === 2}
        <h2 class="font-bold text-xl">{token.text}</h2>
      {:else if token.depth === 3}
        <h3 class="font-bold text-lg">{token.text}</h3>
      {:else if token.depth === 4}
        <h4 class="font-bold text-md">{token.text}</h4>
      {:else if token.depth === 5}
        <h5 class="font-bold text-sm">{token.text}</h5>
      {:else if token.depth === 6}
        <h6 class="font-bold text-xs">{token.text}</h6>
      {/if}
    {:else if token.type === 'space'}
      <br />
    {:else if token.type === 'list'}
      {@render renderList({ token })}
    {:else if token.type === 'paragraph'}
      <p>
        {#each token.tokens ?? [] as t}
          {@render renderText({ token: t })}
        {/each}
      </p>
    {:else if token.type === 'code'}
      {@render code({ token })}
    {:else}
      <span class="bg-red-400 text-black font-bold">UNSUPPORTED MARKDOWN: {token.type}</span>
    {/if}
  {/each}
</div>

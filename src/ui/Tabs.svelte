<script lang="ts">
  import type { Tab } from './Tabs.types';

  export let items: Tab[];
  export let selectedId = items[0]?.id;

  const handleClick = (id: string) => () => (selectedId = id);
</script>

<ul>
  {#each items as item}
    <li class:active={selectedId === item.id}>
      <a on:click={handleClick(item.id)}>{item.displayName}</a>
    </li>
  {/each}
</ul>
{#each items as item}
  {#if selectedId == item.id}
    <div class="box">
      <svelte:component this={item.component} />
    </div>
  {/if}
{/each}

<style>
  ul {
    display: flex;
    flex-wrap: wrap;
    padding-left: 0;
    margin-bottom: 0;
    list-style: none;
    border-bottom: 1px solid #dee2e6;
  }
	li {
		margin-bottom: -1px;
	}

  a {
    border: 1px solid transparent;
    border-top-left-radius: 0.25rem;
    border-top-right-radius: 0.25rem;
    display: block;
    padding: 0.5rem 1rem;
    cursor: pointer;
  }

  a:hover {
    border-color: #e9ecef #e9ecef #dee2e6;
  }

  li.active > a {
    color: #495057;
    background-color: #fff;
    border-color: #dee2e6 #dee2e6 #fff;
  }
</style>
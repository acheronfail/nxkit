<script lang="ts">
  export let label: string;
  export let id = label.toLowerCase();
  export let value: string;
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    type="text"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    bind:value
    {...$$props}
  />
  <!-- TODO: optional (?) with tooltip -->
</div>

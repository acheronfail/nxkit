<script lang="ts" context="module">
  import { type Placement } from '@floating-ui/dom';
  import { type Snippet } from 'svelte';

  export interface Props {
    placement?: Placement;
    disabled?: boolean;
    children: Snippet;
    tooltip?: Snippet;
  }
</script>

<script lang="ts">
  import { computePosition, flip, shift, offset, arrow } from '@floating-ui/dom';
  import { onMount } from 'svelte';

  let { children, tooltip, placement, disabled = false }: Props = $props();

  let referenceEl = $state<HTMLElement | null>(null);
  let tooltipEl = $state<HTMLElement | null>(null);
  let arrowEl = $state<HTMLElement | null>(null);
  let hidden = $state(true);

  const offsetPx = 10;
  const arrowSizePx = 5;

  function update() {
    if (!referenceEl || !tooltipEl || !arrowEl) return;

    computePosition(referenceEl, tooltipEl, {
      placement,
      middleware: [offset(offsetPx), flip(), shift({ padding: 5 }), arrow({ element: arrowEl })],
    }).then(({ x, y, placement, middlewareData }) => {
      if (!referenceEl || !tooltipEl || !arrowEl) return;

      Object.assign(tooltipEl.style, {
        left: `${x}px`,
        top: `${y}px`,
      });

      const placementSide = placement.split('-')[0];
      const staticSide = {
        top: 'bottom',
        right: 'left',
        bottom: 'top',
        left: 'right',
      }[placementSide]!;

      if (middlewareData.arrow) {
        const { x: arrowX, y: arrowY } = middlewareData.arrow;
        const rotation = {
          top: '-45',
          right: '45',
          bottom: '135',
          left: '-135',
        }[placementSide];

        Object.assign(arrowEl.style, {
          left: arrowX != null ? `${arrowX}px` : '',
          top: arrowY != null ? `${arrowY}px` : '',
          right: '',
          bottom: '',
          transform: `rotate(${rotation}deg)`,
          [staticSide]: `${-arrowSizePx - 1}px`,
        });
      }
    });
  }

  function show() {
    if (tooltip && tooltipEl) {
      update();
      tooltipEl.style.display = 'block';
    }
  }

  function hide() {
    if (tooltip && tooltipEl) {
      tooltipEl.style.display = '';
    }
  }

  onMount(() => {
    if (tooltipEl) tooltipEl.style.display = hidden ? '' : 'block';
    if (!hidden) update();
  });

  const c = {
    bg: 'bg-slate-900',
    fill: 'fill-slate-900',
    border: 'border-slate-600',
    stroke: 'stroke-slate-600',
    strokeBg: 'stroke-slate-900',
  };
</script>

<span bind:this={referenceEl} role="tooltip" onmouseenter={show} onmouseleave={hide} onfocus={show} onblur={hide}>
  {@render children()}
</span>

{#if !disabled}
  <div bind:this={tooltipEl} class="hidden absolute w-max text-sm py-1 px-2 rounded border {c.bg} {c.border}">
    {#if tooltip}
      {@render tooltip()}
    {/if}
    <div bind:this={arrowEl} class="absolute" style="width: {arrowSizePx * 2}px;">
      <svg viewBox="0 0 {arrowSizePx} {arrowSizePx}" class="fill-white w-full h-full">
        <polygon
          points="1 1, 1 {arrowSizePx - 1}, {arrowSizePx - 1}, {arrowSizePx - 1}, 1"
          class="{c.strokeBg} {c.fill}"
        />
        <line class="stroke-1 {c.stroke}" x1="0" y1="0" x2="0" y2={arrowSizePx} />
        <line class="stroke-1 {c.stroke}" x1="0" y1={arrowSizePx} x2={arrowSizePx} y2={arrowSizePx} />
      </svg>
    </div>
  </div>
{/if}

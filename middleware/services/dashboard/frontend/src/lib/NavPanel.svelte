<script>
  import { nav } from '../stores/robot.js'

  $: status   = $nav?.status    ?? 'idle'
  $: currX    = $nav?.current_x ?? 0
  $: currY    = $nav?.current_y ?? 0
  $: goalX    = $nav?.goal_x    ?? null
  $: goalY    = $nav?.goal_y    ?? null
  $: distance = $nav?.distance  ?? 0

  $: statusColor = status === 'navigating' ? '#4ecca3' : '#6e7681'

  const MAX_DIST = 28.3 // diagonal of 20x20 grid
  $: distPct = Math.min(distance / MAX_DIST * 100, 100)
</script>

<div class="panel">
  <div class="panel-title">Navigation</div>

  <div class="status-row">
    <span class="dot" style="background: {statusColor}"></span>
    <span class="status-text" style="color: {statusColor}">{status.toUpperCase()}</span>
  </div>

  <div class="divider"></div>

  <div class="coord-group">
    <div class="coord-label">Position</div>
    <div class="coord-values">
      <span>X <b>{currX.toFixed(2)}</b></span>
      <span>Y <b>{currY.toFixed(2)}</b></span>
    </div>
  </div>

  <div class="coord-group">
    <div class="coord-label">Goal</div>
    <div class="coord-values">
      {#if goalX !== null}
        <span>X <b>{goalX.toFixed(2)}</b></span>
        <span>Y <b>{goalY.toFixed(2)}</b></span>
      {:else}
        <span class="none">—</span>
      {/if}
    </div>
  </div>

  <div class="divider"></div>

  <div class="dist-row">
    <span class="dist-label">Distance</span>
    <span class="dist-value">{distance.toFixed(2)} m</span>
  </div>
  <div class="bar-track">
    <div class="bar-fill" style="width: {distPct}%"></div>
  </div>
</div>

<style>
  .panel {
    background: #161b22;
    border-radius: 8px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .panel-title {
    font-weight: 700;
    font-size: 12px;
    color: #8b949e;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .status-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .status-text {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
  }

  .divider { height: 1px; background: #21262d; }

  .coord-group {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .coord-label {
    font-size: 11px;
    color: #6e7681;
    width: 52px;
  }

  .coord-values {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #8b949e;
  }

  .coord-values b {
    color: #e6edf3;
    font-variant-numeric: tabular-nums;
  }

  .none { color: #6e7681; }

  .dist-row {
    display: flex;
    justify-content: space-between;
  }

  .dist-label { font-size: 11px; color: #6e7681; }
  .dist-value { font-size: 12px; color: #e6edf3; font-variant-numeric: tabular-nums; }

  .bar-track {
    height: 4px;
    background: #21262d;
    border-radius: 2px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    background: #4ecca3;
    border-radius: 2px;
    transition: width 0.4s ease;
  }
</style>

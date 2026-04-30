<script>
  import { onMount, onDestroy } from 'svelte'
  import { connected, ws } from './stores/robot.js'
  import MapCanvas   from './lib/MapCanvas.svelte'
  import NeoDMPanel  from './lib/NeoDMPanel.svelte'
  import NavPanel    from './lib/NavPanel.svelte'
  import SensorPanel from './lib/SensorPanel.svelte'
  import EmotionSim  from './lib/EmotionSim.svelte'

  onMount(() => ws.connect())
  onDestroy(() => ws.disconnect())

  let activeView = 'map'
</script>

<div class="layout">
  <header>
    <div class="logo">RobotOS</div>
    <div class="header-center">
      <div class="tabs">
        <button class:active={activeView === 'map'}      on:click={() => activeView = 'map'}>Map</button>
        <button class:active={activeView === 'emotion'}  on:click={() => activeView = 'emotion'}>Emotion Sim</button>
      </div>
    </div>
    <div class="ws-status" class:connected={$connected}>
      <span class="ws-dot"></span>
      {$connected ? 'Connected' : 'Reconnecting…'}
    </div>
  </header>

  <main>
    <aside class="left-col">
      <NeoDMPanel />
      <NavPanel />
    </aside>

    <section class="map-col">
      {#if activeView === 'map'}
        <MapCanvas />
      {:else}
        <EmotionSim />
      {/if}
    </section>
  </main>

  <footer>
    <SensorPanel />
  </footer>
</div>

<style>
  :global(*, *::before, *::after) { box-sizing: border-box; margin: 0; padding: 0; }
  :global(body) {
    background: #0d1117;
    color: #e6edf3;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    height: 100vh;
    overflow: hidden;
  }

  .layout {
    display: grid;
    grid-template-rows: 48px 1fr auto;
    height: 100vh;
    gap: 0;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    background: #161b22;
    border-bottom: 1px solid #21262d;
  }

  .logo {
    font-size: 14px;
    font-weight: 800;
    color: #58a6ff;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .header-center {
    font-size: 13px;
    color: #8b949e;
    letter-spacing: 0.05em;
  }

  .tabs {
    display: flex;
    gap: 4px;
  }

  .tabs button {
    background: none;
    border: 1px solid #30363d;
    border-radius: 6px;
    color: #8b949e;
    cursor: pointer;
    font-size: 12px;
    padding: 4px 14px;
    transition: all 0.15s;
  }

  .tabs button:hover { border-color: #58a6ff; color: #e6edf3; }
  .tabs button.active { background: #1f6feb; border-color: #1f6feb; color: #fff; }

  .ws-status {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #6e7681;
  }

  .ws-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #6e7681;
    transition: background 0.3s;
  }

  .ws-status.connected { color: #4ecca3; }
  .ws-status.connected .ws-dot { background: #4ecca3; }

  main {
    display: grid;
    grid-template-columns: 220px 1fr;
    gap: 12px;
    padding: 12px;
    overflow: hidden;
    min-height: 0;
  }

  .left-col {
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow-y: auto;
  }

  .map-col {
    overflow: hidden;
    min-height: 0;
    display: flex;
    align-items: stretch;
  }

  .map-col > :global(*) {
    flex: 1;
  }

  footer {
    padding: 0 12px 12px;
  }
</style>

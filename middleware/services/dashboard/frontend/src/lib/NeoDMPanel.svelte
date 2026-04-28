<script>
  import { neodm } from '../stores/robot.js'

  const EMOTION_COLORS = {
    HAPPY:    '#4ecca3',
    CURIOUS:  '#58a6ff',
    NEUTRAL:  '#8b949e',
    SLEEPY:   '#6e7681',
    SAD:      '#e05252',
    EXCITED:  '#f0883e',
  }

  $: label   = $neodm?.emotion?.label   ?? '—'
  $: valence = $neodm?.emotion?.valence ?? 0
  $: arousal = $neodm?.emotion?.arousal ?? 0
  $: decision = $neodm?.decision        ?? '—'
  $: loopHz  = $neodm?.loop_hz          ?? 0

  $: emotionColor = EMOTION_COLORS[label] ?? '#8b949e'

  // Map -1..1 → 0..100 for display
  $: valencePct = Math.round((valence + 1) / 2 * 100)
  $: arousalPct = Math.round(arousal * 100)
</script>

<div class="panel">
  <div class="panel-title">NeoDM</div>

  <div class="emotion-badge" style="border-color: {emotionColor}; color: {emotionColor}">
    {label}
  </div>

  <div class="metric-row">
    <span class="metric-label">Valence</span>
    <div class="bar-track">
      <div class="bar-fill" style="width: {valencePct}%; background: {emotionColor}"></div>
    </div>
    <span class="metric-value">{valence.toFixed(2)}</span>
  </div>

  <div class="metric-row">
    <span class="metric-label">Arousal</span>
    <div class="bar-track">
      <div class="bar-fill" style="width: {arousalPct}%; background: #f0883e"></div>
    </div>
    <span class="metric-value">{arousal.toFixed(2)}</span>
  </div>

  <div class="divider"></div>

  <div class="stat-row">
    <span class="stat-label">Decision</span>
    <span class="stat-value decision">{decision}</span>
  </div>

  <div class="stat-row">
    <span class="stat-label">Loop</span>
    <span class="stat-value">{loopHz.toFixed(1)} Hz</span>
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

  .emotion-badge {
    align-self: flex-start;
    border: 2px solid;
    border-radius: 20px;
    padding: 4px 14px;
    font-size: 14px;
    font-weight: 700;
    letter-spacing: 0.05em;
    transition: all 0.3s;
  }

  .metric-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .metric-label {
    font-size: 11px;
    color: #6e7681;
    width: 52px;
    flex-shrink: 0;
  }

  .bar-track {
    flex: 1;
    height: 6px;
    background: #21262d;
    border-radius: 3px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.4s ease;
  }

  .metric-value {
    font-size: 11px;
    color: #8b949e;
    width: 36px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .divider {
    height: 1px;
    background: #21262d;
  }

  .stat-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .stat-label {
    font-size: 11px;
    color: #6e7681;
  }

  .stat-value {
    font-size: 12px;
    color: #e6edf3;
    font-variant-numeric: tabular-nums;
  }

  .stat-value.decision {
    font-weight: 700;
    color: #58a6ff;
  }
</style>

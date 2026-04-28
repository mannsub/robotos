<script>
  import { sensor } from '../stores/robot.js'

  // Keep last 60 samples for sparklines
  const MAX_SAMPLES = 60
  let accelHistory = []
  let gyroHistory  = []

  $: if ($sensor) {
    const imu = $sensor.imu
    accelHistory = [...accelHistory, imu.accel_z].slice(-MAX_SAMPLES)
    gyroHistory  = [...gyroHistory,  imu.gyro_z ].slice(-MAX_SAMPLES)
  }

  $: imu      = $sensor?.imu       ?? { accel_x:0, accel_y:0, accel_z:9.81, gyro_x:0, gyro_y:0, gyro_z:0 }
  $: battery  = $sensor?.battery   ?? { pct:0, voltage:0, is_charging:false }
  $: joints   = $sensor?.joint_state?.position ?? []

  $: battPct     = Math.round((battery.pct ?? 0) * 100)
  $: battColor   = battPct > 50 ? '#4ecca3' : battPct > 20 ? '#f0883e' : '#e05252'
  $: chargingTxt = battery.is_charging ? '⚡' : ''

  function sparkPath(values, w, h) {
    if (values.length < 2) return ''
    const min = Math.min(...values)
    const max = Math.max(...values)
    const range = max - min || 1
    const step = w / (values.length - 1)
    return values.map((v, i) => {
      const x = i * step
      const y = h - ((v - min) / range) * h
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
    }).join(' ')
  }
</script>

<div class="panel">
  <div class="panel-title">Sensors</div>

  <div class="sensor-grid">

    <!-- IMU -->
    <div class="section">
      <div class="section-label">IMU — Accelerometer (m/s²)</div>
      <div class="imu-values">
        <span>X <b>{imu.accel_x.toFixed(2)}</b></span>
        <span>Y <b>{imu.accel_y.toFixed(2)}</b></span>
        <span>Z <b>{imu.accel_z.toFixed(2)}</b></span>
      </div>
      <svg class="spark" viewBox="0 0 200 36">
        <path d={sparkPath(accelHistory, 200, 36)} fill="none" stroke="#58a6ff" stroke-width="1.5"/>
      </svg>
    </div>

    <!-- Gyroscope -->
    <div class="section">
      <div class="section-label">IMU — Gyroscope (rad/s)</div>
      <div class="imu-values">
        <span>X <b>{imu.gyro_x.toFixed(3)}</b></span>
        <span>Y <b>{imu.gyro_y.toFixed(3)}</b></span>
        <span>Z <b>{imu.gyro_z.toFixed(3)}</b></span>
      </div>
      <svg class="spark" viewBox="0 0 200 36">
        <path d={sparkPath(gyroHistory, 200, 36)} fill="none" stroke="#f0883e" stroke-width="1.5"/>
      </svg>
    </div>

    <!-- Battery -->
    <div class="section">
      <div class="section-label">Battery {chargingTxt}</div>
      <div class="battery-row">
        <div class="batt-bar-track">
          <div class="batt-bar-fill" style="width:{battPct}%; background:{battColor}"></div>
        </div>
        <span class="batt-pct" style="color:{battColor}">{battPct}%</span>
        <span class="batt-volt">{battery.voltage?.toFixed(1)}V</span>
      </div>
    </div>

    <!-- Joints -->
    <div class="section">
      <div class="section-label">Joint Positions (rad)</div>
      <div class="joints">
        {#each joints as pos, i}
          <div class="joint-row">
            <span class="joint-label">J{i}</span>
            <div class="joint-track">
              <div class="joint-fill" style="width:{Math.min(Math.abs(pos) / 3.14 * 100, 100)}%"></div>
            </div>
            <span class="joint-val">{pos.toFixed(3)}</span>
          </div>
        {/each}
        {#if joints.length === 0}
          <span class="none">No joint data</span>
        {/if}
      </div>
    </div>

  </div>
</div>

<style>
  .panel {
    background: #161b22;
    border-radius: 8px;
    padding: 16px;
  }

  .panel-title {
    font-weight: 700;
    font-size: 12px;
    color: #8b949e;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    margin-bottom: 12px;
  }

  .sensor-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 16px;
  }

  .section { display: flex; flex-direction: column; gap: 6px; }

  .section-label {
    font-size: 11px;
    color: #6e7681;
  }

  .imu-values {
    display: flex;
    gap: 12px;
    font-size: 12px;
    color: #8b949e;
  }

  .imu-values b {
    color: #e6edf3;
    font-variant-numeric: tabular-nums;
  }

  .spark {
    width: 100%;
    height: 36px;
    display: block;
  }

  .battery-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .batt-bar-track {
    flex: 1;
    height: 8px;
    background: #21262d;
    border-radius: 4px;
    overflow: hidden;
  }

  .batt-bar-fill {
    height: 100%;
    border-radius: 4px;
    transition: width 0.5s ease;
  }

  .batt-pct {
    font-size: 12px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }

  .batt-volt {
    font-size: 11px;
    color: #6e7681;
    font-variant-numeric: tabular-nums;
  }

  .joints { display: flex; flex-direction: column; gap: 4px; }

  .joint-row {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .joint-label {
    font-size: 10px;
    color: #6e7681;
    width: 16px;
  }

  .joint-track {
    flex: 1;
    height: 4px;
    background: #21262d;
    border-radius: 2px;
    overflow: hidden;
  }

  .joint-fill {
    height: 100%;
    background: #8957e5;
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .joint-val {
    font-size: 10px;
    color: #8b949e;
    width: 44px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .none { font-size: 11px; color: #6e7681; }
</style>

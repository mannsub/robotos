<script>
  import { onMount, onDestroy } from 'svelte'
  import { nav, ws } from '../stores/robot.js'

  const GRID_W = 200
  const GRID_H = 200
  const RESOLUTION = 0.1  // metres per cell
  const CANVAS_SIZE = 600

  let canvas
  let ctx
  let mode = 'goal'  // 'goal' | 'obstacle' | 'erase' | 'line' | 'pan'
  let isDragging = false

  // Line mode state
  let lineStart = null   // {cx, cy} in grid coords
  let linePreview = null // {cx, cy} current hover

  // Zoom/pan state
  let viewScale = 1.0
  let viewOffX = 0
  let viewOffY = 0
  let isPanning = false
  let panLast = { x: 0, y: 0 }

  const grid = new Uint8Array(GRID_W * GRID_H)

  let robotX = 0, robotY = 0
  let goalX = null, goalY = null
  let navPath = []  // [[wx, wy], ...]

  const cellPx = CANVAS_SIZE / GRID_W

  function worldToPx(wx, wy) {
    return [wx / RESOLUTION * cellPx, (GRID_H - wy / RESOLUTION) * cellPx]
  }

  function pxToCell(px, py) {
    return [Math.floor(px / cellPx), GRID_H - 1 - Math.floor(py / cellPx)]
  }

  function cellToWorld(cx, cy) {
    return [(cx + 0.5) * RESOLUTION, (cy + 0.5) * RESOLUTION]
  }

  function cellIndex(cx, cy) { return cy * GRID_W + cx }

  function inBounds(cx, cy) {
    return cx >= 0 && cx < GRID_W && cy >= 0 && cy < GRID_H
  }

  // Bresenham's line algorithm — returns all grid cells between two points
  function bresenham(x0, y0, x1, y1) {
    const cells = []
    let dx = Math.abs(x1 - x0), dy = Math.abs(y1 - y0)
    let sx = x0 < x1 ? 1 : -1, sy = y0 < y1 ? 1 : -1
    let err = dx - dy
    let x = x0, y = y0
    while (true) {
      cells.push([x, y])
      if (x === x1 && y === y1) break
      let e2 = 2 * err
      if (e2 > -dy) { err -= dy; x += sx }
      if (e2 < dx)  { err += dx; y += sy }
    }
    return cells
  }

  function applyLineCells(cells, blocked) {
    for (const [cx, cy] of cells) {
      if (!inBounds(cx, cy)) continue
      grid[cellIndex(cx, cy)] = blocked ? 1 : 0
      const [wx, wy] = cellToWorld(cx, cy)
      ws.setObstacle(wx, wy, blocked)
    }
  }

  function draw() {
    if (!ctx) return

    ctx.save()
    ctx.setTransform(1, 0, 0, 1, 0, 0)
    ctx.fillStyle = '#0d1117'
    ctx.fillRect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
    ctx.setTransform(viewScale, 0, 0, viewScale, viewOffX, viewOffY)

    // Minor grid lines (1 m)
    ctx.strokeStyle = '#1e2938'
    ctx.lineWidth = 0.5
    for (let i = 0; i <= GRID_W; i += 10) {
      ctx.beginPath(); ctx.moveTo(i * cellPx, 0); ctx.lineTo(i * cellPx, CANVAS_SIZE); ctx.stroke()
      ctx.beginPath(); ctx.moveTo(0, i * cellPx); ctx.lineTo(CANVAS_SIZE, i * cellPx); ctx.stroke()
    }
    // Major grid lines (5 m)
    ctx.strokeStyle = '#2d3f55'
    ctx.lineWidth = 1
    for (let i = 0; i <= GRID_W; i += 50) {
      ctx.beginPath(); ctx.moveTo(i * cellPx, 0); ctx.lineTo(i * cellPx, CANVAS_SIZE); ctx.stroke()
      ctx.beginPath(); ctx.moveTo(0, i * cellPx); ctx.lineTo(CANVAS_SIZE, i * cellPx); ctx.stroke()
    }

    // Obstacles
    ctx.fillStyle = '#e05252'
    for (let cy = 0; cy < GRID_H; cy++) {
      for (let cx = 0; cx < GRID_W; cx++) {
        if (grid[cellIndex(cx, cy)]) {
          ctx.fillRect(cx * cellPx, (GRID_H - 1 - cy) * cellPx, cellPx, cellPx)
        }
      }
    }

    // Line mode preview
    if (mode === 'line' && lineStart && linePreview) {
      const cells = bresenham(lineStart.cx, lineStart.cy, linePreview.cx, linePreview.cy)
      ctx.fillStyle = 'rgba(224, 82, 82, 0.45)'
      for (const [cx, cy] of cells) {
        if (inBounds(cx, cy)) {
          ctx.fillRect(cx * cellPx, (GRID_H - 1 - cy) * cellPx, cellPx, cellPx)
        }
      }
      // start dot
      ctx.fillStyle = '#f0883e'
      ctx.beginPath()
      ctx.arc(
        lineStart.cx * cellPx + cellPx / 2,
        (GRID_H - 1 - lineStart.cy) * cellPx + cellPx / 2,
        cellPx * 2, 0, Math.PI * 2
      )
      ctx.fill()
    }

    // Planned path
    if (navPath.length > 1) {
      ctx.strokeStyle = 'rgba(78, 204, 163, 0.6)'
      ctx.lineWidth = 1.5
      ctx.beginPath()
      const [px0, py0] = worldToPx(navPath[0][0], navPath[0][1])
      ctx.moveTo(px0, py0)
      for (let i = 1; i < navPath.length; i++) {
        const [pxi, pyi] = worldToPx(navPath[i][0], navPath[i][1])
        ctx.lineTo(pxi, pyi)
      }
      ctx.stroke()
    }

    // Goal marker
    if (goalX !== null) {
      const [gx, gy] = worldToPx(goalX, goalY)
      ctx.strokeStyle = '#4ecca3'
      ctx.lineWidth = 2
      ctx.beginPath()
      ctx.arc(gx, gy, cellPx * 3, 0, Math.PI * 2)
      ctx.stroke()
      ctx.beginPath()
      ctx.moveTo(gx - cellPx * 4, gy); ctx.lineTo(gx + cellPx * 4, gy)
      ctx.moveTo(gx, gy - cellPx * 4); ctx.lineTo(gx, gy + cellPx * 4)
      ctx.stroke()
    }

    // Robot
    const [rx, ry] = worldToPx(robotX, robotY)
    ctx.fillStyle = '#58a6ff'
    ctx.strokeStyle = '#ffffff'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.arc(rx, ry, cellPx * 2.5, 0, Math.PI * 2)
    ctx.fill()
    ctx.stroke()

    ctx.restore()
  }

  const unsubNav = nav.subscribe(data => {
    if (!data) return
    robotX  = data.current_x ?? 0
    robotY  = data.current_y ?? 0
    goalX   = data.status === 'idle' ? null : (data.goal_x ?? null)
    goalY   = data.status === 'idle' ? null : (data.goal_y ?? null)
    navPath = data.path ?? []
    draw()
  })

  function getCell(e) {
    const rect = canvas.getBoundingClientRect()
    const sx = (e.clientX - rect.left) * (CANVAS_SIZE / rect.width)
    const sy = (e.clientY - rect.top)  * (CANVAS_SIZE / rect.height)
    // Inverse transform: screen → logical canvas pixels
    const px = (sx - viewOffX) / viewScale
    const py = (sy - viewOffY) / viewScale
    const [cx, cy] = pxToCell(px, py)
    return { cx, cy, px, py }
  }

  function handleMouseDown(e) {
    if (mode === 'pan' || e.button === 1) {
      e.preventDefault()
      isPanning = true
      panLast = { x: e.clientX, y: e.clientY }
      return
    }

    const { cx, cy, px, py } = getCell(e)

    if (mode === 'line') {
      if (!lineStart) {
        lineStart = { cx, cy }
      } else {
        // Commit line
        const cells = bresenham(lineStart.cx, lineStart.cy, cx, cy)
        applyLineCells(cells, true)
        lineStart = null
        linePreview = null
        draw()
      }
      return
    }

    if (mode === 'goal') return  // goal is handled in handleClick only

    isDragging = true
    applyCell(cx, cy)
  }

  function handleMouseMove(e) {
    if (isPanning) {
      const rect = canvas.getBoundingClientRect()
      const scaleX = CANVAS_SIZE / rect.width
      const scaleY = CANVAS_SIZE / rect.height
      viewOffX += (e.clientX - panLast.x) * scaleX
      viewOffY += (e.clientY - panLast.y) * scaleY
      panLast = { x: e.clientX, y: e.clientY }
      draw()
      return
    }

    const { cx, cy } = getCell(e)

    if (mode === 'line') {
      if (lineStart) {
        linePreview = { cx, cy }
        draw()
      }
      return
    }

    if (isDragging && (mode === 'obstacle' || mode === 'erase') && !isPanning) {
      applyCell(cx, cy)
    }
  }

  function handleMouseUp() { isDragging = false; isPanning = false }
  function handleMouseLeave() { isDragging = false; isPanning = false }

  function handleWheel(e) {
    e.preventDefault()
    if (Math.abs(e.deltaY) < 1) return  // ignore micro-scrolls from trackpad clicks
    const rect = canvas.getBoundingClientRect()
    const sx = (e.clientX - rect.left) * (CANVAS_SIZE / rect.width)
    const sy = (e.clientY - rect.top)  * (CANVAS_SIZE / rect.height)
    const factor = e.deltaY < 0 ? 1.15 : 1 / 1.15
    viewOffX = sx - factor * (sx - viewOffX)
    viewOffY = sy - factor * (sy - viewOffY)
    viewScale = Math.max(0.25, Math.min(8, viewScale * factor))
    draw()
  }

  function fitView() {
    viewScale = 1.0; viewOffX = 0; viewOffY = 0
    draw()
  }

  function handleClick(e) {
    if (mode === 'line') return  // handled in mousedown
    const { cx, cy } = getCell(e)
    if (mode === 'goal') {
      if (!inBounds(cx, cy) || grid[cellIndex(cx, cy)]) return  // ignore wall/out-of-bounds clicks
      const [wx, wy] = cellToWorld(cx, cy)
      ws.setGoal(wx, wy)
    }
  }

  function applyCell(cx, cy) {
    if (!inBounds(cx, cy)) return
    const blocked = mode === 'obstacle'
    grid[cellIndex(cx, cy)] = blocked ? 1 : 0
    const [wx, wy] = cellToWorld(cx, cy)
    ws.setObstacle(wx, wy, blocked)
    draw()
  }

  // Cancel line on Escape
  function handleKeydown(e) {
    if (e.key === 'Escape' && mode === 'line') {
      lineStart = null
      linePreview = null
      draw()
    }
  }

  function clearMap() {
    grid.fill(0)
    lineStart = null
    linePreview = null
    ws.resetMap()
    draw()
  }

  // Recursive Backtracker (DFS) perfect maze generator
  function generateMaze() {
    const MW = 24, MH = 24          // rooms wide × tall
    const ROOM = 7, WALL = 1        // cells per room interior, wall thickness
    const UNIT = ROOM + WALL        // = 8 cells per room+wall unit

    // rWall[y][x]: wall between room(x,y) and room(x+1,y), x in 0..MW-2
    // tWall[y][x]: wall between room(x,y) and room(x,y+1), y in 0..MH-2
    const rWall = Array.from({length: MH}, () => new Array(MW - 1).fill(true))
    const tWall = Array.from({length: MH - 1}, () => new Array(MW).fill(true))

    // DFS with iterative stack to avoid JS stack overflow on large mazes
    const vis = Array.from({length: MH}, () => new Array(MW).fill(false))
    const stack = [[0, 0]]
    vis[0][0] = true
    while (stack.length) {
      const [x, y] = stack[stack.length - 1]
      const dirs = [[1,0],[-1,0],[0,1],[0,-1]].filter(([dx,dy]) => {
        const nx=x+dx, ny=y+dy
        return nx>=0&&nx<MW&&ny>=0&&ny<MH&&!vis[ny][nx]
      })
      if (!dirs.length) { stack.pop(); continue }
      const [dx, dy] = dirs[Math.floor(Math.random() * dirs.length)]
      const nx = x+dx, ny = y+dy
      if (dx === 1)  rWall[y][x]  = false
      if (dx === -1) rWall[y][nx] = false
      if (dy === 1)  tWall[y][x]  = false
      if (dy === -1) tWall[ny][x] = false
      vis[ny][nx] = true
      stack.push([nx, ny])
    }

    clearMap()
    ws.resetRobot()

    const mazeObstacles = []
    function mark(cx, cy) {
      if (!inBounds(cx, cy)) return
      grid[cellIndex(cx, cy)] = 1
      const [wx, wy] = cellToWorld(cx, cy)
      mazeObstacles.push({ x: wx, y: wy, blocked: true })
    }

    // Outer border (top + right; left/bottom = grid boundary)
    for (let cx = 0; cx <= MW*UNIT; cx++) mark(cx, MH*UNIT)
    for (let cy = 0; cy <  MH*UNIT; cy++) mark(MW*UNIT, cy)

    // Interior: corner posts, right walls, top walls
    for (let my = 0; my < MH; my++) {
      for (let mx = 0; mx < MW; mx++) {
        const rx = mx * UNIT   // canvas col of room left edge
        const ry = my * UNIT   // canvas row of room bottom edge

        // Corner post: always at interior intersections; at border edges only when
        // an adjacent wall segment needs a cap cell to close the 1-cell gap.
        const needsPost = (mx < MW-1 && my < MH-1)
          || (my === MH-1 && mx < MW-1 && rWall[my][mx])
          || (mx === MW-1 && my < MH-1 && tWall[my][mx])
        if (needsPost) mark(rx + ROOM, ry + ROOM)

        // Right wall of room(mx, my)
        if (mx < MW-1 && rWall[my][mx])
          for (let cy = ry; cy < ry + ROOM; cy++) mark(rx + ROOM, cy)

        // Top wall of room(mx, my)
        if (my < MH-1 && tWall[my][mx])
          for (let cx = rx; cx < rx + ROOM; cx++) mark(cx, ry + ROOM)
      }
    }

    // Send all obstacles as a single batch message to avoid channel drops
    ws.setMaze(mazeObstacles)

    // Reset local display state immediately without waiting for server response
    robotX = 0; robotY = 0
    goalX = null; goalY = null
    navPath = []

    draw()
  }

  onMount(() => {
    ctx = canvas.getContext('2d')
    draw()
    window.addEventListener('keydown', handleKeydown)
  })

  onDestroy(() => {
    unsubNav()
    window.removeEventListener('keydown', handleKeydown)
  })
</script>

<div class="map-panel">
  <div class="map-header">
    <span class="panel-title">Map</span>
    <div class="toolbar">
      <button class="tool-btn" class:active={mode === 'goal'}     on:click={() => { mode = 'goal'; lineStart = null }}>
        🎯 Goal
      </button>
      <button class="tool-btn" class:active={mode === 'obstacle'} on:click={() => { mode = 'obstacle'; lineStart = null }}>
        ⬛ Obstacle
      </button>
      <button class="tool-btn" class:active={mode === 'line'}     on:click={() => { mode = 'line'; lineStart = null }}>
        📏 Line
      </button>
      <button class="tool-btn" class:active={mode === 'erase'}    on:click={() => { mode = 'erase'; lineStart = null }}>
        ✏️ Erase
      </button>
      <button class="tool-btn" class:active={mode === 'pan'}     on:click={() => { mode = 'pan'; lineStart = null }}>
        🖐️ Pan
      </button>
      <button class="tool-btn tool-sep" on:click={clearMap}>🗑️ Clear</button>
      <button class="tool-btn tool-maze" on:click={generateMaze}>🎲 Maze</button>
      <button class="tool-btn tool-sep" on:click={fitView} title="Reset zoom">⊡</button>
    </div>
    <span class="map-hint">
      Wheel: zoom · 🖐️ Pan mode · ⊡: fit &nbsp;|&nbsp;
      {#if mode === 'goal'}Click to set navigation goal{/if}
      {#if mode === 'obstacle'}Click/drag to add obstacles{/if}
      {#if mode === 'line'}
        {#if lineStart}Click second point to draw wall (Esc to cancel){:else}Click first point to start wall{/if}
      {/if}
      {#if mode === 'erase'}Click/drag to erase obstacles{/if}
    </span>
  </div>

  <canvas
    bind:this={canvas}
    width={CANVAS_SIZE}
    height={CANVAS_SIZE}
    style="cursor: {isPanning ? 'grabbing' : mode === 'pan' ? 'grab' : 'crosshair'}"
    on:click={handleClick}
    on:mousedown={handleMouseDown}
    on:mouseup={handleMouseUp}
    on:mouseleave={handleMouseLeave}
    on:mousemove={handleMouseMove}
    on:wheel|nonpassive={handleWheel}
  />

  <div class="map-legend">
    <span><span class="dot robot"></span> Robot</span>
    <span><span class="dot goal"></span> Goal</span>
    <span><span class="dot obs"></span> Obstacle</span>
    <span class="scale-label">Grid: 20 m × 20 m (0.1 m/cell)</span>
  </div>
</div>

<style>
  .map-panel {
    background: #161b22;
    border-radius: 8px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .map-header {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .panel-title {
    font-weight: 700;
    font-size: 14px;
    color: #e6edf3;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }

  .toolbar { display: flex; gap: 6px; }

  .tool-btn {
    background: #21262d;
    border: 1px solid #30363d;
    color: #8b949e;
    border-radius: 6px;
    padding: 4px 10px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .tool-btn:hover { background: #30363d; color: #e6edf3; }
  .tool-btn.active { background: #1f6feb; border-color: #388bfd; color: #ffffff; }
  .tool-btn.tool-sep { margin-left: 8px; border-color: #444c56; }
  .tool-btn.tool-maze { background: #2d1f47; border-color: #6e40c9; color: #a371f7; }
  .tool-btn.tool-maze:hover { background: #3d2d5c; color: #d2a8ff; }

  .map-hint {
    font-size: 11px;
    color: #6e7681;
    margin-left: auto;
  }

  canvas {
    border-radius: 6px;
    width: 100%;
    height: auto;
    display: block;
  }

  .map-legend {
    display: flex;
    gap: 16px;
    font-size: 11px;
    color: #6e7681;
    align-items: center;
  }

  .dot {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-right: 4px;
  }

  .dot.robot { background: #58a6ff; }
  .dot.goal  { background: #4ecca3; border-radius: 0; }
  .dot.obs   { background: #e05252; border-radius: 2px; }

  .scale-label { margin-left: auto; }
</style>

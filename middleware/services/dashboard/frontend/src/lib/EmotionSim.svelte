<script>
  import { onMount, onDestroy } from 'svelte'
  import * as THREE from 'three'
  import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'
  import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'

  let container
  let animId
  let renderer

  // Reactive display state
  let energy = 80, valence = 0, arousal = 0.5, anxiety = 0.3
  let eyeStr = 'NEUTRAL'
  let feedbackMsg = '', feedbackVisible = false
  let feedbackTimer
  let loadingText = 'Loading model...', loadingVisible = true

  // Internal emotion object (mutated directly for perf, synced to reactive vars each frame)
  const emo = { energy: 80, valence: 0, arousal: 0.5, anxiety: 0.3 }
  const DT = 1 / 25

  const FX = {
    EAR:       { v: -0.05, a: +0.08, ax: +0.15 },
    HEAD:      { v: +0.15, a: +0.05, ax: -0.06 },
    FACE:      { v: +0.18, a: -0.08, ax: -0.10 },
    ARM:       { v: +0.10, a: +0.03, ax: -0.04 },
    BELLY:     { v: +0.20, a: -0.10, ax: -0.12 },
    BACK:      { v: +0.08, a: -0.05, ax: -0.03 },
    NOSE_POKE: { v: -0.15, a: +0.10, ax: +0.12 },
    ROUGH:     { v: -0.30, a: +0.15, ax: +0.20 },
  }

  const ZONE_LABELS = {
    EAR: 'Ear', HEAD: 'Head', FACE: 'Face', ARM: 'Arm',
    BELLY: 'Belly', BACK: 'Back', NOSE_POKE: 'Nose', ROUGH: 'Rough Touch',
  }

  const EYE_LABELS = {
    NEUTRAL: '😐 NEUTRAL', HAPPY: '😊 HAPPY', EXCITED: '🤩 EXCITED',
    SLEEPY: '😴 SLEEPY',   ANXIOUS: '😰 ANXIOUS', SAD: '😢 SAD',
  }

  function clamp(v, a, b) { return Math.max(a, Math.min(b, v)) }

  function eyeState() {
    if (emo.arousal < 0.25)                      return 'SLEEPY'
    if (emo.anxiety > 0.6)                       return 'ANXIOUS'
    if (emo.valence > 0.7 && emo.arousal > 0.7) return 'EXCITED'
    if (emo.valence > 0.5 && emo.arousal > 0.4) return 'HAPPY'
    if (emo.valence < -0.3)                      return 'SAD'
    return 'NEUTRAL'
  }

  function tickEmo() {
    emo.energy  = clamp(emo.energy  - (2 / 60) * DT, 0, 100)
    emo.valence -= emo.valence  * (DT / 120)
    emo.arousal += (0.5 - emo.arousal) * (DT / 60)
    emo.anxiety -= emo.anxiety  * (DT / 300)
  }

  function applyTouch(zone) {
    const fx = FX[zone]; if (!fx) return
    emo.valence = clamp(emo.valence + fx.v, -1, 1)
    emo.arousal = clamp(emo.arousal + fx.a,  0, 1)
    emo.anxiety = clamp(emo.anxiety + fx.ax, 0, 1)
  }

  function syncUI() {
    energy  = emo.energy
    valence = emo.valence
    arousal = emo.arousal
    anxiety = emo.anxiety
    eyeStr  = eyeState()
  }

  function showFeedback(msg) {
    feedbackMsg     = msg
    feedbackVisible = true
    clearTimeout(feedbackTimer)
    feedbackTimer = setTimeout(() => { feedbackVisible = false }, 1500)
  }

  onMount(() => {
    // ── Renderer
    renderer = new THREE.WebGLRenderer({ antialias: true })
    renderer.setPixelRatio(window.devicePixelRatio)
    renderer.setSize(container.clientWidth, container.clientHeight)
    renderer.shadowMap.enabled = true
    renderer.shadowMap.type = THREE.PCFSoftShadowMap
    renderer.outputEncoding = THREE.sRGBEncoding
    container.appendChild(renderer.domElement)

    // ── Scene
    const scene = new THREE.Scene()
    scene.background = new THREE.Color(0x0d0d1a)
    scene.fog = new THREE.FogExp2(0x0d0d1a, 0.04)

    const camera = new THREE.PerspectiveCamera(40, container.clientWidth / container.clientHeight, 0.1, 50)
    camera.position.set(0, 1.4, 4.5)

    const orbit = new OrbitControls(camera, renderer.domElement)
    orbit.target.set(0, 0.9, 0)
    orbit.enableDamping = true
    orbit.dampingFactor = 0.07
    orbit.minDistance = 2.0
    orbit.maxDistance = 10

    // ── Lights
    scene.add(new THREE.AmbientLight(0xffffff, 0.8))
    const sun = new THREE.DirectionalLight(0xfff8ee, 1.2)
    sun.position.set(3, 7, 4)
    sun.castShadow = true
    sun.shadow.mapSize.width = sun.shadow.mapSize.height = 1024
    scene.add(sun)
    const fill = new THREE.DirectionalLight(0xc8d8ff, 0.4)
    fill.position.set(-4, 2, -2)
    scene.add(fill)

    // ── Ground
    const gnd = new THREE.Mesh(
      new THREE.CircleGeometry(5, 48),
      new THREE.MeshPhongMaterial({ color: 0x10102a })
    )
    gnd.rotation.x = -Math.PI / 2
    gnd.receiveShadow = true
    scene.add(gnd)
    scene.add(new THREE.GridHelper(8, 24, 0x1a1a38, 0x1a1a38))

    // ── Robot group
    const robot = new THREE.Group()
    scene.add(robot)

    let hitMeshes = []

    // ── Load GLB
    new GLTFLoader().load(
      'http://localhost:8765/meshes/bunny.glb',
      (gltf) => {
        const model = gltf.scene
        const box   = new THREE.Box3().setFromObject(model)
        const size   = new THREE.Vector3()
        const center = new THREE.Vector3()
        box.getSize(size)
        box.getCenter(center)

        model.scale.setScalar(1.8 / size.y)
        box.setFromObject(model)
        box.getCenter(center)
        box.getSize(size)
        model.position.set(-center.x, -box.min.y, -center.z)

        model.traverse((child) => {
          if (child.isMesh || child.isSkinnedMesh) {
            child.castShadow = true
            child.receiveShadow = true
          }
        })
        robot.add(model)

        const h = size.y, w = size.x, d = size.z

        // Zone hit boxes — ordered front-to-back so raycaster picks closest first
        const zoneDefs = [
          ['EAR',       -w*0.11, h*0.88,    0,    w*0.15, h*0.24, d*0.18],
          ['EAR',       +w*0.11, h*0.88,    0,    w*0.15, h*0.24, d*0.18],
          ['NOSE_POKE',       0, h*0.735, d*0.34, w*0.055, h*0.04, d*0.05],
          ['FACE',            0, h*0.73,  d*0.24, w*0.50, h*0.24, d*0.16],
          ['HEAD',            0, h*0.73,    0,    w*0.72, h*0.28, d*0.48],
          ['ARM',      -w*0.42, h*0.52,    0,    w*0.20, h*0.26, d*0.46],
          ['ARM',      +w*0.42, h*0.52,    0,    w*0.20, h*0.26, d*0.46],
          ['BELLY',           0, h*0.37,  d*0.14, w*0.54, h*0.44, d*0.20],
          ['BACK',            0, h*0.37, -d*0.14, w*0.68, h*0.44, d*0.24],
        ]

        hitMeshes = zoneDefs.map(([zone, cx, cy, cz, bw, bh, bd]) => {
          const mesh = new THREE.Mesh(
            new THREE.BoxGeometry(bw, bh, bd),
            new THREE.MeshBasicMaterial({ transparent: true, opacity: 0 })
          )
          mesh.position.set(cx, cy, cz)
          mesh.userData.zone = zone
          robot.add(mesh)
          return mesh
        })

        loadingVisible = false
      },
      (xhr) => {
        loadingText = `Loading model... ${Math.round(xhr.loaded / xhr.total * 100)}%`
      },
      (err) => {
        loadingText = `Load failed: ${err.message}`
        console.error(err)
      }
    )

    // ── Flash sphere
    const flashMat  = new THREE.MeshBasicMaterial({ color: 0xffdd55, transparent: true, opacity: 0.9 })
    const flashMesh = new THREE.Mesh(new THREE.SphereGeometry(0.04, 8, 8), flashMat)
    flashMesh.visible = false
    scene.add(flashMesh)

    function flashHit(point) {
      flashMesh.position.copy(point)
      flashMesh.visible = true
      setTimeout(() => { flashMesh.visible = false }, 200)
    }

    // ── Raycasting helpers
    const rc = new THREE.Raycaster()
    let mdAt = { x: 0, y: 0 }, wasDrag = false

    function getHits(cx, cy) {
      if (!hitMeshes.length) return []
      const rect = container.getBoundingClientRect()
      const m = new THREE.Vector2(
        ((cx - rect.left) / rect.width)  * 2 - 1,
       -((cy - rect.top)  / rect.height) * 2 + 1
      )
      scene.updateMatrixWorld(true)
      rc.setFromCamera(m, camera)
      return rc.intersectObjects(hitMeshes, false)
    }

    // ── Input events
    renderer.domElement.addEventListener('mousemove', (e) => {
      renderer.domElement.style.cursor = getHits(e.clientX, e.clientY).length ? 'pointer' : 'default'
    })

    const onPtrDown = (e) => { mdAt = { x: e.clientX, y: e.clientY }; wasDrag = false }
    const onPtrMove = (e) => {
      const dx = e.clientX - mdAt.x, dy = e.clientY - mdAt.y
      if (dx*dx + dy*dy > 100) wasDrag = true
    }
    window.addEventListener('pointerdown', onPtrDown, true)
    window.addEventListener('pointermove', onPtrMove, true)

    renderer.domElement.addEventListener('click', (e) => {
      if (wasDrag) return
      const hits = getHits(e.clientX, e.clientY)
      if (!hits.length) return
      const zone = hits[0].object.userData.zone || 'BELLY'
      applyTouch(zone)
      showFeedback('✋  ' + ZONE_LABELS[zone] + '  (' + zone + ')')
      flashHit(hits[0].point)
    })

    renderer.domElement.addEventListener('contextmenu', (e) => {
      e.preventDefault()
      const hits = getHits(e.clientX, e.clientY)
      if (!hits.length) return
      applyTouch('ROUGH')
      showFeedback('👊  Rough Touch  (ROUGH)')
    })

    // ── Animation loop
    let lastTick = 0
    function animate(t) {
      animId = requestAnimationFrame(animate)
      orbit.update()
      if (t - lastTick >= 40) { tickEmo(); lastTick = t }
      syncUI()
      robot.position.y = Math.sin(t * 0.0009) * 0.015
      robot.rotation.y = Math.sin(t * 0.00035) * 0.04
      renderer.render(scene, camera)
    }
    animate(0)

    // ── Resize
    const ro = new ResizeObserver(() => {
      if (!container) return
      camera.aspect = container.clientWidth / container.clientHeight
      camera.updateProjectionMatrix()
      renderer.setSize(container.clientWidth, container.clientHeight)
    })
    ro.observe(container)

    return () => {
      ro.disconnect()
      window.removeEventListener('pointerdown', onPtrDown, true)
      window.removeEventListener('pointermove', onPtrMove, true)
    }
  })

  onDestroy(() => {
    cancelAnimationFrame(animId)
    renderer?.dispose()
  })

  $: energyPct  = energy
  $: valencePct = (valence + 1) / 2 * 100
  $: arousalPct = arousal * 100
  $: anxietyPct = anxiety * 100
</script>

<div class="wrap" bind:this={container}>
  <div class="panel">
    <h3>Emotion State</h3>
    <div class="eye-label">{EYE_LABELS[eyeStr]}</div>
    <div class="bar-row">
      <label><span>ENERGY</span><span>{Math.round(energy)}</span></label>
      <div class="bar-track"><div class="bar-fill" style="background:#4caf50;width:{energyPct}%"></div></div>
    </div>
    <div class="bar-row">
      <label><span>VALENCE</span><span>{valence.toFixed(2)}</span></label>
      <div class="bar-track"><div class="bar-fill" style="background:#2196f3;width:{valencePct}%"></div></div>
    </div>
    <div class="bar-row">
      <label><span>AROUSAL</span><span>{arousal.toFixed(2)}</span></label>
      <div class="bar-track"><div class="bar-fill" style="background:#ff9800;width:{arousalPct}%"></div></div>
    </div>
    <div class="bar-row">
      <label><span>ANXIETY</span><span>{anxiety.toFixed(2)}</span></label>
      <div class="bar-track"><div class="bar-fill" style="background:#f44336;width:{anxietyPct}%"></div></div>
    </div>
  </div>

  <div class="controls-hint">
    Drag: Rotate<br>Scroll: Zoom<br>Click: Touch<br>Right-click: Rough
  </div>

  {#if feedbackVisible}
    <div class="feedback">{feedbackMsg}</div>
  {/if}

  {#if loadingVisible}
    <div class="loading">{loadingText}</div>
  {/if}
</div>

<style>
  .wrap {
    position: relative;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background: #0d0d1a;
    font-family: 'Courier New', monospace;
    color: #ddd;
  }

  .panel {
    position: absolute;
    top: 16px; right: 16px;
    background: rgba(8,8,22,0.93);
    border: 1px solid #252545;
    border-radius: 14px;
    padding: 18px;
    width: 210px;
    backdrop-filter: blur(6px);
    pointer-events: none;
    z-index: 10;
  }

  .panel h3 {
    font-size: 10px; color: #444;
    letter-spacing: 3px; text-transform: uppercase;
    margin-bottom: 12px;
  }

  .eye-label {
    font-size: 18px; text-align: center;
    padding: 8px 6px;
    background: rgba(255,255,255,0.03);
    border-radius: 8px; margin-bottom: 14px;
  }

  .bar-row { margin-bottom: 9px; }

  .bar-row label {
    display: flex; justify-content: space-between;
    font-size: 10px; color: #555; margin-bottom: 3px;
  }

  .bar-track { height: 5px; background: #15152a; border-radius: 3px; overflow: hidden; }
  .bar-fill  { height: 100%; border-radius: 3px; transition: width 0.1s ease; }

  .controls-hint {
    position: absolute; top: 16px; left: 16px;
    font-size: 11px; color: #444; line-height: 2;
    pointer-events: none; z-index: 10;
  }

  .feedback {
    position: absolute; bottom: 52px;
    left: 50%; transform: translateX(-50%);
    font-size: 13px; color: #bbb;
    white-space: nowrap; pointer-events: none; z-index: 10;
  }

  .loading {
    position: absolute; top: 50%; left: 50%;
    transform: translate(-50%, -50%);
    font-size: 13px; color: #555; letter-spacing: 2px; z-index: 10;
  }
</style>

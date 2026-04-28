import { writable } from 'svelte/store'

export const neodm = writable(null)
export const sensor = writable(null)
export const nav = writable(null)
export const connected = writable(false)

// Obstacles: Map of "cx,cy" → true (blocked)
export const obstacles = writable(new Map())

let socket = null

function getWsUrl() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws`
}

export const ws = {
  connect() {
    socket = new WebSocket(getWsUrl())

    socket.onopen = () => connected.set(true)
    socket.onclose = () => {
      connected.set(false)
      setTimeout(() => ws.connect(), 2000)
    }
    socket.onerror = () => socket.close()

    socket.onmessage = (e) => {
      const msg = JSON.parse(e.data)
      switch (msg.type) {
        case 'neodm':  neodm.set(msg.data);  break
        case 'sensor': sensor.set(msg.data); break
        case 'nav':    nav.set(msg.data);    break
      }
    }
  },

  disconnect() {
    socket?.close()
  },

  setGoal(x, y) {
    socket?.send(JSON.stringify({ type: 'set_goal', x, y }))
  },

  setObstacle(x, y, blocked) {
    socket?.send(JSON.stringify({ type: 'set_obstacle', x, y, blocked }))
  },

  resetMap() {
    socket?.send(JSON.stringify({ type: 'reset_map' }))
  },

  resetRobot() {
    socket?.send(JSON.stringify({ type: 'reset_robot' }))
  },

  setMaze(obstacles) {
    socket?.send(JSON.stringify({ type: 'set_maze', obstacles }))
  }
}

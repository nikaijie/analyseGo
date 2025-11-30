<template>
  <div>
    <div style="display:flex; gap:24px; align-items:center; margin:16px 0">
      <div>
        <div style="font-size:13px; color:#666">锁阻塞</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.blockLock ?? 0 }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">IO 阻塞</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.blockIO ?? 0 }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">≥10s 阻塞</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.blockPerm ?? 0 }}</div>
      </div>
    </div>

    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">{{ chartTitle }}</div>
      <div style="display:flex; gap:8px; margin-bottom:8px">
        <button @click="metric='lock'" :style="metric==='lock' ? activeBtn : btn">锁阻塞</button>
        <button @click="metric='io'" :style="metric==='io' ? activeBtn : btn">IO 阻塞</button>
        <button @click="metric='perm'" :style="metric==='perm' ? activeBtn : btn">≥10s 阻塞</button>
      </div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <path :d="path" stroke="#f59e0b" stroke-width="2" fill="none" />
        <g v-if="grid">
          <path v-for="g in grid" :key="g" :d="g" stroke="#eee" stroke-width="1" fill="none" />
        </g>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { API_BASE } from '../config'

type Sample = { time: number; blockLock: number; blockIO: number; blockPerm: number }

const samples = ref<Sample[]>([])
const metric = ref<'lock'|'io'|'perm'>('lock')
const width = 800
const height = 300
const maxPoints = 600

const latest = computed(() => samples.value[samples.value.length - 1])

function pushSample(s: any) {
  samples.value.push({ time: s.time, blockLock: s.blockLock ?? 0, blockIO: s.blockIO ?? 0, blockPerm: s.blockPerm ?? 0 })
  if (samples.value.length > maxPoints) samples.value.splice(0, samples.value.length - maxPoints)
}

function scaleY(v: number, min: number, max: number) {
  if (max === min) return height / 2
  const pad = 8
  const h = height - pad * 2
  return height - pad - ((v - min) / (max - min)) * h
}

const path = computed(() => {
  if (samples.value.length === 0) return ''
  const data = samples.value
  const vals = data.map(d => metric.value === 'lock' ? d.blockLock : metric.value === 'io' ? d.blockIO : d.blockPerm)
  const min = Math.min(...vals)
  const max = Math.max(...vals)
  const stepX = width / Math.max(data.length - 1, 1)
  let d = ''
  for (let i = 0; i < data.length; i++) {
    const x = i * stepX
    const yVal = metric.value === 'lock' ? data[i].blockLock : metric.value === 'io' ? data[i].blockIO : data[i].blockPerm
    const y = scaleY(yVal, min, max)
    d += i === 0 ? `M ${x} ${y}` : ` L ${x} ${y}`
  }
  return d
})

const grid = computed(() => {
  const lines: string[] = []
  const n = 5
  const stepY = height / n
  for (let i = 1; i < n; i++) lines.push(`M 0 ${i * stepY} L ${width} ${i * stepY}`)
  return lines
})

let es: EventSource | null = null
let reconnectTimer: number | null = null
let isMounted = false

function connectStream() {
  if (!isMounted) return
  
  es?.close()
  es = null
  
  try {
    es = new EventSource(`${API_BASE}/api/metrics/stream`)
    es.onmessage = e => {
      if (!isMounted) {
        es?.close()
        return
      }
      try {
        pushSample(JSON.parse(e.data))
      } catch {}
    }
    es.onerror = () => {
      if (!isMounted) {
        es?.close()
        return
      }
      es?.close()
      es = null
      if (reconnectTimer == null && isMounted) {
        reconnectTimer = window.setTimeout(() => {
          reconnectTimer = null
          if (isMounted) {
            connectStream()
          }
        }, 2000)
      }
    }
  } catch (error) {
    console.error('Failed to create EventSource:', error)
  }
}

onMounted(async () => {
  isMounted = true
  try {
    const res = await fetch(`${API_BASE}/api/metrics/history`)
    const h = await res.json()
    if (Array.isArray(h)) h.forEach((s: any) => pushSample(s))
  } catch {}
  connectStream()
})

onUnmounted(() => {
  isMounted = false
  if (reconnectTimer != null) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
  if (es) {
    es.close()
    es = null
  }
})

const btn = 'padding:6px 10px; border:1px solid #ddd; border-radius:6px; background:#fff'
const activeBtn = 'padding:6px 10px; border:1px solid #f59e0b; color:#f59e0b; border-radius:6px; background:#fff7ed'
</script>


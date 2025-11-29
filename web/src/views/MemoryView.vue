<template>
  <div>
    <div style="display:flex; gap:24px; align-items:center; margin:16px 0">
      <div>
        <div style="font-size:13px; color:#666">HeapAlloc</div>
        <div style="font-size:24px; font-weight:600">{{ formatMB(latest?.heapAlloc ?? 0) }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">HeapInuse</div>
        <div style="font-size:24px; font-weight:600">{{ formatMB(latest?.heapInuse ?? 0) }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">HeapObjects</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.heapObjects ?? 0 }}</div>
      </div>
    </div>

    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">HeapAlloc（最近 10 分钟）</div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <path :d="path" stroke="#10b981" stroke-width="2" fill="none" />
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

type Sample = { time: number; goroutines: number; requests: number; heapAlloc: number; heapInuse: number; heapSys: number; heapObjects: number }

const samples = ref<Sample[]>([])
const width = 800
const height = 300
const maxPoints = 600

const latest = computed(() => samples.value[samples.value.length - 1])

function pushSample(s: Sample) {
  samples.value.push(s)
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
  const vals = data.map(d => d.heapAlloc)
  const min = Math.min(...vals)
  const max = Math.max(...vals)
  const stepX = width / Math.max(data.length - 1, 1)
  let d = ''
  for (let i = 0; i < data.length; i++) {
    const x = i * stepX
    const y = scaleY(data[i].heapAlloc, min, max)
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

function connectStream() {
  es?.close()
  es = new EventSource(`${API_BASE}/api/metrics/stream`)
  es.onmessage = e => {
    try {
      const s = JSON.parse(e.data)
      pushSample(s)
    } catch {}
  }
  es.onerror = () => {
    es?.close()
    if (reconnectTimer == null) {
      reconnectTimer = window.setTimeout(() => {
        reconnectTimer = null
        connectStream()
      }, 2000)
    }
  }
}

onMounted(async () => {
  try {
    const res = await fetch(`${API_BASE}/api/metrics/history`)
    const h = await res.json()
    if (Array.isArray(h)) h.forEach((s: Sample) => pushSample(s))
  } catch {}
  connectStream()
})

onUnmounted(() => {
  es?.close()
  if (reconnectTimer != null) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
})

function formatMB(v: number) {
  return (v / 1024 / 1024).toFixed(1) + ' MB'
}
</script>


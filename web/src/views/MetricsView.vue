<template>
  <div>
    <div style="display:flex; gap:24px; align-items:center; margin:16px 0">
      <div>
        <div style="font-size:13px; color:#666">当前 goroutine</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.goroutines ?? 0 }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">最近 10 秒请求数</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.requests ?? 0 }}</div>
      </div>
      <button @click="ping" :style="btn">触发 /api/ping</button>
      <button @click="pingSlow" :style="btn">触发 /api/ping/slow</button>
      <button @click="spawnBusy" :style="btn">触发 /api/busy</button>
    </div>

    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px">
        <div style="font-size:13px; color:#666">{{ chartTitle }}</div>
        <div style="display:flex; gap:8px">
          <button @click="setWindow(600)" :style="windowSec===600 ? activeBtn : btn">10 分钟</button>
          <button @click="setWindow(86400)" :style="windowSec===86400 ? activeBtn : btn">24 小时</button>
        </div>
      </div>
      <div style="display:flex; gap:8px; margin-bottom:8px">
        <button @click="metric='requests'" :style="metric==='requests' ? activeBtn : btn">最近10秒请求数</button>
        <button @click="metric='goroutines'" :style="metric==='goroutines' ? activeBtn : btn">goroutine 数量</button>
      </div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <path :d="axes.x" stroke="#999" stroke-width="1" fill="none" />
        <path :d="axes.y" stroke="#999" stroke-width="1" fill="none" />
        <g v-if="grid">
          <path v-for="g in grid" :key="g" :d="g" stroke="#eee" stroke-width="1" fill="none" />
        </g>
        <path :d="path" stroke="#3b82f6" stroke-width="2" fill="none" />
        <g>
          <text v-for="t in xTicks" :key="t.x" :x="t.x" :y="height - bottomPad + 14" font-size="10" text-anchor="middle" fill="#666">{{ t.label }}</text>
          <text v-for="t in yTicks" :key="t.y" :x="leftPad - 6" :y="t.y + 3" font-size="10" text-anchor="end" fill="#666">{{ t.label }}</text>
        </g>
      </svg>
    </div>

    <div style="margin-top:16px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">接口详情（最近 10 秒）</div>
      <RoutesView />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { API_BASE } from '../config'
import RoutesView from './RoutesView.vue'

type Sample = { time: number; goroutines: number; requests: number; heapAlloc: number; heapInuse: number; heapSys: number; heapObjects: number }

const samples = ref<Sample[]>([])
const metric = ref<'goroutines'|'requests'>('requests')
const width = 800
const height = 320
const leftPad = 44
const rightPad = 8
const topPad = 8
const bottomPad = 24
const windowSec = ref(600)
const maxPoints = ref(600)
const MAX_Y = 5000

const latest = computed(() => samples.value[samples.value.length - 1])

function pushSample(s: Sample) {
  samples.value.push(s)
  if (samples.value.length > maxPoints.value) samples.value.splice(0, samples.value.length - maxPoints.value)
}

function scaleY(v: number, min: number, max: number) {
  if (max === min) return height / 2
  const pad = 8
  const h = height - pad * 2
  return height - pad - ((v - min) / (max - min)) * h
}

const path = computed(() => {
  if (samples.value.length === 0) return ''
  const data = renderData()
  const axisMin = 0
  const axisMax = MAX_Y
  const innerW = width - leftPad - rightPad
  const innerH = height - topPad - bottomPad
  const stepX = innerW / Math.max(data.length - 1, 1)
  let d = ''
  const stride = Math.max(1, Math.ceil(data.length / innerW))
  for (let i = 0; i < data.length; i += stride) {
    const x = leftPad + i * stepX
    let yVal = metric.value === 'goroutines' ? data[i].goroutines : data[i].requests
    if (yVal < axisMin) yVal = axisMin
    if (yVal > axisMax) yVal = axisMax
    const y = topPad + (innerH - ((yVal - axisMin) / (axisMax - axisMin)) * innerH)
    d += i === 0 ? `M ${x} ${y}` : ` L ${x} ${y}`
  }
  return d
})

const grid = computed(() => {
  const lines: string[] = []
  const n = 5
  const innerH = height - topPad - bottomPad
  const stepY = innerH / n
  for (let i = 1; i < n; i++) lines.push(`M ${leftPad} ${topPad + i * stepY} L ${width - rightPad} ${topPad + i * stepY}`)
  return lines
})

async function ping() {
  await fetch(`${API_BASE}/api/ping`)
}

async function pingSlow() {
  await fetch(`${API_BASE}/api/ping/slow?ms=2000`)
}

async function spawnBusy() {
  await fetch(`${API_BASE}/busy?n=50&ms=2000`)
}

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
  await loadHistory()
  connectStream()
})

onUnmounted(() => {
  es?.close()
  if (reconnectTimer != null) {
    window.clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
})

const chartTitle = computed(() => metric.value === 'goroutines' ? 'goroutine 数量（最近 10 分钟）' : '最近 10 秒请求数（最近 10 分钟）')
const btn = 'padding:6px 10px; border:1px solid #ddd; border-radius:6px; background:#fff'
const activeBtn = 'padding:6px 10px; border:1px solid #3b82f6; color:#3b82f6; border-radius:6px; background:#eef5ff'

const axes = computed(() => ({
  x: `M ${leftPad} ${height - bottomPad} L ${width - rightPad} ${height - bottomPad}`,
  y: `M ${leftPad} ${topPad} L ${leftPad} ${height - bottomPad}`
}))

const xTicks = computed(() => {
  const res: { x: number; label: string }[] = []
  if (samples.value.length === 0) return res
  const data = renderData()
  const innerW = width - leftPad - rightPad
  const count = windowSec.value === 86400 ? 12 : 6
  const stepX = innerW / Math.max(data.length - 1, 1)
  for (let i = 0; i < count; i++) {
    const ratio = i / (count - 1)
    const idx = Math.round(ratio * (data.length - 1))
    const x = leftPad + idx * stepX
    const d = new Date(data[idx].time)
    const label = windowSec.value === 86400 ? `${String(d.getHours()).padStart(2,'0')}:00` : `${d.getHours()}:${String(d.getMinutes()).padStart(2,'0')}`
    res.push({ x, label })
  }
  return res
})

const yTicks = computed(() => {
  const res: { y: number; label: string }[] = []
  const axisMin = 0
  const axisMax = MAX_Y
  const innerH = height - topPad - bottomPad
  const step = 1000
  for (let v = axisMin; v <= axisMax; v += step) {
    const y = topPad + (innerH - ((v - axisMin) / (axisMax - axisMin)) * innerH)
    res.push({ y, label: String(v) })
  }
  return res
})

function renderData() {
  const data = samples.value
  if (windowSec.value !== 86400) return data
  const byMinute: Record<number, { time: number; sum: number; count: number; gsum: number }> = {}
  for (const s of data) {
    const key = Math.floor(s.time / 60000)
    const val = metric.value === 'goroutines' ? s.goroutines : s.requests
    const prev = byMinute[key]
    if (prev) { prev.sum += val; prev.count += 1; prev.gsum += s.goroutines }
    else { byMinute[key] = { time: key * 60000, sum: val, count: 1, gsum: s.goroutines } }
  }
  const out: any[] = []
  const keys = Object.keys(byMinute).map(k => Number(k)).sort((a,b)=>a-b)
  for (const k of keys) {
    const b = byMinute[k]
    out.push({ time: b.time, goroutines: Math.round(b.gsum / b.count), requests: Math.round(b.sum / b.count) })
  }
  return out
}

async function loadHistory() {
  maxPoints.value = windowSec.value
  samples.value = []
  try {
    const res = await fetch(`${API_BASE}/api/metrics/history?window=${windowSec.value}`)
    const h = await res.json()
    if (Array.isArray(h)) h.forEach((s: Sample) => pushSample(s))
  } catch {}
}

function setWindow(sec: number) {
  windowSec.value = sec
  loadHistory()
}
</script>

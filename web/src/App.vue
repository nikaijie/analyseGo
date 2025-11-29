<template>
  <div style="max-width: 900px; margin: 24px auto; font-family: -apple-system, system-ui, Segoe UI, Roboto, Helvetica, Arial">
    <h2>Gin 服务实时指标</h2>
    <div style="display:flex; gap:24px; align-items:center; margin:16px 0">
      <div>
        <div style="font-size:13px; color:#666">当前 goroutine</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.goroutines ?? 0 }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">最近 10 秒请求数</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.requests ?? 0 }}</div>
      </div>
      <button @click="ping" style="padding:8px 12px; border:1px solid #ddd; border-radius:6px; background:#fff">触发 /api/ping</button>
      <button @click="pingSlow" style="padding:8px 12px; border:1px solid #ddd; border-radius:6px; background:#fff">触发 /api/ping/slow</button>
      <button @click="spawnBusy" style="padding:8px 12px; border:1px solid #ddd; border-radius:6px; background:#fff">触发 /api/busy</button>
    </div>

    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">{{ chartTitle }}</div>
      <div style="display:flex; gap:8px; margin-bottom:8px">
        <button @click="metric='requests'" :style="metric==='requests' ? activeBtn : btn">最近10秒请求数</button>
        <button @click="metric='goroutines'" :style="metric==='goroutines' ? activeBtn : btn">goroutine 数量</button>
      </div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <path :d="path" stroke="#3b82f6" stroke-width="2" fill="none" />
        <g v-if="grid">
          <path v-for="g in grid" :key="g" :d="g" stroke="#eee" stroke-width="1" fill="none" />
        </g>
      </svg>
    </div>
  </div>
  </template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'

type Sample = { time: number; goroutines: number; requests: number }

const samples = ref<Sample[]>([])
const metric = ref<'goroutines'|'requests'>('requests')
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
  const vals = data.map(d => metric.value === 'goroutines' ? d.goroutines : d.requests)
  const min = Math.min(...vals)
  const max = Math.max(...vals)
  const stepX = width / Math.max(data.length - 1, 1)
  let d = ''
  for (let i = 0; i < data.length; i++) {
    const x = i * stepX
    const yVal = metric.value === 'goroutines' ? data[i].goroutines : data[i].requests
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

import { API_BASE } from './config'

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

const chartTitle = computed(() => metric.value === 'goroutines' ? 'goroutine 数量（最近 10 分钟）' : '最近 10 秒请求数（最近 10 分钟）')
const btn = 'padding:6px 10px; border:1px solid #ddd; border-radius:6px; background:#fff'
const activeBtn = 'padding:6px 10px; border:1px solid #3b82f6; color:#3b82f6; border-radius:6px; background:#eef5ff'
</script>

<style>
body { margin: 0; }
</style>

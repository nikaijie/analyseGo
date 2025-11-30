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
          <button @click="yMode='fixed'" :style="yMode==='fixed' ? activeBtn : btn">纵轴固定(0-5000)</button>
          <button @click="yMode='auto'" :style="yMode==='auto' ? activeBtn : btn">纵轴自动</button>
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
        <g v-if="points.length <= 2">
          <circle v-for="(p,i) in points" :key="i" :cx="p.x" :cy="p.y" r="3" fill="#3b82f6" />
        </g>
        <rect :x="leftPad" :y="topPad" :width="width - leftPad - rightPad" :height="height - topPad - bottomPad" fill="transparent" @mousemove="onMove" @mouseleave="onLeave" />
        <g v-if="hover">
          <line :x1="hover.x" :x2="hover.x" :y1="topPad" :y2="height - bottomPad" stroke="#bbb" stroke-dasharray="4 4" />
          <circle :cx="hover.x" :cy="hover.y" r="3.5" fill="#3b82f6" />
          <text :x="hover.x + 8" :y="hover.y - 8" font-size="11" fill="#111">{{ hover.label }}</text>
        </g>
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
const yMode = ref<'fixed'|'auto'>('fixed')
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
  const pts = points.value
  if (pts.length < 2) return ''
  let d = `M ${pts[0].x} ${pts[0].y}`
  for (let i = 1; i < pts.length; i++) d += ` L ${pts[i].x} ${pts[i].y}`
  return d
})

const grid = computed(() => {
  const lines: string[] = []
  const innerH = height - topPad - bottomPad
  const amin = axisMin.value
  const amax = axisMax.value
  const step = yStep.value
  for (let v = amin + step; v < amax; v += step) {
    const y = topPad + (innerH - ((v - amin) / (amax - amin)) * innerH)
    lines.push(`M ${leftPad} ${y} L ${width - rightPad} ${y}`)
  }
  return lines
})

async function ping() {
  await fetch(`${API_BASE}/api/ping`)
}

async function pingSlow() {
  await fetch(`${API_BASE}/api/ping/slow?ms=2000`)
}

async function spawnBusy() {
  await fetch(`${API_BASE}/api/busy?n=50&ms=2000`)
}

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
        const s = JSON.parse(e.data)
        pushSample(s)
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
  await loadHistory()
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

const chartTitle = computed(() => metric.value === 'goroutines' ? 'goroutine 数量（最近 10 分钟）' : '最近 10 秒请求数（最近 10 分钟）')
const btn = 'padding:6px 10px; border:1px solid #ddd; border-radius:6px; background:#fff'
const activeBtn = 'padding:6px 10px; border:1px solid #3b82f6; color:#3b82f6; border-radius:6px; background:#eef5ff'

const axes = computed(() => ({
  x: `M ${leftPad} ${height - bottomPad} L ${width - rightPad} ${height - bottomPad}`,
  y: `M ${leftPad} ${topPad} L ${leftPad} ${height - bottomPad}`
}))

const xTicks = computed(() => {
  const res: { x: number; label: string }[] = []
  const innerW = width - leftPad - rightPad
  
  // 24小时视图：固定显示0-24点（一整天）
  if (windowSec.value === 86400) {
    const tickCount = 13 // 0, 2, 4, ..., 22, 24点，共13个刻度
    const stepX = innerW / (tickCount - 1) // 横轴均匀分布
    for (let i = 0; i < tickCount; i++) {
      const hour = i * 2 // 每2小时一个刻度：0, 2, 4, ..., 22, 24
      const x = leftPad + i * stepX
      const label = hour === 24 ? '24:00' : `${String(hour).padStart(2, '0')}:00`
      res.push({ x, label })
    }
  }
  // 10分钟视图：保持原有逻辑（基于数据时间）
  else {
    if (samples.value.length === 0) return res
    const data = renderData()
    const count = 6
    const stepX = innerW / Math.max(data.length - 1, 1)
    for (let i = 0; i < count; i++) {
      const ratio = i / (count - 1)
      const idx = Math.round(ratio * (data.length - 1))
      const x = leftPad + idx * stepX
      const d = new Date(data[idx].time)
      const label = `${d.getHours()}:${String(d.getMinutes()).padStart(2, '0')}`
      res.push({ x, label })
    }
  }
  return res
})

const yTicks = computed(() => {
  const res: { y: number; label: string }[] = []
  const innerH = height - topPad - bottomPad
  const amin = axisMin.value
  const amax = axisMax.value
  const step = yStep.value
  for (let v = amin; v <= amax; v += step) {
    const y = topPad + (innerH - ((v - amin) / (amax - amin)) * innerH)
    res.push({ y, label: String(v) })
  }
  return res
})

function renderData() {
  const data = samples.value
  if (windowSec.value !== 86400) return data
  
  // 24小时视图：按小时和分钟聚合数据（基于时间戳中的小时和分钟）
  // 使用 "小时:分钟" 作为key，这样同一天同一时间的数据会被聚合
  const byHourMinute: Record<string, { time: number; sum: number; count: number; gsum: number }> = {}
  
  for (const s of data) {
    const date = new Date(s.time)
    const hours = date.getHours()
    const minutes = date.getMinutes()
    const key = `${hours}:${minutes}` // 使用 "小时:分钟" 作为key
    
    const val = metric.value === 'goroutines' ? s.goroutines : s.requests
    const prev = byHourMinute[key]
    
    if (prev) {
      prev.sum += val
      prev.count += 1
      prev.gsum += s.goroutines
      // 保留最新的时间戳
      if (s.time > prev.time) {
        prev.time = s.time
      }
    } else {
      byHourMinute[key] = {
        time: s.time,
        sum: val,
        count: 1,
        gsum: s.goroutines
      }
    }
  }
  
  // 转换为数组并按时间排序
  const out: any[] = []
  for (const key in byHourMinute) {
    const b = byHourMinute[key]
    out.push({
      time: b.time,
      goroutines: Math.round(b.gsum / b.count),
      requests: Math.round(b.sum / b.count)
    })
  }
  
  // 按时间排序
  out.sort((a, b) => {
    const dateA = new Date(a.time)
    const dateB = new Date(b.time)
    const minutesA = dateA.getHours() * 60 + dateA.getMinutes()
    const minutesB = dateB.getHours() * 60 + dateB.getMinutes()
    return minutesA - minutesB
  })
  
  return out
}

function niceStep(x: number) {
  const p = Math.pow(10, Math.floor(Math.log10(x || 1)))
  const v = (x || 1) / p
  let s = 1
  if (v >= 5) s = 5
  else if (v >= 2) s = 2
  return s * p
}

const axisMin = computed(() => 0)
const axisMax = computed(() => {
  if (yMode.value === 'fixed') return MAX_Y
  const data = dataComp.value
  const arr = data.map(d => (metric.value === 'goroutines' ? d.goroutines : d.requests))
  const max = Math.max(1, ...arr)
  const step = niceStep(max / 5)
  let up = Math.ceil(max / step) * step
  if (up <= 50) return 50
  if (up <= 100) return 100
  return up
})

const yStep = computed(() => {
  if (yMode.value === 'fixed') return 1000
  const range = axisMax.value - axisMin.value
  return niceStep(range / 5)
})

const dataComp = computed(() => renderData())
const points = computed(() => {
  const data = dataComp.value
  if (data.length === 0) return []
  
  const innerW = width - leftPad - rightPad
  const innerH = height - topPad - bottomPad
  const amin = axisMin.value
  const amax = axisMax.value
  
  const res: { x:number; y:number }[] = []
  
  if (windowSec.value === 86400) {
    // 24小时视图：基于数据的时间戳中的小时和分钟映射到0-24小时的时间轴
    for (let i = 0; i < data.length; i++) {
      const sampleDate = new Date(data[i].time)
      const hours = sampleDate.getHours()
      const minutes = sampleDate.getMinutes()
      
      // 计算该数据点在一天中的位置（0-1之间）
      // 例如：12:30 = 12*60 + 30 = 750分钟，占一天的 750/1440 ≈ 0.521
      const minutesInDay = hours * 60 + minutes
      const timeRatio = minutesInDay / 1440 // 1440分钟 = 24小时
      
      let yVal = metric.value === 'goroutines' ? data[i].goroutines : data[i].requests
      if (yVal < amin) yVal = amin
      if (yVal > amax) yVal = amax
      
      // X坐标基于时间比例（0点在最左边，24点在最右边）
      const x = leftPad + timeRatio * innerW
      const y = topPad + (innerH - ((yVal - amin) / (amax - amin)) * innerH)
      res.push({ x, y })
    }
  } else {
    // 10分钟视图：基于数据索引
    const stepX = innerW / Math.max(data.length - 1, 1)
    for (let i = 0; i < data.length; i++) {
      let yVal = metric.value === 'goroutines' ? data[i].goroutines : data[i].requests
      if (yVal < amin) yVal = amin
      if (yVal > amax) yVal = amax
      const x = leftPad + i * stepX
      const y = topPad + (innerH - ((yVal - amin) / (amax - amin)) * innerH)
      res.push({ x, y })
    }
  }
  
  return res
})

const hoverIndex = ref<number|null>(null)
function onMove(e: MouseEvent) {
  const x = (e as any).offsetX
  const pts = points.value
  if (!pts.length) { hoverIndex.value = null; return }
  let min = Infinity, idx = 0
  for (let i = 0; i < pts.length; i++) {
    const d = Math.abs(pts[i].x - x)
    if (d < min) { min = d; idx = i }
  }
  hoverIndex.value = idx
}
function onLeave() { hoverIndex.value = null }

const metricLabel = computed(() => metric.value === 'goroutines' ? 'goroutine 数量' : '最近10秒请求数')
const hover = computed(() => {
  if (hoverIndex.value == null) return null
  const i = hoverIndex.value
  const pts = points.value
  const data = dataComp.value
  
  if (i >= pts.length) return null
  
  const p = pts[i]
  
  // 找到对应的数据点
  let dataIdx = -1
  let v = 0
  let timeStr = ''
  
  if (windowSec.value === 86400) {
    // 24小时视图：通过X坐标反推时间（0-24小时），找到最接近的数据点
    const innerW = width - leftPad - rightPad
    const x = p.x - leftPad
    const timeRatio = x / innerW // 0-1之间，对应0-24小时
    const targetMinutes = timeRatio * 1440 // 目标分钟数（0-1440）
    
    // 找到最接近的数据点（基于小时和分钟）
    let minDiff = Infinity
    for (let j = 0; j < data.length; j++) {
      const sampleDate = new Date(data[j].time)
      const sampleMinutes = sampleDate.getHours() * 60 + sampleDate.getMinutes()
      const diff = Math.abs(sampleMinutes - targetMinutes)
      if (diff < minDiff) {
        minDiff = diff
        dataIdx = j
      }
    }
    
    if (dataIdx >= 0 && dataIdx < data.length) {
      v = metric.value === 'goroutines' ? data[dataIdx]?.goroutines ?? 0 : data[dataIdx]?.requests ?? 0
      const date = new Date(data[dataIdx].time)
      timeStr = `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
    }
  } else {
    // 10分钟视图：直接使用索引
    dataIdx = i
    if (dataIdx < data.length) {
      v = metric.value === 'goroutines' ? data[dataIdx]?.goroutines ?? 0 : data[dataIdx]?.requests ?? 0
      const date = new Date(data[dataIdx].time)
      timeStr = `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
    }
  }
  
  if (dataIdx < 0 || dataIdx >= data.length) return null
  
  return {
    x: p.x,
    y: p.y,
    label: timeStr ? `${timeStr} - ${metricLabel.value}: ${v}` : `${metricLabel.value}: ${v}`
  }
})

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

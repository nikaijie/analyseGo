<template>
  <div>
    <div style="display:flex; gap:24px; align-items:center; margin:16px 0">
      <div>
        <div style="font-size:13px; color:#666">堆内存使用率</div>
        <div style="font-size:24px; font-weight:600">{{ heapUsagePercent }}%</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">HeapAlloc</div>
        <div style="font-size:24px; font-weight:600">{{ formatMB(latest?.heapAlloc ?? 0) }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">HeapSys</div>
        <div style="font-size:24px; font-weight:600">{{ formatMB(latest?.heapSys ?? 0) }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">HeapObjects</div>
        <div style="font-size:24px; font-weight:600">{{ latest?.heapObjects ?? 0 }}</div>
      </div>
      <div>
        <div style="font-size:13px; color:#666">GC总次数</div>
        <div style="font-size:24px; font-weight:600">{{ totalGC }}</div>
      </div>
    </div>

    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px; margin-bottom: 20px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">堆内存使用率（最近 10 分钟）</div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <!-- 坐标轴 -->
        <path :d="axes.x" stroke="#999" stroke-width="1" fill="none" />
        <path :d="axes.y" stroke="#999" stroke-width="1" fill="none" />
        
        <!-- 网格线 -->
        <g v-if="grid">
          <path v-for="g in grid" :key="g" :d="g" stroke="#eee" stroke-width="1" fill="none" />
        </g>
        
        <!-- 数据线 -->
        <path :d="path" stroke="#10b981" stroke-width="2" fill="none" />
        
        <!-- 数据点（数据少时显示） -->
        <g v-if="points.length <= 10">
          <circle v-for="(p,i) in points" :key="i" :cx="p.x" :cy="p.y" r="3" fill="#10b981" />
        </g>
        
        <!-- 悬停区域 -->
        <rect
          :x="leftPad"
          :y="topPad"
          :width="width - leftPad - rightPad"
          :height="height - topPad - bottomPad"
          fill="transparent"
          @mousemove="onMove"
          @mouseleave="onLeave"
        />
        
        <!-- 悬停提示 -->
        <g v-if="hover">
          <line
            :x1="hover.x"
            :x2="hover.x"
            :y1="topPad"
            :y2="height - bottomPad"
            stroke="#bbb"
            stroke-dasharray="4 4"
          />
          <circle :cx="hover.x" :cy="hover.y" r="3.5" fill="#10b981" />
          <text
            :x="hover.x + 8"
            :y="hover.y - 8"
            font-size="11"
            fill="#111"
          >{{ hover.label }}</text>
        </g>
        
        <!-- 坐标轴标签 -->
        <g>
          <text
            v-for="t in xTicks"
            :key="t.x"
            :x="t.x"
            :y="height - bottomPad + 14"
            font-size="10"
            text-anchor="middle"
            fill="#666"
          >{{ t.label }}</text>
          <text
            v-for="t in yTicks"
            :key="t.y"
            :x="leftPad - 6"
            :y="t.y + 3"
            font-size="10"
            text-anchor="end"
            fill="#666"
          >{{ t.label }}</text>
        </g>
      </svg>
    </div>

    <!-- GC次数图表 -->
    <div style="border:1px solid #e5e5e5; border-radius:8px; padding:12px">
      <div style="font-size:13px; color:#666; margin-bottom:8px">GC次数（最近 10 分钟）</div>
      <svg :width="width" :height="height" :viewBox="`0 0 ${width} ${height}`" style="width:100%">
        <!-- 坐标轴 -->
        <path :d="gcAxes.x" stroke="#999" stroke-width="1" fill="none" />
        <path :d="gcAxes.y" stroke="#999" stroke-width="1" fill="none" />
        
        <!-- 网格线 -->
        <g v-if="gcGrid">
          <path v-for="g in gcGrid" :key="g" :d="g" stroke="#eee" stroke-width="1" fill="none" />
        </g>
        
        <!-- 数据线 -->
        <path :d="gcPath" stroke="#f59e0b" stroke-width="2" fill="none" />
        
        <!-- 数据点（数据少时显示） -->
        <g v-if="gcPoints.length <= 10">
          <circle v-for="(p,i) in gcPoints" :key="i" :cx="p.x" :cy="p.y" r="3" fill="#f59e0b" />
        </g>
        
        <!-- 悬停区域 -->
        <rect
          :x="leftPad"
          :y="topPad"
          :width="width - leftPad - rightPad"
          :height="height - topPad - bottomPad"
          fill="transparent"
          @mousemove="onGCMove"
          @mouseleave="onGCLeave"
        />
        
        <!-- 悬停提示 -->
        <g v-if="gcHover">
          <line
            :x1="gcHover.x"
            :x2="gcHover.x"
            :y1="topPad"
            :y2="height - bottomPad"
            stroke="#bbb"
            stroke-dasharray="4 4"
          />
          <circle :cx="gcHover.x" :cy="gcHover.y" r="3.5" fill="#f59e0b" />
          <text
            :x="gcHover.x + 8"
            :y="gcHover.y - 8"
            font-size="11"
            fill="#111"
          >{{ gcHover.label }}</text>
        </g>
        
        <!-- 坐标轴标签 -->
        <g>
          <text
            v-for="t in gcXTicks"
            :key="t.x"
            :x="t.x"
            :y="height - bottomPad + 14"
            font-size="10"
            text-anchor="middle"
            fill="#666"
          >{{ t.label }}</text>
          <text
            v-for="t in gcYTicks"
            :key="t.y"
            :x="leftPad - 6"
            :y="t.y + 3"
            font-size="10"
            text-anchor="end"
            fill="#666"
          >{{ t.label }}</text>
        </g>
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { API_BASE } from '../config'

type Sample = {
  time: number
  goroutines: number
  requests: number
  heapAlloc: number
  heapInuse: number
  heapSys: number
  heapObjects: number
  numGC?: number
  gcIncrement?: number
}

const samples = ref<Sample[]>([])
const width = 800
const height = 320
const leftPad = 44
const rightPad = 8
const topPad = 8
const bottomPad = 24
const maxPoints = 600

const latest = computed(() => samples.value[samples.value.length - 1])

// 计算堆内存使用率百分比
const heapUsagePercent = computed(() => {
  if (!latest.value || latest.value.heapSys === 0) return 0
  return ((latest.value.heapInuse / latest.value.heapSys) * 100).toFixed(1)
})

// GC相关计算
const totalGC = computed(() => {
  return latest.value?.numGC ?? 0
})

function pushSample(s: Sample) {
  samples.value.push(s)
  if (samples.value.length > maxPoints) {
    samples.value.splice(0, samples.value.length - maxPoints)
  }
}

// 计算百分比
function calculatePercent(sample: Sample): number {
  if (!sample || sample.heapSys === 0) return 0
  return (sample.heapInuse / sample.heapSys) * 100
}

// 坐标轴
const axes = computed(() => ({
  x: `M ${leftPad} ${height - bottomPad} L ${width - rightPad} ${height - bottomPad}`,
  y: `M ${leftPad} ${topPad} L ${leftPad} ${height - bottomPad}`
}))

// X轴刻度（时间）
const xTicks = computed(() => {
  const res: { x: number; label: string }[] = []
  if (samples.value.length === 0) return res
  
  const data = samples.value
  const innerW = width - leftPad - rightPad
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
  
  return res
})

// Y轴刻度（百分比）
const yTicks = computed(() => {
  const res: { y: number; label: string }[] = []
  const innerH = height - topPad - bottomPad
  const maxPercent = 100
  
  // 生成5个刻度：0%, 25%, 50%, 75%, 100%
  for (let i = 0; i <= 4; i++) {
    const percent = i * 25
    const y = topPad + (innerH - (percent / maxPercent) * innerH)
    res.push({ y, label: `${percent}%` })
  }
  
  return res
})

// 网格线
const grid = computed(() => {
  const lines: string[] = []
  const innerH = height - topPad - bottomPad
  const maxPercent = 100
  
  // 水平网格线（对应Y轴刻度）
  for (let i = 1; i < 4; i++) {
    const percent = i * 25
    const y = topPad + (innerH - (percent / maxPercent) * innerH)
    lines.push(`M ${leftPad} ${y} L ${width - rightPad} ${y}`)
  }
  
  return lines
})

// 数据点
const points = computed(() => {
  if (samples.value.length === 0) return []
  
  const data = samples.value
  const innerW = width - leftPad - rightPad
  const innerH = height - topPad - bottomPad
  const maxPercent = 100
  
  const res: { x: number; y: number }[] = []
  const stepX = innerW / Math.max(data.length - 1, 1)
  
  for (let i = 0; i < data.length; i++) {
    const percent = calculatePercent(data[i])
    const x = leftPad + i * stepX
    const y = topPad + (innerH - (percent / maxPercent) * innerH)
    res.push({ x, y })
  }
  
  return res
})

// 路径
const path = computed(() => {
  const pts = points.value
  if (pts.length < 2) return ''
  
  let d = `M ${pts[0].x} ${pts[0].y}`
  for (let i = 1; i < pts.length; i++) {
    d += ` L ${pts[i].x} ${pts[i].y}`
  }
  return d
})

// 悬停
const hoverIndex = ref<number | null>(null)

function onMove(e: MouseEvent) {
  const x = (e as any).offsetX
  const pts = points.value
  if (!pts.length) {
    hoverIndex.value = null
    return
  }
  
  let min = Infinity
  let idx = 0
  for (let i = 0; i < pts.length; i++) {
    const d = Math.abs(pts[i].x - x)
    if (d < min) {
      min = d
      idx = i
    }
  }
  hoverIndex.value = idx
}

function onLeave() {
  hoverIndex.value = null
}

const hover = computed(() => {
  if (hoverIndex.value == null) return null
  
  const i = hoverIndex.value
  const pts = points.value
  const data = samples.value
  
  if (i >= pts.length || i >= data.length) return null
  
  const p = pts[i]
  const percent = calculatePercent(data[i])
  const date = new Date(data[i].time)
  const timeStr = `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`
  
  return {
    x: p.x,
    y: p.y,
    label: `${timeStr} - ${percent.toFixed(1)}%`
  }
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
  try {
    const res = await fetch(`${API_BASE}/api/metrics/history`)
    const h = await res.json()
    if (Array.isArray(h)) {
      h.forEach((s: Sample) => pushSample(s))
    }
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

function formatMB(v: number) {
  return (v / 1024 / 1024).toFixed(1) + ' MB'
}

// GC次数图表相关计算
const gcAxes = computed(() => ({
  x: `M ${leftPad} ${height - bottomPad} L ${width - rightPad} ${height - bottomPad}`,
  y: `M ${leftPad} ${topPad} L ${leftPad} ${height - bottomPad}`
}))

// GC X轴刻度（时间）
const gcXTicks = computed(() => {
  const res: { x: number; label: string }[] = []
  if (samples.value.length === 0) return res
  
  const data = samples.value
  const innerW = width - leftPad - rightPad
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
  
  return res
})

// GC Y轴刻度（次数）
const gcYTicks = computed(() => {
  const res: { y: number; label: string }[] = []
  if (samples.value.length === 0) {
    // 如果没有数据，返回默认刻度
    const innerH = height - topPad - bottomPad
    for (let i = 0; i <= 4; i++) {
      const value = i
      const y = topPad + (innerH - (value / 4) * innerH)
      res.push({ y, label: String(value) })
    }
    return res
  }
  
  const innerH = height - topPad - bottomPad
  
  // 计算最大GC增量
  const gcValues = samples.value.map(s => s.gcIncrement ?? 0).filter(v => !isNaN(v))
  const maxGC = gcValues.length > 0 ? Math.max(1, ...gcValues) : 1
  const step = Math.ceil(maxGC / 4) || 1
  
  // 生成5个刻度
  for (let i = 0; i <= 4; i++) {
    const value = i * step
    const y = topPad + (innerH - (value / (step * 4)) * innerH)
    res.push({ y, label: String(value) })
  }
  
  return res
})

// GC网格线
const gcGrid = computed(() => {
  const lines: string[] = []
  if (samples.value.length === 0) return lines
  
  const innerH = height - topPad - bottomPad
  
  const gcValues = samples.value.map(s => s.gcIncrement ?? 0).filter(v => !isNaN(v))
  const maxGC = gcValues.length > 0 ? Math.max(1, ...gcValues) : 1
  const step = Math.ceil(maxGC / 4) || 1
  
  // 水平网格线
  for (let i = 1; i < 4; i++) {
    const value = i * step
    const y = topPad + (innerH - (value / (step * 4)) * innerH)
    lines.push(`M ${leftPad} ${y} L ${width - rightPad} ${y}`)
  }
  
  return lines
})

// GC数据点
const gcPoints = computed(() => {
  if (samples.value.length === 0) return []
  
  const data = samples.value
  const innerW = width - leftPad - rightPad
  const innerH = height - topPad - bottomPad
  
  const gcValues = data.map(s => s.gcIncrement ?? 0).filter(v => !isNaN(v))
  const maxGC = gcValues.length > 0 ? Math.max(1, ...gcValues) : 1
  
  const res: { x: number; y: number }[] = []
  const stepX = innerW / Math.max(data.length - 1, 1)
  
  for (let i = 0; i < data.length; i++) {
    const gcValue = data[i].gcIncrement ?? 0
    if (isNaN(gcValue)) continue
    const x = leftPad + i * stepX
    const y = topPad + (innerH - (gcValue / maxGC) * innerH)
    res.push({ x, y })
  }
  
  return res
})

// GC路径
const gcPath = computed(() => {
  const pts = gcPoints.value
  if (pts.length < 2) return ''
  
  let d = `M ${pts[0].x} ${pts[0].y}`
  for (let i = 1; i < pts.length; i++) {
    d += ` L ${pts[i].x} ${pts[i].y}`
  }
  return d
})

// GC悬停
const gcHoverIndex = ref<number | null>(null)

function onGCMove(e: MouseEvent) {
  const x = (e as any).offsetX
  const pts = gcPoints.value
  if (!pts.length) {
    gcHoverIndex.value = null
    return
  }
  
  let min = Infinity
  let idx = 0
  for (let i = 0; i < pts.length; i++) {
    const d = Math.abs(pts[i].x - x)
    if (d < min) {
      min = d
      idx = i
    }
  }
  gcHoverIndex.value = idx
}

function onGCLeave() {
  gcHoverIndex.value = null
}

const gcHover = computed(() => {
  if (gcHoverIndex.value == null) return null
  
  const i = gcHoverIndex.value
  const pts = gcPoints.value
  const data = samples.value
  
  if (i >= pts.length || i >= data.length || !pts[i] || !data[i]) return null
  
  const p = pts[i]
  const gcValue = data[i].gcIncrement ?? 0
  if (!data[i].time) return null
  
  const date = new Date(data[i].time)
  if (isNaN(date.getTime())) return null
  
  const timeStr = `${date.getHours()}:${String(date.getMinutes()).padStart(2, '0')}`
  
  return {
    x: p.x,
    y: p.y,
    label: `${timeStr} - GC次数: ${gcValue}`
  }
})
</script>

<style scoped>
</style>

<template>
  <div>
    <el-table :data="rows" style="width: 100%" size="small" v-loading="loading">
      <el-table-column prop="route" label="接口" min-width="240" />
      <el-table-column prop="requests" label="请求数" width="100" />
      <el-table-column label="内存消耗" width="120">
        <template #default="{ row }">
          {{ formatMemory(row.memoryUsage) }}
        </template>
      </el-table-column>
      <el-table-column label="CPU消耗" width="120">
        <template #default="{ row }">
          {{ formatCPU(row.cpuUsage) }}
        </template>
      </el-table-column>
      <el-table-column prop="blockLock" label="锁阻塞" width="100" />
      <el-table-column prop="blockIO" label="IO 阻塞" width="100" />
      <el-table-column prop="blockPerm" label="≥10s 阻塞" width="120" />
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { API_BASE } from '../config'

type Row = {
  route: string
  requests: number
  memoryUsage: number
  cpuUsage: number
  blockLock: number
  blockIO: number
  blockPerm: number
}

function formatMemory(mb: number): string {
  if (!mb || mb === 0) return '0 MB'
  if (mb < 0.01) return '< 0.01 MB'
  return mb.toFixed(2) + ' MB'
}

function formatCPU(ms: number): string {
  if (!ms || ms === 0) return '0 ms'
  if (ms < 0.01) return '< 0.01 ms'
  if (ms < 1) return ms.toFixed(2) + ' ms'
  return ms.toFixed(1) + ' ms'
}

const rows = ref<Row[]>([])
const loading = ref(false)
let timer: number | null = null

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/metrics/routes`)
    const list = await res.json()
    if (Array.isArray(list)) rows.value = list
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 2000)
})

onUnmounted(() => {
  if (timer != null) { window.clearInterval(timer); timer = null }
})
</script>


<template>
  <el-container style="min-height:100vh">
    <el-aside width="220px" style="border-right:1px solid #eee">
      <div style="padding:16px; font-weight:600">Gin 服务实时指标</div>
      <el-menu :default-active="tab" @select="onSelect">
        <el-menu-item index="metrics">协程与请求</el-menu-item>
        <el-menu-item index="memory">堆内存</el-menu-item>
        <el-menu-item index="blocks">阻塞分析</el-menu-item>
      </el-menu>
    </el-aside>
    <el-main>
      <div style="max-width: 900px; margin: 0 auto">
        <MetricsView v-if="tab==='metrics'" />
        <MemoryView v-else-if="tab==='memory'" />
        <BlocksView v-else />
      </div>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import MetricsView from './views/MetricsView.vue'
import MemoryView from './views/MemoryView.vue'
import BlocksView from './views/BlocksView.vue'

const tab = ref<'metrics'|'memory'|'blocks'>('metrics')
function onSelect(i: string) { tab.value = i as 'metrics'|'memory'|'blocks' }
</script>

<style>
body { margin: 0; }
</style>

<template>
  <el-container style="min-height:100vh">
    <el-aside width="220px" style="border-right:1px solid #eee; background: #fff">
      <div style="padding:20px; font-weight:600; font-size:18px; color:#409eff">系统管理</div>
      <el-menu :default-active="tab" @select="onSelect" :default-openeds="['metrics-group', 'blog-group']">
        <el-sub-menu index="metrics-group">
          <template #title>
            <span>性能监控</span>
          </template>
          <el-menu-item index="metrics">协程与请求</el-menu-item>
          <el-menu-item index="memory">堆内存</el-menu-item>
          <el-menu-item index="blocks">阻塞分析</el-menu-item>
        </el-sub-menu>
        <el-sub-menu index="blog-group">
          <template #title>
            <span>博客管理</span>
          </template>
          <el-menu-item index="blog-list">文章列表</el-menu-item>
          <el-menu-item index="blog-create">写文章</el-menu-item>
        </el-sub-menu>
      </el-menu>
    </el-aside>
    <el-main style="background: #f5f7fa">
      <div style="max-width: 1200px; margin: 0 auto">
        <MetricsView v-if="tab==='metrics'" :key="'metrics'" />
        <MemoryView v-else-if="tab==='memory'" :key="'memory'" />
        <BlocksView v-else-if="tab==='blocks'" :key="'blocks'" />
        <BlogListView v-else-if="tab==='blog-list'" :key="'blog-list'" />
        <BlogEditView v-else-if="tab==='blog-create'" :key="'blog-create'" />
        <MetricsView v-else :key="'default'" />
      </div>
    </el-main>
  </el-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import MetricsView from './views/MetricsView.vue'
import MemoryView from './views/MemoryView.vue'
import BlocksView from './views/BlocksView.vue'
import BlogListView from './views/BlogListView.vue'
import BlogEditView from './views/BlogEditView.vue'

const tab = ref<string>('metrics')
function onSelect(i: string) { 
  tab.value = i
  window.location.hash = `#${i}`
}

// 监听hash变化和自定义事件
onMounted(() => {
  const hash = window.location.hash.replace('#', '')
  if (hash) {
    if (hash.startsWith('blog-')) {
      tab.value = hash.includes('edit') ? 'blog-create' : hash
    } else {
      tab.value = hash || 'metrics'
    }
  }

  window.addEventListener('switch-tab', ((e: CustomEvent) => {
    tab.value = e.detail
  }) as EventListener)
})
</script>

<style>
body { margin: 0; }
</style>

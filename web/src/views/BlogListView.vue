<template>
  <div>
    <el-card shadow="never" style="border-radius: 8px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span style="font-size: 18px; font-weight: 600">文章列表</span>
          <el-button type="primary" @click="handleCreate">
            ✏️ 写文章
          </el-button>
        </div>
      </template>

      <!-- 搜索栏 -->
      <div style="margin-bottom: 20px">
        <el-row :gutter="16">
          <el-col :span="8">
            <el-input
              v-model="searchForm.keyword"
              placeholder="🔍 搜索文章标题或内容"
              clearable
              @clear="loadPosts"
              @keyup.enter="loadPosts"
            />
          </el-col>
          <el-col :span="4">
            <el-select v-model="searchForm.status" placeholder="状态" clearable @change="loadPosts" style="width: 100%">
              <el-option label="草稿" value="draft" />
              <el-option label="已发布" value="published" />
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-select v-model="searchForm.categoryId" placeholder="分类" clearable @change="loadPosts" style="width: 100%">
              <el-option
                v-for="cat in categories"
                :key="cat.id"
                :label="cat.name"
                :value="cat.id"
              />
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-button type="primary" @click="loadPosts">搜索</el-button>
          </el-col>
        </el-row>
      </div>

      <!-- 文章列表 -->
      <el-table :data="posts" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="封面" width="100">
          <template #default="{ row }">
            <el-image
              v-if="row.coverImage"
              :src="row.coverImage"
              style="width: 60px; height: 60px; border-radius: 4px"
              fit="cover"
            />
            <div v-else style="width: 60px; height: 60px; background: #f0f0f0; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 20px">
              🖼️
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="200" />
        <el-table-column label="分类" width="120">
          <template #default="{ row }">
            <el-tag v-if="row.category" type="info" size="small">{{ row.category.name }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="标签" width="200">
          <template #default="{ row }">
            <el-tag
              v-for="tag in row.tags"
              :key="tag.id"
              size="small"
              style="margin-right: 4px"
            >
              {{ tag.name }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'published' ? 'success' : 'info'" size="small">
              {{ row.status === 'published' ? '已发布' : '草稿' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="views" label="浏览量" width="100" />
        <el-table-column prop="createdAt" label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.createdAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleView(row)">查看</el-button>
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div style="margin-top: 20px; display: flex; justify-content: flex-end">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadPosts"
          @current-change="loadPosts"
        />
      </div>
    </el-card>

    <!-- 文章详情对话框 -->
    <el-dialog v-model="detailVisible" title="文章详情" width="800px">
      <div v-if="currentPost">
        <h2 style="margin-top: 0">{{ currentPost.title }}</h2>
        <div style="margin: 16px 0; color: #666; font-size: 14px">
          <el-tag v-if="currentPost.category" type="info" size="small" style="margin-right: 8px">
            {{ currentPost.category.name }}
          </el-tag>
          <el-tag
            v-for="tag in currentPost.tags"
            :key="tag.id"
            size="small"
            style="margin-right: 8px"
          >
            {{ tag.name }}
          </el-tag>
          <span style="margin-left: 16px">浏览量: {{ currentPost.views }}</span>
          <span style="margin-left: 16px">{{ formatDate(currentPost.createdAt) }}</span>
        </div>
        <el-divider />
        <div v-html="formatContent(currentPost.content)" style="line-height: 1.8"></div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
// 使用简单的图标替代
import { API_BASE } from '../config'

type Post = {
  id: number
  title: string
  content: string
  excerpt: string
  coverImage: string
  status: string
  views: number
  createdAt: string
  category?: { id: number; name: string }
  tags?: Array<{ id: number; name: string }>
}

type Category = {
  id: number
  name: string
}

const posts = ref<Post[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const detailVisible = ref(false)
const currentPost = ref<Post | null>(null)

const searchForm = ref({
  keyword: '',
  status: '',
  categoryId: undefined as number | undefined
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0
})

async function loadPosts() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: pagination.value.page.toString(),
      pageSize: pagination.value.pageSize.toString()
    })
    if (searchForm.value.keyword) params.append('keyword', searchForm.value.keyword)
    if (searchForm.value.status) params.append('status', searchForm.value.status)
    if (searchForm.value.categoryId) params.append('categoryId', searchForm.value.categoryId.toString())

    const res = await fetch(`${API_BASE}/api/blog/posts?${params}`)
    const data = await res.json()
    posts.value = data.data || []
    pagination.value.total = data.total || 0
  } catch (error) {
    ElMessage.error('加载文章列表失败')
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  try {
    const res = await fetch(`${API_BASE}/api/blog/categories`)
    categories.value = await res.json()
  } catch (error) {
    console.error('Failed to load categories:', error)
  }
}

function handleCreate() {
  // 通过事件通知父组件切换
  window.dispatchEvent(new CustomEvent('switch-tab', { detail: 'blog-create' }))
  // 更新hash
  window.location.hash = '#blog-create'
}

function handleView(post: Post) {
  currentPost.value = post
  detailVisible.value = true
}

function handleEdit(post: Post) {
  // 跳转到编辑页面，传递文章ID
  window.dispatchEvent(new CustomEvent('switch-tab', { detail: 'blog-create' }))
  window.location.hash = `#blog-edit-${post.id}`
  // 延迟加载文章数据，等待组件切换完成
  setTimeout(() => {
    window.dispatchEvent(new CustomEvent('load-post', { detail: post.id }))
  }, 100)
}

async function handleDelete(post: Post) {
  try {
    await ElMessageBox.confirm(`确定要删除文章《${post.title}》吗？`, '提示', {
      type: 'warning'
    })
    const res = await fetch(`${API_BASE}/api/blog/posts/${post.id}`, {
      method: 'DELETE'
    })
    if (res.ok) {
      ElMessage.success('删除成功')
      loadPosts()
    } else {
      ElMessage.error('删除失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
    }
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString('zh-CN')
}

function formatContent(content: string) {
  // 简单的换行处理，实际应该使用 markdown 解析器
  return content.replace(/\n/g, '<br>')
}

onMounted(() => {
  loadPosts()
  loadCategories()
})
</script>

<style scoped>
:deep(.el-card__header) {
  padding: 20px;
  border-bottom: 1px solid #ebeef5;
}
</style>


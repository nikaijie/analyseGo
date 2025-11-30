<template>
  <div>
    <el-card shadow="never" style="border-radius: 8px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span style="font-size: 18px; font-weight: 600">{{ isEdit ? '编辑文章' : '写文章' }}</span>
          <div>
            <el-button @click="handleCancel">取消</el-button>
            <el-button type="primary" @click="handleSave" :loading="saving">保存</el-button>
            <el-button type="success" @click="handlePublish" :loading="saving">发布</el-button>
          </div>
        </div>
      </template>

      <el-form :model="form" label-width="100px" style="max-width: 1000px">
        <el-form-item label="文章标题" required>
          <el-input v-model="form.title" placeholder="请输入文章标题" />
        </el-form-item>

        <el-form-item label="URL Slug" required>
          <el-input v-model="form.slug" placeholder="例如: my-first-post" />
          <div style="font-size: 12px; color: #999; margin-top: 4px">
            用于生成文章URL，只能包含字母、数字和连字符
          </div>
        </el-form-item>

        <el-form-item label="封面图片">
          <div style="display: flex; gap: 8px; width: 100%">
            <el-input
              v-model="form.coverImage"
              placeholder="输入图片URL（支持Bing图片搜索链接自动提取）"
              @blur="handleImageUrlBlur"
            />
            <el-button @click="extractImageUrl" :loading="extracting">
              提取图片
            </el-button>
          </div>
          <div style="font-size: 12px; color: #999; margin-top: 4px">
            提示：如果输入的是Bing图片搜索链接，点击"提取图片"按钮可自动提取真实图片URL
          </div>
          <div v-if="form.coverImage" style="margin-top: 10px">
            <el-image
              :src="form.coverImage"
              style="width: 200px; height: 120px; border-radius: 4px"
              fit="cover"
              :preview-src-list="[form.coverImage]"
              :hide-on-click-modal="true"
              @error="handleImageError"
              @load="handleImageLoad"
            >
              <template #error>
                <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; width: 100%; height: 100%; background: #f5f7fa; color: #999; padding: 10px">
                  <span style="font-size: 24px; margin-bottom: 8px">🖼️</span>
                  <span style="font-size: 12px; text-align: center">图片加载失败</span>
                  <span style="font-size: 11px; color: #bbb; margin-top: 4px; word-break: break-all; max-width: 180px">{{ form.coverImage.substring(0, 30) }}...</span>
                </div>
              </template>
            </el-image>
            <div v-if="imageLoadStatus" style="margin-top: 8px; font-size: 12px" :style="{ color: imageLoadStatus === 'success' ? '#67c23a' : '#f56c6c' }">
              {{ imageLoadStatus === 'success' ? '✓ 图片加载成功' : '✗ 图片加载失败，请检查URL是否正确' }}
            </div>
          </div>
        </el-form-item>

        <el-form-item label="分类">
          <el-select v-model="form.categoryId" placeholder="选择分类" clearable style="width: 300px">
            <el-option
              v-for="cat in categories"
              :key="cat.id"
              :label="cat.name"
              :value="cat.id"
            />
          </el-select>
          <el-button link type="primary" @click="showCategoryDialog = true" style="margin-left: 10px">
            + 新建分类
          </el-button>
        </el-form-item>

        <el-form-item label="标签">
          <el-select
            v-model="form.tagIds"
            multiple
            placeholder="选择标签"
            style="width: 500px"
          >
            <el-option
              v-for="tag in tags"
              :key="tag.id"
              :label="tag.name"
              :value="tag.id"
            />
          </el-select>
          <el-button link type="primary" @click="showTagDialog = true" style="margin-left: 10px">
            + 新建标签
          </el-button>
        </el-form-item>

        <el-form-item label="摘要">
          <el-input
            v-model="form.excerpt"
            type="textarea"
            :rows="3"
            placeholder="文章摘要，用于列表页显示"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="正文内容" required>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="20"
            placeholder="请输入文章内容（支持Markdown）"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 新建分类对话框 -->
    <el-dialog v-model="showCategoryDialog" title="新建分类" width="400px">
      <el-form :model="categoryForm" label-width="80px">
        <el-form-item label="分类名称" required>
          <el-input v-model="categoryForm.name" placeholder="例如: 技术" />
        </el-form-item>
        <el-form-item label="Slug" required>
          <el-input v-model="categoryForm.slug" placeholder="例如: tech" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCategoryDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateCategory">确定</el-button>
      </template>
    </el-dialog>

    <!-- 新建标签对话框 -->
    <el-dialog v-model="showTagDialog" title="新建标签" width="400px">
      <el-form :model="tagForm" label-width="80px">
        <el-form-item label="标签名称" required>
          <el-input v-model="tagForm.name" placeholder="例如: Go语言" />
        </el-form-item>
        <el-form-item label="Slug" required>
          <el-input v-model="tagForm.slug" placeholder="例如: golang" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showTagDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreateTag">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { API_BASE } from '../config'

type Category = { id: number; name: string; slug: string }
type Tag = { id: number; name: string; slug: string }

const form = ref({
  title: '',
  slug: '',
  content: '',
  excerpt: '',
  coverImage: '',
  categoryId: undefined as number | undefined,
  tagIds: [] as number[]
})

const categories = ref<Category[]>([])
const tags = ref<Tag[]>([])
const saving = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const showCategoryDialog = ref(false)
const showTagDialog = ref(false)
const extracting = ref(false)
const imageLoadStatus = ref<'success' | 'error' | null>(null)

const categoryForm = ref({ name: '', slug: '' })
const tagForm = ref({ name: '', slug: '' })

async function loadCategories() {
  try {
    const res = await fetch(`${API_BASE}/api/blog/categories`)
    categories.value = await res.json()
  } catch (error) {
    ElMessage.error('加载分类失败')
  }
}

async function loadTags() {
  try {
    const res = await fetch(`${API_BASE}/api/blog/tags`)
    tags.value = await res.json()
  } catch (error) {
    ElMessage.error('加载标签失败')
  }
}

async function loadPost(id: number) {
  try {
    const res = await fetch(`${API_BASE}/api/blog/posts/${id}`)
    const post = await res.json()
    form.value = {
      title: post.title,
      slug: post.slug,
      content: post.content,
      excerpt: post.excerpt || '',
      coverImage: post.coverImage || '',
      categoryId: post.categoryId,
      tagIds: post.tags?.map((t: Tag) => t.id) || []
    }
  } catch (error) {
    ElMessage.error('加载文章失败')
  }
}

async function handleSave() {
  if (!form.value.title || !form.value.slug || !form.value.content) {
    ElMessage.warning('请填写标题、Slug和内容')
    return
  }

  saving.value = true
  try {
    const url = isEdit.value
      ? `${API_BASE}/api/blog/posts/${editId.value}`
      : `${API_BASE}/api/blog/posts`
    const method = isEdit.value ? 'PUT' : 'POST'

    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...form.value,
        status: 'draft'
      })
    })

    if (res.ok) {
      ElMessage.success('保存成功')
      if (!isEdit.value) {
        const data = await res.json()
        editId.value = data.id
        isEdit.value = true
      }
    } else {
      ElMessage.error('保存失败')
    }
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

async function handlePublish() {
  if (!form.value.title || !form.value.slug || !form.value.content) {
    ElMessage.warning('请填写标题、Slug和内容')
    return
  }

  saving.value = true
  try {
    const url = isEdit.value
      ? `${API_BASE}/api/blog/posts/${editId.value}`
      : `${API_BASE}/api/blog/posts`
    const method = isEdit.value ? 'PUT' : 'POST'

    const res = await fetch(url, {
      method,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ...form.value,
        status: 'published'
      })
    })

    if (res.ok) {
      ElMessage.success('发布成功')
      if (!isEdit.value) {
        const data = await res.json()
        editId.value = data.id
        isEdit.value = true
      }
      // 跳转到列表页
      setTimeout(() => {
        window.location.hash = '#blog-list'
        const event = new CustomEvent('switch-tab', { detail: 'blog-list' })
        window.dispatchEvent(event)
      }, 1000)
    } else {
      ElMessage.error('发布失败')
    }
  } catch (error) {
    ElMessage.error('发布失败')
  } finally {
    saving.value = false
  }
}

function handleCancel() {
  window.location.hash = '#blog-list'
  const event = new CustomEvent('switch-tab', { detail: 'blog-list' })
  window.dispatchEvent(event)
}

// 从Bing图片搜索URL中提取真实图片URL
function extractImageUrlFromBing(url: string): string | null {
  try {
    console.log('Extracting from URL:', url)
    
    // 解析URL
    const urlObj = new URL(url)
    
    // 方法1: 从mediaurl参数中提取
    const mediaurl = urlObj.searchParams.get('mediaurl')
    if (mediaurl) {
      console.log('Found mediaurl param:', mediaurl)
      // 需要多次解码，因为Bing的URL是双重编码的
      let decodedUrl = decodeURIComponent(mediaurl)
      console.log('First decode:', decodedUrl)
      
      // 尝试再次解码（处理双重编码的情况）
      try {
        decodedUrl = decodeURIComponent(decodedUrl)
        console.log('Second decode:', decodedUrl)
      } catch (e) {
        // 如果第二次解码失败，使用第一次解码的结果
      }
      
      // 从mediaurl中提取实际的图片URL（通常在riu参数中）
      try {
        const mediaUrlObj = new URL(decodedUrl)
        const riu = mediaUrlObj.searchParams.get('riu')
        if (riu) {
          let finalUrl = decodeURIComponent(riu)
          console.log('Extracted riu:', finalUrl)
          // 可能需要再次解码
          try {
            finalUrl = decodeURIComponent(finalUrl)
            console.log('Final URL after double decode:', finalUrl)
          } catch (e) {}
          return finalUrl
        }
      } catch (e) {
        // 如果无法解析为URL，尝试正则匹配
        console.log('Trying regex match on decodedUrl')
        const riuMatch = decodedUrl.match(/riu=([^&]+)/)
        if (riuMatch) {
          let finalUrl = decodeURIComponent(riuMatch[1])
          try {
            finalUrl = decodeURIComponent(finalUrl) // 双重解码
          } catch (e) {}
          console.log('Extracted via regex:', finalUrl)
          return finalUrl
        }
      }
      
      // 如果没有riu，检查是否是直接的图片URL
      if (decodedUrl.match(/\.(jpg|jpeg|png|gif|webp|bmp)(\?|$)/i)) {
        return decodedUrl.split('?')[0]
      }
    }
    
    // 方法2: 直接从URL字符串中匹配（处理编码的情况）
    const mediaurlMatch = url.match(/mediaurl=([^&]+)/)
    if (mediaurlMatch) {
      let extracted = mediaurlMatch[1]
      // 多次解码
      try {
        extracted = decodeURIComponent(extracted)
        extracted = decodeURIComponent(extracted) // 双重解码
      } catch (e) {}
      
      // 检查是否包含riu参数
      const riuMatch = extracted.match(/riu=([^&]+)/)
      if (riuMatch) {
        let finalUrl = riuMatch[1]
        try {
          finalUrl = decodeURIComponent(finalUrl)
          finalUrl = decodeURIComponent(finalUrl) // 双重解码
        } catch (e) {}
        console.log('Extracted via string match:', finalUrl)
        return finalUrl
      }
      
      // 如果本身就是图片URL
      if (extracted.match(/\.(jpg|jpeg|png|gif|webp|bmp)/i)) {
        return extracted.split('?')[0]
      }
    }
    
    console.log('Failed to extract image URL')
    return null
  } catch (error) {
    console.error('Failed to extract image URL:', error)
    return null
  }
}

// 提取图片URL
async function extractImageUrl() {
  if (!form.value.coverImage) {
    ElMessage.warning('请先输入图片URL')
    return
  }

  extracting.value = true
  try {
    // 检查是否是Bing图片搜索链接
    if (form.value.coverImage.includes('bing.com/images/search')) {
      const extractedUrl = extractImageUrlFromBing(form.value.coverImage)
      if (extractedUrl) {
        form.value.coverImage = extractedUrl
        ElMessage.success(`图片URL提取成功: ${extractedUrl.substring(0, 50)}...`)
        console.log('Extracted URL:', extractedUrl)
      } else {
        ElMessage.warning('无法从该链接中提取图片URL，请直接使用图片的直链地址')
      }
    } else {
      // 验证URL是否有效（不进行实际请求，只检查格式）
      try {
        new URL(form.value.coverImage)
        ElMessage.success('图片URL格式正确')
      } catch (error) {
        ElMessage.warning('图片URL格式不正确，请检查链接是否正确')
      }
    }
  } catch (error) {
    console.error('Extract error:', error)
    ElMessage.error('提取图片URL失败: ' + (error as Error).message)
  } finally {
    extracting.value = false
  }
}

// 当图片URL输入框失去焦点时自动尝试提取
function handleImageUrlBlur() {
  imageLoadStatus.value = null // 重置状态
  if (form.value.coverImage && form.value.coverImage.includes('bing.com/images/search')) {
    const extractedUrl = extractImageUrlFromBing(form.value.coverImage)
    if (extractedUrl && extractedUrl !== form.value.coverImage) {
      // 询问用户是否要替换
      setTimeout(() => {
        if (confirm('检测到Bing图片搜索链接，是否自动提取真实图片URL？\n\n提取的URL: ' + extractedUrl.substring(0, 80) + '...')) {
          form.value.coverImage = extractedUrl
          imageLoadStatus.value = null // 重置状态，等待新图片加载
          ElMessage.success('已自动提取图片URL')
        }
      }, 100)
    } else if (!extractedUrl) {
      ElMessage.warning('无法自动提取图片URL，请手动点击"提取图片"按钮或使用图片直链')
    }
  }
}

// 图片加载成功
function handleImageLoad() {
  imageLoadStatus.value = 'success'
  console.log('Image loaded successfully:', form.value.coverImage)
}

// 图片加载失败
function handleImageError() {
  imageLoadStatus.value = 'error'
  console.error('Image load failed:', form.value.coverImage)
}

async function handleCreateCategory() {
  if (!categoryForm.value.name || !categoryForm.value.slug) {
    ElMessage.warning('请填写分类名称和Slug')
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/blog/categories`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(categoryForm.value)
    })
    if (res.ok) {
      ElMessage.success('创建成功')
      showCategoryDialog.value = false
      categoryForm.value = { name: '', slug: '' }
      loadCategories()
    } else {
      ElMessage.error('创建失败')
    }
  } catch (error) {
    ElMessage.error('创建失败')
  }
}

async function handleCreateTag() {
  if (!tagForm.value.name || !tagForm.value.slug) {
    ElMessage.warning('请填写标签名称和Slug')
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/blog/tags`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(tagForm.value)
    })
    if (res.ok) {
      ElMessage.success('创建成功')
      showTagDialog.value = false
      tagForm.value = { name: '', slug: '' }
      loadTags()
    } else {
      ElMessage.error('创建失败')
    }
  } catch (error) {
    ElMessage.error('创建失败')
  }
}

// 监听hash变化，支持编辑功能
watch(() => window.location.hash, (hash) => {
  const match = hash.match(/blog-edit-(\d+)/)
  if (match) {
    const id = parseInt(match[1])
    isEdit.value = true
    editId.value = id
    loadPost(id)
  } else if (hash === '#blog-create' || hash === '') {
    isEdit.value = false
    editId.value = null
    form.value = {
      title: '',
      slug: '',
      content: '',
      excerpt: '',
      coverImage: '',
      categoryId: undefined,
      tagIds: []
    }
  }
}, { immediate: true })

// 监听加载文章事件
onMounted(() => {
  window.addEventListener('load-post', ((e: CustomEvent) => {
    const id = e.detail
    isEdit.value = true
    editId.value = id
    loadPost(id)
  }) as EventListener)
})

onMounted(() => {
  loadCategories()
  loadTags()
})
</script>

<style scoped>
:deep(.el-card__header) {
  padding: 20px;
  border-bottom: 1px solid #ebeef5;
}
</style>


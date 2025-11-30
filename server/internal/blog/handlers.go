package blog

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetPosts 获取文章列表
func GetPosts(c *gin.Context) {
	var posts []Post
	query := GetDB().Preload("Category").Preload("Tags")

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 分类筛选
	if categoryID := c.Query("categoryId"); categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * pageSize

	var total int64
	query.Model(&Post{}).Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts)

	c.JSON(http.StatusOK, gin.H{
		"data":     posts,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetPost 获取单篇文章
func GetPost(c *gin.Context) {
	id := c.Param("id")
	var post Post
	if err := GetDB().Preload("Category").Preload("Tags").First(&post, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 增加浏览量
	GetDB().Model(&post).Update("views", gorm.Expr("views + ?", 1))

	c.JSON(http.StatusOK, post)
}

// CreatePost 创建文章
func CreatePost(c *gin.Context) {
	var req PostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	post := Post{
		Title:      req.Title,
		Slug:       req.Slug,
		Content:    req.Content,
		Excerpt:    req.Excerpt,
		CoverImage: req.CoverImage,
		Status:     req.Status,
		CategoryID: req.CategoryID,
	}

	if post.Status == "" {
		post.Status = "draft"
	}

	// 关联标签
	if len(req.TagIDs) > 0 {
		var tags []Tag
		GetDB().Where("id IN ?", req.TagIDs).Find(&tags)
		post.Tags = tags
	}

	if err := GetDB().Create(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	GetDB().Preload("Category").Preload("Tags").First(&post, post.ID)
	c.JSON(http.StatusOK, post)
}

// UpdatePost 更新文章
func UpdatePost(c *gin.Context) {
	id := c.Param("id")
	var post Post
	if err := GetDB().First(&post, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req PostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新字段
	if req.Title != "" {
		post.Title = req.Title
	}
	if req.Slug != "" {
		post.Slug = req.Slug
	}
	if req.Content != "" {
		post.Content = req.Content
	}
	if req.Excerpt != "" {
		post.Excerpt = req.Excerpt
	}
	if req.CoverImage != "" {
		post.CoverImage = req.CoverImage
	}
	if req.Status != "" {
		post.Status = req.Status
	}
	if req.CategoryID > 0 {
		post.CategoryID = req.CategoryID
	}

	// 更新标签
	if req.TagIDs != nil {
		var tags []Tag
		GetDB().Where("id IN ?", req.TagIDs).Find(&tags)
		GetDB().Model(&post).Association("Tags").Replace(tags)
	}

	if err := GetDB().Save(&post).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	GetDB().Preload("Category").Preload("Tags").First(&post, post.ID)
	c.JSON(http.StatusOK, post)
}

// DeletePost 删除文章
func DeletePost(c *gin.Context) {
	id := c.Param("id")
	if err := GetDB().Delete(&Post{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}

// GetCategories 获取分类列表
func GetCategories(c *gin.Context) {
	var categories []Category
	GetDB().Find(&categories)
	c.JSON(http.StatusOK, categories)
}

// CreateCategory 创建分类
func CreateCategory(c *gin.Context) {
	var req CategoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category := Category{
		Name: req.Name,
		Slug: req.Slug,
	}

	if err := GetDB().Create(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// GetTags 获取标签列表
func GetTags(c *gin.Context) {
	var tags []Tag
	GetDB().Find(&tags)
	c.JSON(http.StatusOK, tags)
}

// CreateTag 创建标签
func CreateTag(c *gin.Context) {
	var req TagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag := Tag{
		Name: req.Name,
		Slug: req.Slug,
	}

	if err := GetDB().Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tag)
}

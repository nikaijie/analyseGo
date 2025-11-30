package blog

import (
	"time"

	"gorm.io/gorm"
)

// Category 分类模型
type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Slug      string         `gorm:"type:varchar(50);not null;uniqueIndex" json:"slug"`
	Posts     []Post         `gorm:"foreignKey:CategoryID" json:"posts,omitempty"`
}

// Tag 标签模型
type Tag struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name      string         `gorm:"type:varchar(30);not null;uniqueIndex" json:"name"`
	Slug      string         `gorm:"type:varchar(30);not null;uniqueIndex" json:"slug"`
	Posts     []Post         `gorm:"many2many:post_tags;" json:"posts,omitempty"`
}

// Post 文章模型
type Post struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Title      string         `gorm:"type:varchar(200);not null" json:"title"`
	Slug       string         `gorm:"type:varchar(200);not null;uniqueIndex" json:"slug"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	Excerpt    string         `gorm:"type:varchar(500)" json:"excerpt"`
	CoverImage string         `gorm:"type:varchar(500)" json:"coverImage"`
	Status     string         `gorm:"type:varchar(20);default:'draft'" json:"status"` // draft, published
	Views      int            `gorm:"default:0" json:"views"`
	CategoryID uint           `json:"categoryId"`
	Category   Category       `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Tags       []Tag          `gorm:"many2many:post_tags;" json:"tags,omitempty"`
}

// PostCreateRequest 创建文章请求
type PostCreateRequest struct {
	Title      string `json:"title" binding:"required"`
	Slug       string `json:"slug" binding:"required"`
	Content    string `json:"content" binding:"required"`
	Excerpt    string `json:"excerpt"`
	CoverImage string `json:"coverImage"`
	Status     string `json:"status"`
	CategoryID uint   `json:"categoryId"`
	TagIDs     []uint `json:"tagIds"`
}

// PostUpdateRequest 更新文章请求
type PostUpdateRequest struct {
	Title      string `json:"title"`
	Slug       string `json:"slug"`
	Content    string `json:"content"`
	Excerpt    string `json:"excerpt"`
	CoverImage string `json:"coverImage"`
	Status     string `json:"status"`
	CategoryID uint   `json:"categoryId"`
	TagIDs     []uint `json:"tagIds"`
}

// CategoryCreateRequest 创建分类请求
type CategoryCreateRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

// TagCreateRequest 创建标签请求
type TagCreateRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

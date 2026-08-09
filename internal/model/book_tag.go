package model

import (
	"errors"
	"sync"
	"time"
)

type SingleBookTag string

var (
	// 书籍标签，需要初始化
	singleBookTagList []string

	onceInit = sync.Once{}
)

func (s SingleBookTag) Tag() string {
	for _, tagItem := range singleBookTagList {
		if tagItem == string(s) {
			return string(s)
		}
	}

	return ""
}

func StringToSingleBookTag(s string) (SingleBookTag, error) {
	for _, tagItem := range singleBookTagList {
		if string(tagItem) == s {
			return SingleBookTag(tagItem), nil
		}
	}
	return "", errors.New("no tags match")
}

func InitBookTags(tags []string) {
	onceInit.Do(func() {
		singleBookTagList = tags
	})
}

type BookTag struct {
	ID        string        `json:"id" gorm:"primary_key;type:varchar(36)"`
	BookID    string        `json:"book_id" gorm:"type:varchar(36);index"`
	Tag       SingleBookTag `json:"tag" gorm:"type:varchar(255)"`
	CreatedAt time.Time     `json:"created_at"`
}

func (BookTag) TableName() string {
	return "book_tag"
}

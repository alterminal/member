package model

import "gorm.io/gorm"

type Pagination[T any] struct {
	Items []T `json:"items"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

type Option func(*gorm.DB) *gorm.DB

func ListByOption[T any](db *gorm.DB, limit, page int, opts ...Option) (Pagination[*T], error) {
	var ts []*T
	db = db.Model(&ts)
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		db = opt(db)
	}
	var count int64 = 0
	db.Count(&count)
	db = db.Limit(limit).Offset(page * limit)
	db.Find(&ts)
	pagination := Pagination[*T]{
		Items: ts,
		Page:  page,
		Pages: int(count) / limit,
		Limit: limit,
		Total: int(count),
	}
	return pagination, nil
}

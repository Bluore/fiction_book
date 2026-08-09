package model

import "time"

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type IDPage struct {
	LastID *string `json:"last_id"` // 分页查询时，上一页的最后一个ID
	Limit  int     `json:"limit"`   // 分页查询时，每页数量
}

type TimePage struct {
	LastTime time.Time `json:"last_time"` // 分页查询时，上一页的最后一个时间
	LastID   *string   `json:"last_id"`   // 分页查询时，上一页的最后一个ID
	Page     int       `json:"page"`      // 分页查询时，当前页码
	Limit    int       `json:"limit"`     // 分页查询时，每页数量
}

package models

import advModels "api-tests-template/internal/managers/advertisement/models"

// Favorite элемент избранного (одинаков в ответе POST /favorites и в элементах GET /favorites)
type Favorite struct {
	AddedAt       string                  `json:"added_at,omitempty"`
	Advertisement advModels.Advertisement `json:"advertisement,omitempty"`
	FavoriteId    string                  `json:"favorite_id,omitempty"`
}

// FavoritesList ответ GET /favorites (пагинация page/page_size)
type FavoritesList struct {
	Items      []Favorite `json:"items"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	Total      int        `json:"total"`
	TotalPages int        `json:"total_pages"`
}

// BulkAddFavoritesResult ответ POST /favorites/bulk
type BulkAddFavoritesResult struct {
	Added        []string `json:"added"`
	LimitReached bool     `json:"limit_reached"`
	NotFound     []string `json:"not_found"`
	Skipped      []string `json:"skipped"`
}

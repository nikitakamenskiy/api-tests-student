package models

type AddFavoriteRequest struct {
	AdvertisementId string `json:"advertisement_id"`
}

type BulkAddFavoritesRequest struct {
	AdvertisementIds []string `json:"advertisement_ids"`
}

type ListQuery struct {
	Page      string
	PageSize  string
	SortBy    string
	SortOrder string
	Search    string
	Quantity  string
}

package favorites

import (
	"net/http"
	"strings"
	"testing"

	"api-tests-template/internal/constants/path"
	apiRunner "api-tests-template/internal/helpers/api-runner"
	"api-tests-template/internal/managers/favorites/models"
)

func HttpGetFavorites(t *testing.T, token string, statusCode int, query models.ListQuery) *http.Response {
	request := apiRunner.GetRunner().Auth(token).Create().Get(path.FavoritesPath)

	for key, value := range map[string]string{
		"page":       query.Page,
		"page_size":  query.PageSize,
		"sort_by":    query.SortBy,
		"sort_order": query.SortOrder,
		"search":     query.Search,
		"quantity":   query.Quantity,
	} {
		if value != "" {
			request = request.Query(key, value)
		}
	}
	return request.
		Expect(t).
		Status(statusCode).
		End().Response
}

func HttpPostFavorite(t *testing.T, token string, statusCode int, request models.AddFavoriteRequest) *http.Response {
	return apiRunner.GetRunner().Auth(token).Create().Post(path.FavoritesPath).
		ContentType("application/json").
		JSON(request).
		Expect(t).
		Status(statusCode).
		End().Response
}

func HttpPostFavoritesBulk(t *testing.T, token string, statusCode int, request models.BulkAddFavoritesRequest) *http.Response {
	return apiRunner.GetRunner().Auth(token).Create().Post(path.FavoritesBulkPath).
		ContentType("application/json").
		JSON(request).
		Expect(t).
		Status(statusCode).
		End().Response
}

func HttpDeleteFavorite(t *testing.T, token string, statusCode int, adID string) *http.Response {
	return apiRunner.GetRunner().Auth(token).Create().
		Delete(strings.ReplaceAll(path.FavoritesByAdPath, "{ad_id}", adID)).
		Expect(t).
		Status(statusCode).
		End().Response
}

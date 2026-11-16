package favorites

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"api-tests-template/internal/client/http/favorites"
	advModels "api-tests-template/internal/managers/advertisement/models"
	"api-tests-template/internal/managers/favorites/models"
)

// GetFavorites возвращает страницу избранного пользователя и десериализованный ответ
func GetFavorites(t *testing.T, token string, query models.ListQuery) models.FavoritesList {
	rawResponse := favorites.HttpGetFavorites(t, token, http.StatusOK, query)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	result := models.FavoritesList{}
	err := json.Unmarshal(respArray, &result)
	require.NoError(t, err)

	return result
}

// GetFavoritesError запрашивает список избранного и возвращает ошибку с указанным статусом
func GetFavoritesError(t *testing.T, token string, statusCode int, query models.ListQuery) advModels.CreateErrorResponse {
	rawResponse := favorites.HttpGetFavorites(t, token, statusCode, query)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	errResp := advModels.CreateErrorResponse{}
	err := json.Unmarshal(respArray, &errResp)
	require.NoError(t, err)

	return errResp
}
func AddFavorite(t *testing.T, token string, request models.AddFavoriteRequest) models.Favorite {
	rawResponse := favorites.HttpPostFavorite(t, token, http.StatusCreated, request)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	result := models.Favorite{}
	err := json.Unmarshal(respArray, &result)
	require.NoError(t, err)

	return result
}

func GetFavoritesErrorBadRequest(t *testing.T, token string, query models.ListQuery) advModels.CreateErrorResponse {
	return GetFavoritesError(t, token, http.StatusBadRequest, query)
}

func GetFavoritesErrorUnauthorized(t *testing.T, query models.ListQuery) advModels.CreateErrorResponse {
	return GetFavoritesError(t, "invalid_token", http.StatusUnauthorized, query)
}

func GetFavoritesErrorNoToken(t *testing.T, query models.ListQuery) advModels.CreateErrorResponse {
	return GetFavoritesError(t, "", http.StatusUnauthorized, query)
}

func GetFavoritesErrorEmptyBearer(t *testing.T, query models.ListQuery) advModels.CreateErrorResponse {
	return GetFavoritesError(t, " ", http.StatusUnauthorized, query)
}

func AddFavoriteError(t *testing.T, token string, statusCode int, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	rawResponse := favorites.HttpPostFavorite(t, token, statusCode, request)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	errResp := advModels.CreateErrorResponse{}
	err := json.Unmarshal(respArray, &errResp)
	require.NoError(t, err)

	return errResp
}

func AddFavoriteErrorBadRequest(t *testing.T, token string, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, token, http.StatusBadRequest, request)
}

func AddFavoriteErrorForbidden(t *testing.T, token string, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, token, http.StatusForbidden, request)
}

func AddFavoriteErrorNotFound(t *testing.T, token string, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, token, http.StatusNotFound, request)
}

func AddFavoriteErrorConflict(t *testing.T, token string, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, token, http.StatusConflict, request)
}

func AddFavoriteErrorUnauthorized(t *testing.T, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, "invalid_token", http.StatusUnauthorized, request)
}

func AddFavoriteErrorNoToken(t *testing.T, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, "", http.StatusUnauthorized, request)
}

func AddFavoriteErrorEmptyBearer(t *testing.T, request models.AddFavoriteRequest) advModels.CreateErrorResponse {
	return AddFavoriteError(t, " ", http.StatusUnauthorized, request)
}

func DeleteFavorite(t *testing.T, token, adID string) {
	rawResponse := favorites.HttpDeleteFavorite(t, token, http.StatusNoContent, adID)
	defer rawResponse.Body.Close()
}

func DeleteFavoriteError(t *testing.T, token string, statusCode int, adID string) advModels.CreateErrorResponse {
	rawResponse := favorites.HttpDeleteFavorite(t, token, statusCode, adID)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	errResp := advModels.CreateErrorResponse{}
	err := json.Unmarshal(respArray, &errResp)
	require.NoError(t, err)

	return errResp
}

func DeleteFavoriteErrorBadRequest(t *testing.T, token, adID string) advModels.CreateErrorResponse {
	return DeleteFavoriteError(t, token, http.StatusBadRequest, adID)
}

func DeleteFavoriteErrorNotFound(t *testing.T, token, adID string) advModels.CreateErrorResponse {
	return DeleteFavoriteError(t, token, http.StatusNotFound, adID)
}

func DeleteFavoriteErrorUnauthorized(t *testing.T, adID string) advModels.CreateErrorResponse {
	return DeleteFavoriteError(t, "invalid_token", http.StatusUnauthorized, adID)
}

func DeleteFavoriteErrorNoToken(t *testing.T, adID string) advModels.CreateErrorResponse {
	return DeleteFavoriteError(t, "", http.StatusUnauthorized, adID)
}

func DeleteFavoriteErrorEmptyBearer(t *testing.T, adID string) advModels.CreateErrorResponse {
	return DeleteFavoriteError(t, " ", http.StatusUnauthorized, adID)
}

func BulkAddFavorites(t *testing.T, token string, request models.BulkAddFavoritesRequest) models.BulkAddFavoritesResult {
	rawResponse := favorites.HttpPostFavoritesBulk(t, token, http.StatusOK, request)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	result := models.BulkAddFavoritesResult{}
	err := json.Unmarshal(respArray, &result)
	require.NoError(t, err)

	return result
}

func BulkAddFavoritesError(t *testing.T, token string, statusCode int, request models.BulkAddFavoritesRequest) advModels.CreateErrorResponse {
	rawResponse := favorites.HttpPostFavoritesBulk(t, token, statusCode, request)
	defer rawResponse.Body.Close()

	respArray, errResponse := io.ReadAll(rawResponse.Body)
	require.NoError(t, errResponse)

	errResp := advModels.CreateErrorResponse{}
	err := json.Unmarshal(respArray, &errResp)
	require.NoError(t, err)

	return errResp
}

func BulkAddFavoritesErrorBadRequest(t *testing.T, token string, request models.BulkAddFavoritesRequest) advModels.CreateErrorResponse {
	return BulkAddFavoritesError(t, token, http.StatusBadRequest, request)
}

func BulkAddFavoritesErrorUnauthorized(t *testing.T, request models.BulkAddFavoritesRequest) advModels.CreateErrorResponse {
	return BulkAddFavoritesError(t, "invalid_token", http.StatusUnauthorized, request)
}

func BulkAddFavoritesErrorNoToken(t *testing.T, request models.BulkAddFavoritesRequest) advModels.CreateErrorResponse {
	return BulkAddFavoritesError(t, "", http.StatusUnauthorized, request)
}

func BulkAddFavoritesErrorEmptyBearer(t *testing.T, request models.BulkAddFavoritesRequest) advModels.CreateErrorResponse {
	return BulkAddFavoritesError(t, " ", http.StatusUnauthorized, request)
}

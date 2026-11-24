package fixtures

import (
	"net/http"
	"testing"

	"api-tests-template/internal/managers/favorites"
	favModels "api-tests-template/internal/managers/favorites/models"
	"api-tests-template/internal/utils"
)

// FavoritesTracker добавляет объявления в избранное и запоминает их ID, чтобы Cleanup удалил только их.
// Нулевое значение готово к использованию
type FavoritesTracker struct {
	added []string
}

// Add добавляет объявление в избранное и запоминает его ID
func (f *FavoritesTracker) Add(t *testing.T, token string, request favModels.AddFavoriteRequest) favModels.Favorite {
	added := favorites.AddFavorite(t, token, request)
	f.added = append(f.added, request.AdvertisementId)

	return added
}

// BulkAdd добавляет пакетом и запоминает только added: skipped уже были в избранном до запроса
func (f *FavoritesTracker) BulkAdd(t *testing.T, token string, request favModels.BulkAddFavoritesRequest) favModels.BulkAddFavoritesResult {
	result := favorites.BulkAddFavorites(t, token, request)
	f.added = append(f.added, result.Added...)

	return result
}

// Cleanup удаляет из избранного только запомненные объявления
func (f *FavoritesTracker) Cleanup(t *testing.T, token string) {
	ClearFavorites(t, token, f.added)
	f.added = nil
}

// ClearFavorites удаляет из избранного пользователя только объявления с ID из addedIDs
// (то есть добавленные самим сценарием). Записи, которые были в избранном до теста, не трогаем.
// 404 не считаем ошибкой: тест мог удалить запись сам или она ушла каскадом вместе с объявлением
func ClearFavorites(t *testing.T, token string, addedIDs []string) {
	for _, adID := range addedIDs {
		utils.LogWithLabelAndTimeStamp("Precondition", "Удаляем из избранного объявление с ID: "+adID)

		status, err := favorites.DeleteFavoriteStatus(t, token, adID)
		if err != nil || (status != http.StatusNoContent && status != http.StatusNotFound) {
			t.Errorf("не удалось удалить объявление %s из избранного: статус %d, ошибка %v", adID, status, err)
		}
	}
}

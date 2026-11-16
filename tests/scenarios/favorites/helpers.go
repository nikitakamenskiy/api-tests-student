package favorites

import (
	"crypto/rand"
	"fmt"

	favModels "api-tests-template/internal/managers/favorites/models"
)

// nonExistentAdvertisementID возвращает случайный ID в формате UUID v4, которого заведомо нет на стенде
func nonExistentAdvertisementID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	b[6] = (b[6] & 0x0f) | 0x40 // версия 4
	b[8] = (b[8] & 0x3f) | 0x80 // вариант RFC 4122

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// favoriteAdvertisementIDs возвращает ID объявлений из списка избранного
func favoriteAdvertisementIDs(list favModels.FavoritesList) []string {
	ids := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		ids = append(ids, item.Advertisement.Id)
	}

	return ids
}

// randomAdvertisementIDs возвращает n случайных UUID несуществующих объявлений
func randomAdvertisementIDs(n int) []string {
	ids := make([]string, 0, n)
	for len(ids) < n {
		ids = append(ids, nonExistentAdvertisementID())
	}

	return ids
}

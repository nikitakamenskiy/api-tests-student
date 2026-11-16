package fixtures

import (
	"testing"

	"github.com/stretchr/testify/require"

	"api-tests-template/internal/managers/favorites"
	favModels "api-tests-template/internal/managers/favorites/models"
	"api-tests-template/internal/utils"
)

const maxClearIterations = 5

func ClearFavorites(t *testing.T, token string) {
	utils.LogWithLabelAndTimeStamp("Precondition", "Очищаем избранное пользователя")

	for i := 0; i < maxClearIterations; i++ {
		list := favorites.GetFavorites(t, token, favModels.ListQuery{
			PageSize: "50",
		})

		if len(list.Items) == 0 {
			return
		}

		for _, item := range list.Items {
			favorites.DeleteFavorite(t, token, item.Advertisement.Id)
		}
	}

	require.Fail(t, "не удалось очистить избранное пользователя")
}

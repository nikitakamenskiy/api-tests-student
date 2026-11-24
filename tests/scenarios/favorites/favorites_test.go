package favorites

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"api-tests-template/internal/fixtures"
	advModels "api-tests-template/internal/managers/advertisement/models"
	"api-tests-template/internal/managers/auth"
	"api-tests-template/internal/managers/auth/models"
	"api-tests-template/internal/managers/favorites"
	favModels "api-tests-template/internal/managers/favorites/models"
	"api-tests-template/internal/utils"

	base "api-tests-template/tests"
)

const (
	maxFavoritesPerUser = 50
	defaultPageSize     = 15
	minPageSize         = 10

	invalidRequestError = "invalid_request"
	unauthorizedError   = "unauthorized"
	notFoundError       = "not_found"
	forbiddenError      = "forbidden"
	alreadyExistsError  = "already_exists"

	noTokenMessage     = "Authorization header required"
	emptyBearerMessage = "Invalid authorization header format"
	// #nosec G101 -- This is an error message, not a credential.
	invalidTokenMessage           = "Invalid or expired token"
	addNotFoundMessage            = "Advertisement not found"
	deleteNotFoundMessage         = "Favorite entry not found"
	ownAdvertisementMessage       = "Cannot add own advertisement to favorites"
	duplicateMessage              = "Advertisement is already in favorites"
	emptyAdvertisementIDMessage   = "invalid UUID length: 0"
	invalidAdvertisementIDMessage = "Invalid advertisement ID"
	invalidUUIDMessage            = "invalid UUID length: 10"
	limitReachedError             = "limit_reached"

	limitReachedMessage         = "Favorites limit of 50 reached"
	bulkMissingMessage          = "Key: 'BulkAddFavoritesRequest.AdvertisementIDs' Error:Field validation for 'AdvertisementIDs' failed on the 'required' tag"
	bulkEmptyMessage            = "Key: 'BulkAddFavoritesRequest.AdvertisementIDs' Error:Field validation for 'AdvertisementIDs' failed on the 'min' tag"
	bulkTooManyMessage          = "Key: 'BulkAddFavoritesRequest.AdvertisementIDs' Error:Field validation for 'AdvertisementIDs' failed on the 'max' tag"
	invalidPageMessage          = "Key: 'FavoriteListRequest.Page' Error:Field validation for 'Page' failed on the 'min' tag"
	invalidPageSizeMessage      = "Key: 'FavoriteListRequest.PageSize' Error:Field validation for 'PageSize' failed on the 'oneof' tag"
	invalidSortByMessage        = "Key: 'FavoriteListRequest.SortBy' Error:Field validation for 'SortBy' failed on the 'oneof' tag"
	invalidSortOrderMessage     = "Key: 'FavoriteListRequest.SortOrder' Error:Field validation for 'SortOrder' failed on the 'oneof' tag"
	parseIntMessage             = `strconv.ParseInt: parsing "abc": invalid syntax`
	parseBoolMessage            = `strconv.ParseBool: parsing "abc": invalid syntax`
	invalidRequestErrorAssert   = "Неправильная ошибка"
	invalidRequestMessageAssert = "Неправильное сообщение об ошибке"
)

type TestSuite struct {
	suite.Suite

	actor models.LoginOkResponse // пользователь, у которого проверяем избранное (TEST_LOGIN)
	owner models.LoginOkResponse // владелец объявлений, которые actor добавляет в избранное (TEST_SECOND_LOGIN)
	ads   fixtures.AdvertisementFactory
	favs  fixtures.FavoritesTracker
	// ad — чужое объявление, готовое к добавлению в избранное. Создаётся перед каждым тестом.
	ad advModels.Advertisement
}

func TestSuiteRun(t *testing.T) {
	suite.Run(t, &TestSuite{})
}

func (s *TestSuite) SetupSuite() {
	base.SetupSuite()

	base.Precondition("Авторизация основного пользователя")
	s.actor = auth.Login(s.T(), os.Getenv("TEST_LOGIN"), os.Getenv("TEST_PASSWORD"))

	base.Precondition("Авторизация второго пользователя")
	s.owner = auth.Login(s.T(), os.Getenv("TEST_SECOND_LOGIN"), os.Getenv("TEST_SECOND_PASSWORD"))

	require.NotEqual(s.T(), s.actor.User.Id, s.owner.User.Id,
		"TEST_LOGIN и TEST_SECOND_LOGIN должны быть разными пользователями")
}

func (s *TestSuite) SetupTest() {
	base.Precondition("Создаём объявление от имени второго пользователя")
	s.ad = s.ads.Create(s.T(), s.owner.Token)
}

func (s *TestSuite) TearDownTest() {
	s.favs.Cleanup(s.T(), s.actor.Token)
	s.ads.Cleanup(s.T())
}

func (s *TestSuite) TearDownSuite() {
	base.TearDownSuite()
}

// TestFavoritesFullCycle проверяет полный цикл работы с избранным:
// добавление, получение списка, удаление и каскадное удаление объявления.
func (s *TestSuite) TestFavoritesFullCycle() {
	var added favModels.Favorite

	s.Run("Добавляем чужое объявление в избранное", func() {
		added = s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id})
	})

	s.Run("Проверяем ответ на добавление", func() {
		require.NotEmpty(s.T(), added.FavoriteId, "у избранного нет favorite_id")
		require.Equal(s.T(), s.ad.Id, added.Advertisement.Id, "id объявления не совпадает")
		require.Equal(s.T(), s.ad.Title, added.Advertisement.Title, "title объявления не совпадает")
	})

	s.Run("Проверяем, что объявление появилось в списке избранного", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.Contains(s.T(), favoriteAdvertisementIDs(list), s.ad.Id, "объявления нет в избранном")
	})

	s.Run("Удаляем объявление из избранного", func() {
		favorites.DeleteFavorite(s.T(), s.actor.Token, s.ad.Id)
	})

	s.Run("Проверяем, что объявление пропало из списка избранного", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.NotContains(s.T(), favoriteAdvertisementIDs(list), s.ad.Id, "объявление осталось в избранном")
	})

	// Возвращаем объявление в избранное для проверки каскадного удаления.
	s.Run("Снова добавляем объявление в избранное", func() {
		s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id})
	})

	s.Run("Владелец удаляет само объявление", func() {
		s.ads.Delete(s.T(), s.owner.Token, s.ad.Id)
	})

	s.Run("Проверяем, что удалённое объявление исчезло из избранного", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.NotContains(s.T(), favoriteAdvertisementIDs(list), s.ad.Id, "удалённое объявление осталось в избранном")
	})
}

// TestFavoritesAuthentication проверяет авторизацию для всех операций с избранным.
func (s *TestSuite) TestFavoritesAuthentication() {
	addRequest := favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id}
	bulkRequest := favModels.BulkAddFavoritesRequest{AdvertisementIds: []string{s.ad.Id}}

	// Подготавливаем состояние, чтобы убедиться, что неавторизованные запросы его не меняют.
	s.Run("Добавляем объявление в избранное перед проверкой авторизации", func() {
		s.favs.Add(s.T(), s.actor.Token, addRequest)
	})

	operations := []struct {
		name         string
		noToken      func() advModels.CreateErrorResponse
		emptyBearer  func() advModels.CreateErrorResponse
		invalidToken func() advModels.CreateErrorResponse
	}{
		{
			name: "Получение списка избранного",
			noToken: func() advModels.CreateErrorResponse {
				return favorites.GetFavoritesErrorNoToken(s.T(), favModels.ListQuery{})
			},
			emptyBearer: func() advModels.CreateErrorResponse {
				return favorites.GetFavoritesErrorEmptyBearer(s.T(), favModels.ListQuery{})
			},
			invalidToken: func() advModels.CreateErrorResponse {
				return favorites.GetFavoritesErrorUnauthorized(s.T(), favModels.ListQuery{})
			},
		},
		{
			name: "Добавление объявления в избранное",
			noToken: func() advModels.CreateErrorResponse {
				return favorites.AddFavoriteErrorNoToken(s.T(), addRequest)
			},
			emptyBearer: func() advModels.CreateErrorResponse {
				return favorites.AddFavoriteErrorEmptyBearer(s.T(), addRequest)
			},
			invalidToken: func() advModels.CreateErrorResponse {
				return favorites.AddFavoriteErrorUnauthorized(s.T(), addRequest)
			},
		},
		{
			name: "Пакетное добавление объявлений в избранное",
			noToken: func() advModels.CreateErrorResponse {
				return favorites.BulkAddFavoritesErrorNoToken(s.T(), bulkRequest)
			},
			emptyBearer: func() advModels.CreateErrorResponse {
				return favorites.BulkAddFavoritesErrorEmptyBearer(s.T(), bulkRequest)
			},
			invalidToken: func() advModels.CreateErrorResponse {
				return favorites.BulkAddFavoritesErrorUnauthorized(s.T(), bulkRequest)
			},
		},
		{
			name: "Удаление объявления из избранного",
			noToken: func() advModels.CreateErrorResponse {
				return favorites.DeleteFavoriteErrorNoToken(s.T(), s.ad.Id)
			},
			emptyBearer: func() advModels.CreateErrorResponse {
				return favorites.DeleteFavoriteErrorEmptyBearer(s.T(), s.ad.Id)
			},
			invalidToken: func() advModels.CreateErrorResponse {
				return favorites.DeleteFavoriteErrorUnauthorized(s.T(), s.ad.Id)
			},
		},
	}

	for _, operation := range operations {
		s.Run(operation.name, func() {
			cases := []struct {
				name        string
				call        func() advModels.CreateErrorResponse
				wantMessage string
			}{
				{
					name:        "Без токена авторизации",
					call:        operation.noToken,
					wantMessage: noTokenMessage,
				},
				{
					name:        "С пустым Bearer токеном",
					call:        operation.emptyBearer,
					wantMessage: emptyBearerMessage,
				},
				{
					name:        "С невалидным токеном",
					call:        operation.invalidToken,
					wantMessage: invalidTokenMessage,
				},
			}

			for _, tc := range cases {
				s.Run(tc.name, func() {
					errResponse := tc.call()
					require.Equal(s.T(), unauthorizedError, errResponse.Error, invalidRequestErrorAssert)
					require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
				})
			}
		})
	}

	s.Run("Проверяем, что запросы без авторизации не изменили избранное", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.Equal(s.T(), []string{s.ad.Id}, favoriteAdvertisementIDs(list), "избранное изменилось после запросов без авторизации")
	})
}

// TestAddFavoriteValidationNegative проверяет валидацию advertisement_id при добавлении в избранное.
func (s *TestSuite) TestAddFavoriteValidationNegative() {
	cases := []struct {
		name        string
		adID        string
		wantMessage string
	}{
		{
			name:        "пустой advertisement_id",
			adID:        "",
			wantMessage: emptyAdvertisementIDMessage,
		},
		{
			name:        "id не в формате UUID",
			adID:        "not-a-uuid",
			wantMessage: invalidUUIDMessage,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			errResponse := favorites.AddFavoriteErrorBadRequest(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: tc.adID})

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

// TestDeleteFavoriteInvalidIDNegative проверяет удаление из избранного с некорректным UUID.
func (s *TestSuite) TestDeleteFavoriteInvalidIDNegative() {
	errResponse := favorites.DeleteFavoriteErrorBadRequest(s.T(), s.actor.Token, "not-a-uuid")

	require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
	require.Equal(s.T(), invalidAdvertisementIDMessage, errResponse.Message, invalidRequestMessageAssert)
}

// TestAddNonExistentAdvertisementToFavorites проверяет добавление несуществующего объявления в избранное.
func (s *TestSuite) TestAddNonExistentAdvertisementToFavorites() {
	errResponse := favorites.AddFavoriteErrorNotFound(
		s.T(),
		s.actor.Token,
		favModels.AddFavoriteRequest{AdvertisementId: nonExistentAdvertisementID()},
	)

	require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
	require.Equal(s.T(), addNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)
}

// TestDeleteNonExistentAdvertisementFromFavorites проверяет удаление несуществующего объявления из избранного.
func (s *TestSuite) TestDeleteNonExistentAdvertisementFromFavorites() {
	errResponse := favorites.DeleteFavoriteErrorNotFound(s.T(), s.actor.Token, nonExistentAdvertisementID())

	require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
	require.Equal(s.T(), deleteNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)
}

// TestDeleteAdvertisementNotAddedToFavorites проверяет удаление существующего объявления, которое не добавляли в избранное.
func (s *TestSuite) TestDeleteAdvertisementNotAddedToFavorites() {
	errResponse := favorites.DeleteFavoriteErrorNotFound(s.T(), s.actor.Token, s.ad.Id)

	require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
	require.Equal(s.T(), deleteNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)
}

// TestAddOwnAdvertisementToFavorites проверяет, что нельзя добавить собственное объявление в избранное.
func (s *TestSuite) TestAddOwnAdvertisementToFavorites() {
	var ownAdvertisement advModels.Advertisement
	var errResponse advModels.CreateErrorResponse

	s.Run("Создаём собственное объявление", func() {
		ownAdvertisement = s.ads.Create(s.T(), s.actor.Token)
	})

	s.Run("Добавляем собственное объявление в избранное", func() {
		errResponse = favorites.AddFavoriteErrorForbidden(
			s.T(),
			s.actor.Token,
			favModels.AddFavoriteRequest{AdvertisementId: ownAdvertisement.Id},
		)
	})

	s.Run("Проверяем ошибку", func() {
		require.Equal(s.T(), forbiddenError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), ownAdvertisementMessage, errResponse.Message, invalidRequestMessageAssert)
	})
}

// TestAddDuplicateAdvertisementToFavorites проверяет, что нельзя повторно добавить объявление в избранное.
func (s *TestSuite) TestAddDuplicateAdvertisementToFavorites() {
	request := favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id}
	var errResponse advModels.CreateErrorResponse

	s.Run("Добавляем объявление в избранное", func() {
		s.favs.Add(s.T(), s.actor.Token, request)
	})

	s.Run("Повторно добавляем то же объявление в избранное", func() {
		errResponse = favorites.AddFavoriteErrorConflict(s.T(), s.actor.Token, request)
	})

	s.Run("Проверяем ошибку", func() {
		require.Equal(s.T(), alreadyExistsError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), duplicateMessage, errResponse.Message, invalidRequestMessageAssert)
	})
}

// TestAddDeletedAdvertisementToFavorites проверяет добавление объявления, которое ранее удалил владелец.
func (s *TestSuite) TestAddDeletedAdvertisementToFavorites() {
	var deletedAdvertisement advModels.Advertisement
	var errResponse advModels.CreateErrorResponse

	s.Run("Создаём объявление", func() {
		deletedAdvertisement = s.ads.Create(s.T(), s.owner.Token)
	})

	s.Run("Владелец удаляет объявление", func() {
		s.ads.Delete(s.T(), s.owner.Token, deletedAdvertisement.Id)
	})

	s.Run("Добавляем удалённое объявление в избранное", func() {
		errResponse = favorites.AddFavoriteErrorNotFound(
			s.T(),
			s.actor.Token,
			favModels.AddFavoriteRequest{AdvertisementId: deletedAdvertisement.Id},
		)
	})

	s.Run("Проверяем ошибку", func() {
		require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), addNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)
	})
}

// TestAddFavoriteLimitNegative проверяет, что 51-е объявление нельзя добавить в избранное.
func (s *TestSuite) TestAddFavoriteLimitNegative() {
	if testing.Short() {
		s.T().Skip("долгий тест: создаёт 50 объявлений")
	}

	var advertisementIDs []string
	var extraAdvertisement advModels.Advertisement

	s.Run("Создаём объявления для заполнения избранного", func() {
		advertisementIDs = append(advertisementIDs, s.ad.Id)
		for len(advertisementIDs) < maxFavoritesPerUser {
			ad := s.ads.Create(s.T(), s.owner.Token)
			advertisementIDs = append(advertisementIDs, ad.Id)
		}
	})

	s.Run("Заполняем избранное до лимита", func() {
		result := s.favs.BulkAdd(s.T(), s.actor.Token, favModels.BulkAddFavoritesRequest{AdvertisementIds: advertisementIDs})

		require.Len(s.T(), result.Added, maxFavoritesPerUser, "не все объявления добавились в избранное")
		require.False(s.T(), result.LimitReached, "limit_reached не должен быть true, пока лимит не превышен")
	})

	s.Run("Создаём объявление сверх лимита", func() {
		extraAdvertisement = s.ads.Create(s.T(), s.owner.Token)
	})

	s.Run("Пытаемся добавить объявление сверх лимита", func() {
		errResponse := favorites.AddFavoriteErrorBadRequest(
			s.T(),
			s.actor.Token,
			favModels.AddFavoriteRequest{AdvertisementId: extraAdvertisement.Id},
		)

		require.Equal(s.T(), limitReachedError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), limitReachedMessage, errResponse.Message, invalidRequestMessageAssert)
	})

	s.Run("Пытаемся добавить объявление сверх лимита пакетно", func() {
		result := s.favs.BulkAdd(s.T(), s.actor.Token,
			favModels.BulkAddFavoritesRequest{AdvertisementIds: []string{extraAdvertisement.Id}})

		require.Empty(s.T(), result.Added, "объявление сверх лимита не должно добавляться")
		require.True(s.T(), result.LimitReached, "limit_reached должен быть true при заполненном избранном")
	})

	s.Run("Проверяем, что в избранном осталось ровно 50 объявлений", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{PageSize: "50"})

		require.Equal(s.T(), maxFavoritesPerUser, list.Total, "total не равен лимиту")
		require.NotContains(s.T(), favoriteAdvertisementIDs(list), extraAdvertisement.Id, "объявление сверх лимита попало в избранное")
	})

	s.Run("После удаления одного объявления можно добавить новое", func() {
		favorites.DeleteFavorite(s.T(), s.actor.Token, s.ad.Id)

		added := s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: extraAdvertisement.Id})
		require.Equal(s.T(), extraAdvertisement.Id, added.Advertisement.Id, "добавилось не то объявление")
	})
}

// TestDeleteFavoriteTwiceNegative проверяет, что повторное удаление возвращает 404.
func (s *TestSuite) TestDeleteFavoriteTwiceNegative() {
	s.Run("Добавляем и удаляем объявление из избранного", func() {
		s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id})
		favorites.DeleteFavorite(s.T(), s.actor.Token, s.ad.Id)
	})

	var errResponse advModels.CreateErrorResponse
	s.Run("Повторно удаляем то же объявление", func() {
		errResponse = favorites.DeleteFavoriteErrorNotFound(s.T(), s.actor.Token, s.ad.Id)
	})

	s.Run("Проверяем ошибку", func() {
		require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), deleteNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)
	})
}

// TestFavoritesIsolationBetweenUsers проверяет, что избранное принадлежит конкретному пользователю.
func (s *TestSuite) TestFavoritesIsolationBetweenUsers() {
	s.Run("Основной пользователь добавляет объявление в избранное", func() {
		s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: s.ad.Id})
	})

	s.Run("Другой пользователь не видит чужое избранное", func() {
		list := favorites.GetFavorites(s.T(), s.owner.Token, favModels.ListQuery{})
		require.NotContains(s.T(), favoriteAdvertisementIDs(list), s.ad.Id, "чужое избранное видно другому пользователю")
	})

	var errResponse advModels.CreateErrorResponse
	s.Run("Другой пользователь пытается удалить чужое избранное", func() {
		errResponse = favorites.DeleteFavoriteErrorNotFound(s.T(), s.owner.Token, s.ad.Id)
	})

	s.Run("Проверяем ошибку и что избранное основного пользователя не изменилось", func() {
		require.Equal(s.T(), notFoundError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), deleteNotFoundMessage, errResponse.Message, invalidRequestMessageAssert)

		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.Contains(s.T(), favoriteAdvertisementIDs(list), s.ad.Id, "объявление пропало из избранного владельца списка")
	})
}

// TestGetFavoritesPagination проверяет пагинацию списка избранного: значения по умолчанию и переход между страницами.
func (s *TestSuite) TestGetFavoritesPagination() {
	s.Run("Проверяем значения по умолчанию для пустого избранного", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})

		require.Empty(s.T(), list.Items, "в пустом избранном не должно быть объявлений")
		require.Equal(s.T(), 1, list.Page, "неверная страница по умолчанию")
		require.Equal(s.T(), defaultPageSize, list.PageSize, "неверный размер страницы по умолчанию")
		require.Zero(s.T(), list.Total, "total должен быть равен 0")
		require.Zero(s.T(), list.TotalPages, "total_pages должен быть равен 0")
	})

	var advertisementIDs []string
	s.Run("Добавляем в избранное на одно объявление больше размера страницы", func() {
		advertisementIDs = append(advertisementIDs, s.ad.Id)
		for len(advertisementIDs) <= minPageSize {
			advertisementIDs = append(advertisementIDs, s.ads.Create(s.T(), s.owner.Token).Id)
		}

		result := s.favs.BulkAdd(s.T(), s.actor.Token, favModels.BulkAddFavoritesRequest{AdvertisementIds: advertisementIDs})
		require.Len(s.T(), result.Added, len(advertisementIDs), "не все объявления добавились в избранное")
	})

	var firstPage favModels.FavoritesList
	s.Run("Получаем первую страницу", func() {
		firstPage = favorites.GetFavorites(s.T(), s.actor.Token,
			favModels.ListQuery{Page: "1", PageSize: strconv.Itoa(minPageSize)})

		require.Len(s.T(), firstPage.Items, minPageSize, "неверное количество элементов на первой странице")
		require.Equal(s.T(), 1, firstPage.Page, "неверный номер страницы")
		require.Equal(s.T(), minPageSize, firstPage.PageSize, "неверный размер страницы")
		require.Equal(s.T(), len(advertisementIDs), firstPage.Total, "неверный total")
		require.Equal(s.T(), 2, firstPage.TotalPages, "неверный total_pages")
	})

	var secondPage favModels.FavoritesList
	s.Run("Получаем вторую страницу", func() {
		secondPage = favorites.GetFavorites(s.T(), s.actor.Token,
			favModels.ListQuery{Page: "2", PageSize: strconv.Itoa(minPageSize)})

		require.Len(s.T(), secondPage.Items, 1, "неверное количество элементов на второй странице")
		require.Equal(s.T(), 2, secondPage.Page, "неверный номер страницы")
	})

	s.Run("Проверяем, что страницы не пересекаются и вместе содержат всё избранное", func() {
		allIDs := append(favoriteAdvertisementIDs(firstPage), favoriteAdvertisementIDs(secondPage)...)

		require.ElementsMatch(s.T(), advertisementIDs, allIDs, "страницы пересекаются или потеряны объявления")
	})
}

// TestGetFavoritesSorting проверяет сортировку списка избранного по added_at, price и title.
func (s *TestSuite) TestGetFavoritesSorting() {
	prefix := "sort_" + utils.RandomString(8)

	// Порядок создания и добавления в избранное: ads[0], ads[1], ads[2].
	var ads [3]advModels.Advertisement
	s.Run("Создаём объявления с разными price и title и добавляем в избранное по очереди", func() {
		overrides := []map[string]string{
			{"title": prefix + "_b", "price": "300"},
			{"title": prefix + "_c", "price": "100"},
			{"title": prefix + "_a", "price": "200"},
		}

		for i, override := range overrides {
			ads[i] = s.ads.CreateWithParams(s.T(), s.owner.Token, override)
			s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: ads[i].Id})
		}
	})

	cases := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantOrder []int // индексы ads в ожидаемом порядке
	}{
		{name: "по умолчанию: сначала новые", wantOrder: []int{2, 1, 0}},
		{name: "added_at: по возрастанию", sortBy: "added_at", sortOrder: "asc", wantOrder: []int{0, 1, 2}},
		{name: "added_at: по убыванию", sortBy: "added_at", sortOrder: "desc", wantOrder: []int{2, 1, 0}},
		{name: "price: по возрастанию", sortBy: "price", sortOrder: "asc", wantOrder: []int{1, 2, 0}},
		{name: "price: по убыванию", sortBy: "price", sortOrder: "desc", wantOrder: []int{0, 2, 1}},
		{name: "title: по возрастанию", sortBy: "title", sortOrder: "asc", wantOrder: []int{2, 0, 1}},
		{name: "title: по убыванию", sortBy: "title", sortOrder: "desc", wantOrder: []int{1, 0, 2}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{SortBy: tc.sortBy, SortOrder: tc.sortOrder})

			wantIDs := make([]string, 0, len(tc.wantOrder))
			for _, i := range tc.wantOrder {
				wantIDs = append(wantIDs, ads[i].Id)
			}

			require.Equal(s.T(), wantIDs, favoriteAdvertisementIDs(list), "неверный порядок объявлений в избранном")
		})
	}
}

// TestGetFavoritesSearch проверяет поиск по названию объявления в избранном.
func (s *TestSuite) TestGetFavoritesSearch() {
	unique := utils.RandomString(8)

	var ads [2]advModels.Advertisement
	s.Run("Создаём два объявления с разными названиями и добавляем в избранное", func() {
		overrides := []map[string]string{
			{"title": "apple_" + unique},
			{"title": "banana_" + unique},
		}

		for i, override := range overrides {
			ads[i] = s.ads.CreateWithParams(s.T(), s.owner.Token, override)
			s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: ads[i].Id})
		}
	})

	cases := []struct {
		name    string
		search  string
		wantAds []int // индексы ads, которые должны найтись
	}{
		{name: "по названию первого объявления", search: "apple_" + unique, wantAds: []int{0}},
		{name: "по названию второго объявления", search: "banana_" + unique, wantAds: []int{1}},
		{name: "по общей части названий", search: unique, wantAds: []int{0, 1}},
		{name: "ничего не найдено", search: "missing_" + utils.RandomString(10), wantAds: []int{}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{Search: tc.search})

			wantIDs := make([]string, 0, len(tc.wantAds))
			for _, i := range tc.wantAds {
				wantIDs = append(wantIDs, ads[i].Id)
			}

			require.ElementsMatch(s.T(), wantIDs, favoriteAdvertisementIDs(list), "неверный результат поиска")
			require.Equal(s.T(), len(tc.wantAds), list.Total, "неверный total")
		})
	}
}

// TestGetFavoritesQuantityFilter проверяет фильтр quantity: только объявления с количеством больше 1.
func (s *TestSuite) TestGetFavoritesQuantityFilter() {
	var ads [2]advModels.Advertisement
	s.Run("Создаём объявления с quantity 1 и 5 и добавляем в избранное", func() {
		overrides := []map[string]string{
			{"title": "single_" + utils.RandomString(8), "quantity": "1"},
			{"title": "many_" + utils.RandomString(8), "quantity": "5"},
		}

		for i, override := range overrides {
			ads[i] = s.ads.CreateWithParams(s.T(), s.owner.Token, override)
			s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: ads[i].Id})
		}
	})

	cases := []struct {
		name     string
		quantity string
		wantAds  []int
	}{
		{name: "quantity=true: только с количеством больше 1", quantity: "true", wantAds: []int{1}},
		{name: "quantity=false: без фильтра", quantity: "false", wantAds: []int{0, 1}},
		{name: "без параметра: без фильтра", quantity: "", wantAds: []int{0, 1}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{Quantity: tc.quantity})

			wantIDs := make([]string, 0, len(tc.wantAds))
			for _, i := range tc.wantAds {
				wantIDs = append(wantIDs, ads[i].Id)
			}

			require.ElementsMatch(s.T(), wantIDs, favoriteAdvertisementIDs(list), "неверный результат фильтрации")
		})
	}
}

// TestGetFavoritesValidationNegative проверяет ошибки валидации query-параметров списка избранного.
func (s *TestSuite) TestGetFavoritesValidationNegative() {
	cases := []struct {
		name        string
		query       favModels.ListQuery
		wantMessage string
		skip        string
	}{
		{
			name:        "page: ноль",
			query:       favModels.ListQuery{Page: "0"},
			wantMessage: invalidPageMessage,
			skip:        "BugXXX: сервер принимает page=0 и отвечает 200 вместо 400",
		},
		{name: "page: отрицательная", query: favModels.ListQuery{Page: "-1"}, wantMessage: invalidPageMessage},
		{name: "page: строка", query: favModels.ListQuery{Page: "abc"}, wantMessage: parseIntMessage},
		{name: "page_size: не из списка допустимых", query: favModels.ListQuery{PageSize: "7"}, wantMessage: invalidPageSizeMessage},
		{name: "page_size: больше максимума", query: favModels.ListQuery{PageSize: "100"}, wantMessage: invalidPageSizeMessage},
		{name: "page_size: строка", query: favModels.ListQuery{PageSize: "abc"}, wantMessage: parseIntMessage},
		{name: "sort_by: неизвестное поле", query: favModels.ListQuery{SortBy: "unknown"}, wantMessage: invalidSortByMessage},
		{name: "sort_order: неизвестное значение", query: favModels.ListQuery{SortOrder: "up"}, wantMessage: invalidSortOrderMessage},
		{name: "quantity: не boolean", query: favModels.ListQuery{Quantity: "abc"}, wantMessage: parseBoolMessage},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			if tc.skip != "" {
				s.T().Skip(tc.skip)
			}
			errResponse := favorites.GetFavoritesErrorBadRequest(s.T(), s.actor.Token, tc.query)

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

// TestBulkAddFavorites проверяет пакетное добавление: добавленные, уже существующие, свои и несуществующие объявления.
func (s *TestSuite) TestBulkAddFavorites() {
	var alreadyFavorite, ownAdvertisement advModels.Advertisement
	missingID := nonExistentAdvertisementID()

	s.Run("Готовим объявления: уже в избранном, своё", func() {
		alreadyFavorite = s.ads.Create(s.T(), s.owner.Token)
		s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: alreadyFavorite.Id})

		ownAdvertisement = s.ads.Create(s.T(), s.actor.Token)
	})

	var result favModels.BulkAddFavoritesResult
	s.Run("Пакетно добавляем новое, уже добавленное, своё и несуществующее объявления", func() {
		result = s.favs.BulkAdd(s.T(), s.actor.Token, favModels.BulkAddFavoritesRequest{
			AdvertisementIds: []string{s.ad.Id, alreadyFavorite.Id, ownAdvertisement.Id, missingID},
		})
	})

	s.Run("Проверяем результат", func() {
		require.Equal(s.T(), []string{s.ad.Id}, result.Added, "added: должно быть только новое объявление")
		require.ElementsMatch(s.T(), []string{alreadyFavorite.Id, ownAdvertisement.Id}, result.Skipped,
			"skipped: должны быть уже добавленное и своё объявления")
		require.Equal(s.T(), []string{missingID}, result.NotFound, "not_found: должно быть несуществующее объявление")
		require.False(s.T(), result.LimitReached, "limit_reached не должен быть true")
	})

	s.Run("Проверяем, что в избранном только новое и ранее добавленное объявления", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.ElementsMatch(s.T(), []string{s.ad.Id, alreadyFavorite.Id}, favoriteAdvertisementIDs(list),
			"неверное содержимое избранного")
	})
}

// TestBulkAddFavoritesIdempotent проверяет, что повторное пакетное добавление ничего не меняет.
func (s *TestSuite) TestBulkAddFavoritesIdempotent() {
	request := favModels.BulkAddFavoritesRequest{AdvertisementIds: []string{s.ad.Id}}

	s.Run("Пакетно добавляем объявление", func() {
		result := s.favs.BulkAdd(s.T(), s.actor.Token, request)
		require.Equal(s.T(), []string{s.ad.Id}, result.Added, "объявление должно добавиться")
	})

	s.Run("Повторно пакетно добавляем то же объявление", func() {
		result := s.favs.BulkAdd(s.T(), s.actor.Token, request)

		require.Empty(s.T(), result.Added, "повторно ничего не должно добавляться")
		require.Equal(s.T(), []string{s.ad.Id}, result.Skipped, "объявление должно попасть в skipped")
	})

	s.Run("Проверяем, что в избранном одна запись", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
		require.Equal(s.T(), []string{s.ad.Id}, favoriteAdvertisementIDs(list), "в избранном должно быть одно объявление")
	})
}

// TestBulkAddFavoritesValidationNegative проверяет валидацию списка advertisement_ids при пакетном добавлении.
func (s *TestSuite) TestBulkAddFavoritesValidationNegative() {
	cases := []struct {
		name        string
		ids         []string
		wantMessage string
	}{
		{name: "advertisement_ids: отсутствует", ids: nil, wantMessage: bulkMissingMessage},
		{name: "advertisement_ids: пустой список", ids: []string{}, wantMessage: bulkEmptyMessage},
		{name: "advertisement_ids: больше максимума", ids: randomAdvertisementIDs(maxFavoritesPerUser + 1), wantMessage: bulkTooManyMessage},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			errResponse := favorites.BulkAddFavoritesErrorBadRequest(s.T(), s.actor.Token,
				favModels.BulkAddFavoritesRequest{AdvertisementIds: tc.ids})

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

// TestGuestFavoritesMergeAfterLogin проверяет слияние гостевого избранного: после логина фронт отправляет
// id из localStorage одним запросом на POST /favorites/bulk. В списке есть новые, уже добавленные,
// собственные, удалённые владельцем и несуществующие объявления.
func (s *TestSuite) TestGuestFavoritesMergeAfterLogin() {
	var second, alreadyFavorite, ownAdvertisement, deletedAdvertisement advModels.Advertisement
	missingID := nonExistentAdvertisementID()
	var guestIDs []string

	s.Run("Готовим избранное аккаунта и гостевой список", func() {
		second = s.ads.Create(s.T(), s.owner.Token)
		alreadyFavorite = s.ads.Create(s.T(), s.owner.Token)
		s.favs.Add(s.T(), s.actor.Token, favModels.AddFavoriteRequest{AdvertisementId: alreadyFavorite.Id})

		ownAdvertisement = s.ads.Create(s.T(), s.actor.Token)

		deletedAdvertisement = s.ads.Create(s.T(), s.owner.Token)
		s.ads.Delete(s.T(), s.owner.Token, deletedAdvertisement.Id)

		guestIDs = []string{s.ad.Id, alreadyFavorite.Id, ownAdvertisement.Id, deletedAdvertisement.Id, missingID, second.Id}
	})

	var session models.LoginOkResponse
	s.Run("Гость логинится в аккаунт", func() {
		session = auth.Login(s.T(), os.Getenv("TEST_LOGIN"), os.Getenv("TEST_PASSWORD"))
	})

	var firstResult favModels.BulkAddFavoritesResult
	s.Run("Отправляем гостевой список на слияние", func() {
		firstResult = s.favs.BulkAdd(s.T(), session.Token,
			favModels.BulkAddFavoritesRequest{AdvertisementIds: guestIDs})
	})

	s.Run("Проверяем, что каждый id попал в нужную корзину", func() {
		require.ElementsMatch(s.T(), []string{s.ad.Id, second.Id}, firstResult.Added,
			"added: только новые чужие объявления")
		require.ElementsMatch(s.T(), []string{alreadyFavorite.Id, ownAdvertisement.Id}, firstResult.Skipped,
			"skipped: должны быть уже добавленное и своё объявления")
		require.ElementsMatch(s.T(), []string{deletedAdvertisement.Id, missingID}, firstResult.NotFound,
			"not_found: должны быть удалённое и несуществующее объявления")
		require.False(s.T(), firstResult.LimitReached, "limit_reached не должен быть true")
		require.Equal(s.T(), len(guestIDs), len(firstResult.Added)+len(firstResult.Skipped)+len(firstResult.NotFound),
			"каждый id из запроса должен попасть ровно в одну корзину")
	})

	s.Run("Проверяем содержимое избранного после слияния", func() {
		list := favorites.GetFavorites(s.T(), session.Token, favModels.ListQuery{})

		require.ElementsMatch(s.T(), []string{s.ad.Id, alreadyFavorite.Id, second.Id}, favoriteAdvertisementIDs(list),
			"неверное содержимое избранного")
		require.Equal(s.T(), 3, list.Total, "неверный total")
	})

	var secondResult favModels.BulkAddFavoritesResult
	s.Run("Тот же гостевой список приходит со второго устройства", func() {
		secondResult = s.favs.BulkAdd(s.T(), session.Token,
			favModels.BulkAddFavoritesRequest{AdvertisementIds: guestIDs})
	})

	s.Run("Проверяем, что повторное слияние ничего не меняет", func() {
		require.Empty(s.T(), secondResult.Added, "повторно ничего не должно добавляться")
		require.ElementsMatch(s.T(), []string{s.ad.Id, alreadyFavorite.Id, second.Id, ownAdvertisement.Id}, secondResult.Skipped,
			"skipped: должны быть все добавленные и своё объявления")
		require.ElementsMatch(s.T(), []string{deletedAdvertisement.Id, missingID}, secondResult.NotFound,
			"not_found: должны быть удалённое и несуществующее объявления")
		require.False(s.T(), secondResult.LimitReached, "limit_reached не должен быть true")

		list := favorites.GetFavorites(s.T(), session.Token, favModels.ListQuery{})
		require.Equal(s.T(), 3, list.Total, "total изменился после повторного слияния")
	})
}

func (s *TestSuite) TestBulkAddFavoritesDuplicateIDsInRequest() {
	s.Run("Пакетно добавляем один и тот же id дважды", func() {
		s.favs.BulkAdd(s.T(), s.actor.Token,
			favModels.BulkAddFavoritesRequest{AdvertisementIds: []string{s.ad.Id, s.ad.Id}})
	})

	s.Run("Проверяем, что в избранном одна запись", func() {
		list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})

		require.Equal(s.T(), []string{s.ad.Id}, favoriteAdvertisementIDs(list), "в избранном должно быть одно объявление")
		require.Equal(s.T(), 1, list.Total, "неверный total")
	})
}

func (s *TestSuite) TestBulkAddFavoritesInvalidIDInListNegative() {
	cases := []struct {
		name        string
		ids         []string
		wantMessage string
	}{
		{
			name:        "невалидный id первым",
			ids:         []string{"not-a-uuid", s.ad.Id},
			wantMessage: invalidUUIDMessage,
		},
		{
			name:        "невалидный id последним",
			ids:         []string{s.ad.Id, "not-a-uuid"},
			wantMessage: invalidUUIDMessage,
		},
		{
			name:        "невалидный id между валидными",
			ids:         []string{s.ad.Id, "not-a-uuid", nonExistentAdvertisementID()},
			wantMessage: invalidUUIDMessage,
		},
		{
			name:        "пустая строка среди валидных",
			ids:         []string{s.ad.Id, ""},
			wantMessage: emptyAdvertisementIDMessage,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			errResponse := favorites.BulkAddFavoritesErrorBadRequest(s.T(), s.actor.Token,
				favModels.BulkAddFavoritesRequest{AdvertisementIds: tc.ids})

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)

			list := favorites.GetFavorites(s.T(), s.actor.Token, favModels.ListQuery{})
			require.Empty(s.T(), list.Items, "запрос отклонён, но валидный id попал в избранное")
			require.Zero(s.T(), list.Total, "total должен быть равен 0")
		})
	}
}

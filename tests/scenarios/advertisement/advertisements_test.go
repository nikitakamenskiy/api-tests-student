package advertisement

import (
	"api-tests-template/internal/fixtures"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"api-tests-template/internal/managers/advertisement"
	advModels "api-tests-template/internal/managers/advertisement/models"
	advList "api-tests-template/internal/managers/advertisements"
	advListModels "api-tests-template/internal/managers/advertisements/models"
	"api-tests-template/internal/managers/auth"
	"api-tests-template/internal/managers/auth/models"
	"api-tests-template/internal/utils"
	base "api-tests-template/tests"
)

type TestSuite struct {
	suite.Suite
	loginData      models.LoginOkResponse
	currentDirName string
	createdAdvID   string
}

const (
	invalidRequestError         = "invalid_request"
	unauthorizedError           = "unauthorized"
	invalidRequestErrorAssert   = "Неправильная ошибка"
	invalidRequestMessageAssert = "Неправильное сообщение об ошибке"
)

func TestSuiteRun(t *testing.T) {
	suite.Run(t, &TestSuite{})
}

func (s *TestSuite) SetupSuite() {
	base.SetupSuite()

	base.Precondition("Получаем текущую директорию")
	var err error
	s.currentDirName, err = os.Getwd()
	require.NoError(s.T(), err, "невозможно получить текущую директорию")

	base.Precondition("Авторизация пользователя с кредами из переменных окружения и получение его параметров")
	s.loginData = auth.Login(s.T(), os.Getenv("TEST_LOGIN"), os.Getenv("TEST_PASSWORD"))
}

func (s *TestSuite) TearDownTest() {
	if s.createdAdvID != "" {
		base.Precondition("Удаляем созданное объявление с ID: " + s.createdAdvID)
		advertisement.DeleteAdvertisement(s.T(), s.loginData.Token, s.createdAdvID)
		s.createdAdvID = ""
	}
}

func (s *TestSuite) TearDownSuite() {
	base.TearDownSuite()
}

func (s *TestSuite) TestPostAdvertisementPositive() {
	params := fixtures.NewAdvertisementParams()
	photos := newAdvertisementPhotos(s.currentDirName)

	var advCreateResponse advModels.Advertisement
	s.Run("Создаём объявление со всеми параметрами", func() {
		advCreateResponse = advertisement.CreateAdvertisement(s.T(), s.loginData.Token, params, photos)
		s.createdAdvID = advCreateResponse.Id
	})

	s.Run("Проверяем, что все поля объявления соответствуют отправленным данным", func() {
		require.Equal(s.T(), params["title"], advCreateResponse.Title, "title не совпадает")
		require.Equal(s.T(), params["description"], advCreateResponse.Description, "description не совпадает")
		require.Equal(s.T(), params["price"], strconv.FormatFloat(advCreateResponse.Price, 'f', -1, 64), "price не совпадает")
		require.Equal(s.T(), params["quantity"], strconv.Itoa(advCreateResponse.Quantity), "quantity не совпадает")
		require.Len(s.T(), advCreateResponse.Photos, 3, "загружены не все фотографии")
	})

	var advertisementGet advModels.Advertisement
	s.Run("Получаем объявление по ID", func() {
		advertisementGet = advertisement.GetAdvertisement(s.T(), advCreateResponse.Id)
	})

	s.Run("Проверяем что вернулось тоже объявление что и при создании", func() {
		advertisementGetWithoutUser := advertisementGet
		advertisementGetWithoutUser.User = models.User{}
		require.Equal(s.T(), advCreateResponse, advertisementGetWithoutUser, "объекты объявлений не совпадают")
	})

	var advertisementPhotos []advModels.Photo
	s.Run("Получаем фотографии объявления", func() {
		advertisementPhotos = advertisement.GetAdvertisementPhotos(s.T(), s.loginData.Token, advCreateResponse.Id)
	})

	s.Run("Проверяем что фотографии совпадают с загруженными", func() {
		require.Len(s.T(), advertisementPhotos, 3, "количество фотографий не совпадает")
		for i, photo := range advCreateResponse.Photos {
			require.Equal(s.T(), photo, advertisementPhotos[i], "фотография не совпадает")
		}
	})

	var advertisementsList advListModels.AdvertisementsList
	s.Run("Получаем список объявлений  по полному названию", func() {
		advertisementsList = advList.GetAdvertisements(s.T(), advCreateResponse.Title)
	})

	s.Run("Проверяем что созданное объявление появилось в поиске", func() {
		require.Len(s.T(), advertisementsList.Items, 1, "количество найденных объявлений не равно 1")
		require.Equal(s.T(), advertisementGet, advertisementsList.Items[0], "вернулось другое объявление")
	})
}

const (
	maxTitleLen       = 50
	maxDescriptionLen = 500
)

type formFieldCase struct {
	name        string
	field       string // поле формы: title, description, price, quantity
	value       string // отправляемое значение
	omit        bool   // true - поле не отправляем вообще
	wantMessage string
	skip        string // причина пропуска (известный баг)
}

func (s *TestSuite) TestPostAdvertisementFieldValidationNegative() {
	cases := []formFieldCase{
		{
			name:        "title: отсутствует",
			field:       "title",
			omit:        true,
			wantMessage: "title is required",
		},
		{
			name:        "title: пустой",
			field:       "title",
			value:       "",
			wantMessage: "title is required",
		},
		{
			name:        "title: длиннее допустимой длины",
			field:       "title",
			value:       utils.RandomString(maxTitleLen + 1),
			wantMessage: fmt.Sprintf("title must not exceed 50 characters provided %d", maxTitleLen+1),
		},
		{
			name:        "title: из пробелов",
			field:       "title",
			value:       "     ",
			wantMessage: "title is required",
			skip:        "Bug001: сервер принимает title состоящий из пробелов вместо возврата 400",
		},

		{
			name:        "description: отсутствует",
			field:       "description",
			omit:        true,
			wantMessage: "description is required",
		},
		{
			name:        "description: пустой",
			field:       "description",
			value:       "",
			wantMessage: "description is required",
		},
		{
			name:        "description: длиннее допустимой длины",
			field:       "description",
			value:       utils.RandomString(maxDescriptionLen + 1),
			wantMessage: fmt.Sprintf("description must be between 1 and %d characters", maxDescriptionLen),
		},
		{
			name:        "description: из пробелов",
			field:       "description",
			value:       "     ",
			wantMessage: "description is required",
			skip:        "Bug002: сервер принимает description состоящий из пробелов вместо возврата 400",
		},

		{
			name:  "price: отрицательная",
			field: "price", value: "-1",
			wantMessage: "price must be between 0 and 1000000 inclusive",
		},
		{
			name:        "price: строка",
			field:       "price",
			value:       "abc",
			wantMessage: "cannot convert string value abc to float",
		},
		{
			name:        "price: больше максимального на 1",
			field:       "price",
			value:       "1000001",
			wantMessage: "price must be between 0 and 1000000 inclusive",
		},
		{
			name:        "price: переполнение int",
			field:       "price",
			value:       "999999999999999",
			wantMessage: "price must be between 0 and 1000000 inclusive",
		},
		{
			name:        "quantity: ноль",
			field:       "quantity",
			value:       "0",
			wantMessage: "quantity must be between 1 and 100 inclusive",
		},
		{
			name:        "quantity: отрицательное",
			field:       "quantity",
			value:       "-1",
			wantMessage: "quantity must be between 1 and 100 inclusive",
		},
		{
			name:        "quantity: дробное",
			field:       "quantity",
			value:       "1.5",
			wantMessage: "cannot convert string value to int",
		},
		{
			name:        "quantity: строка",
			field:       "quantity",
			value:       "abc",
			wantMessage: "cannot convert string value to int",
		},
		{
			name:        "quantity: больше максимального на 1",
			field:       "quantity",
			value:       "101",
			wantMessage: "quantity must be between 1 and 100 inclusive",
		},
		{
			name:        "quantity: переполнение int",
			field:       "quantity",
			value:       "999999999999",
			wantMessage: "quantity must be between 1 and 100 inclusive",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			if tc.skip != "" {
				s.T().Skip(tc.skip)
			}

			params := fixtures.NewAdvertisementParams()
			setFormField(s.T(), params, tc.field, tc.value, tc.omit)

			errResponse := advertisement.CreateAdvertisementErrorBadRequest(s.T(), s.loginData.Token, params,
				newAdvertisementPhotos(s.currentDirName))

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

func (s *TestSuite) TestPostAdvertisementPhotosNegative() {
	cases := []struct {
		name        string
		photos      func(dir string) []string
		wantMessage string
		skip        string
	}{
		{
			name:        "без фото",
			photos:      func(string) []string { return []string{} },
			wantMessage: "at least 1 photo is required",
		},
		{
			name: "4 фото вместо максимум 3",
			photos: func(dir string) []string {
				return append(newAdvertisementPhotos(dir), filepath.Join(dir, "images", "advertisementImage1.jpg"))
			},
			wantMessage: "maximum 3 photos allowed",
		},
		{
			name:        "файл больше 5MB",
			photos:      newLargePhoto,
			wantMessage: "photo 1: file size exceeds maximum allowed: 5242880",
		},
		{
			name: "пустой файл (0 байт)",
			photos: func(dir string) []string {
				return []string{filepath.Join(dir, "images", "emptyImage.jpg")}
			},
			wantMessage: "photo 1: file type not allowed: detected content text/plain",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			if tc.skip != "" {
				s.T().Skip(tc.skip)
			}

			errResponse := advertisement.CreateAdvertisementErrorBadRequest(s.T(), s.loginData.Token,
				fixtures.NewAdvertisementParams(), tc.photos(s.currentDirName))

			require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

func (s *TestSuite) TestPostAdvertisementInvalidPhotoFormatNegative() {
	var errResponse advModels.CreateErrorResponse
	s.Run("Создаём объявление с файлом .txt вместо фото", func() {
		photos := []string{filepath.Join(s.currentDirName, "images", "incorrectImage.txt")}
		errResponse = advertisement.CreateAdvertisementErrorBadRequest(s.T(), s.loginData.Token, fixtures.NewAdvertisementParams(), photos)
	})

	s.Run("Проверяем ошибку", func() {
		require.Equal(s.T(), invalidRequestError, errResponse.Error, invalidRequestErrorAssert)
		require.Equal(s.T(), `photo 1: file type not allowed: extension ".txt"`,
			errResponse.Message, invalidRequestMessageAssert)
	})
}

func (s *TestSuite) TestPostAdvertisementAuthNegative() {
	cases := []struct {
		name        string
		token       string
		wantMessage string
	}{
		{"без токена", "", "Authorization header required"},
		{"пустой Bearer токен", " ", "Invalid authorization header format"},
		{"невалидный токен", "invalid_token", "Invalid or expired token"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			errResponse := advertisement.CreateAdvertisementError(s.T(), tc.token, http.StatusUnauthorized,
				fixtures.NewAdvertisementParams(), newAdvertisementPhotos(s.currentDirName))

			require.Equal(s.T(), unauthorizedError, errResponse.Error, invalidRequestErrorAssert)
			require.Equal(s.T(), tc.wantMessage, errResponse.Message, invalidRequestMessageAssert)
		})
	}
}

func (s *TestSuite) TestPostAdvertisementBoundaryPositive() {
	cases := []struct {
		name  string
		field string
		value string
	}{
		{"title: 1 символ", "title", utils.RandomString(1)},
		{"title: максимальная длина 50", "title", utils.RandomString(maxTitleLen)},
		{"description: 1 символ", "description", utils.RandomString(1)},
		{"description: максимальная длина 500", "description", utils.RandomString(maxDescriptionLen)},
		{"price: минимальное значение 0", "price", "0"},
		{"price: максимальное значение 1000000", "price", "1000000"},
		{"quantity: минимальное значение 1", "quantity", "1"},
		{"quantity: максимальное значение 100", "quantity", "100"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			params := fixtures.NewAdvertisementParams()
			setFormField(s.T(), params, tc.field, tc.value, false)

			resp := advertisement.CreateAdvertisement(s.T(), s.loginData.Token, params,
				newAdvertisementPhotos(s.currentDirName))
			s.T().Cleanup(func() {
				advertisement.DeleteAdvertisement(s.T(), s.loginData.Token, resp.Id)
			})

			require.Equal(s.T(), params["title"], resp.Title, "title не совпадает")
			require.Equal(s.T(), params["description"], resp.Description, "description не совпадает")
			require.Equal(s.T(), params["price"], strconv.FormatFloat(resp.Price, 'f', -1, 64),
				"price не совпадает",
			)
			require.Equal(s.T(), params["quantity"], strconv.Itoa(resp.Quantity), "quantity не совпадает")
		})
	}
}

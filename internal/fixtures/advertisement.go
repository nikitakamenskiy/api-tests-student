package fixtures

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"api-tests-template/internal/managers/advertisement"
	advModels "api-tests-template/internal/managers/advertisement/models"
	"api-tests-template/internal/utils"
)

const cleanupDelay = 200 * time.Millisecond

type createdAdvertisement struct {
	id         string
	ownerToken string
}

// AdvertisementFactory создаёт объявления для тестов, где объявление - не предмет проверки, а входные данные.
// Помнит всё созданное и удаляет в Cleanup. Нулевое значение готово к использованию
type AdvertisementFactory struct {
	created []createdAdvertisement
}

// NewAdvertisementParams генерирует уникальные валидные параметры объявления
func NewAdvertisementParams() map[string]string {
	return map[string]string{
		"title":       "fixture_title_" + utils.RandomString(14),
		"description": "fixture_description_" + utils.RandomString(14),
		"price":       strconv.Itoa(utils.RandomInt(1, 10000)),
		"quantity":    strconv.Itoa(utils.RandomInt(1, 100)),
	}
}

// Create создаёт объявление от имени владельца и запоминает его для удаления в Cleanup
func (f *AdvertisementFactory) Create(t *testing.T, ownerToken string) advModels.Advertisement {
	return f.CreateWithParams(t, ownerToken, nil)
}

// CreateWithParams создаёт объявление так же, как Create, но с заданными полями формы (title, price, quantity...).
// Поля, которых нет в overrides, генерируются через NewAdvertisementParams
func (f *AdvertisementFactory) CreateWithParams(t *testing.T, ownerToken string, overrides map[string]string) advModels.Advertisement {
	params := NewAdvertisementParams()
	for field, value := range overrides {
		params[field] = value
	}

	photo := photoPath()
	require.FileExists(t, photo, "не найден файл фото для фикстуры объявления")

	adv := advertisement.CreateAdvertisement(t, ownerToken, params, []string{photo})
	require.NotEmpty(t, adv.Id, "у созданного объявления нет ID")

	f.created = append(f.created, createdAdvertisement{id: adv.Id, ownerToken: ownerToken})

	return adv
}

// Delete удаляет объявление прямо в тесте (например, для проверки каскада)
// и снимает его с очистки, чтобы Cleanup не пытался удалить его повторно
func (f *AdvertisementFactory) Delete(t *testing.T, ownerToken, advertisementID string) {
	advertisement.DeleteAdvertisement(t, ownerToken, advertisementID)

	remaining := make([]createdAdvertisement, 0, len(f.created))
	for _, ad := range f.created {
		if ad.id != advertisementID {
			remaining = append(remaining, ad)
		}
	}
	f.created = remaining
}

var deleteBackoff = []time.Duration{0, 200 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}

// deleteWithRetry удаляет объявление, повторяя попытки по расписанию deleteBackoff.
func deleteWithRetry(t *testing.T, ad createdAdvertisement) error {
	var lastErr error

	for attempt, pause := range deleteBackoff {
		time.Sleep(pause)

		status, err := advertisement.DeleteAdvertisementStatus(t, ad.ownerToken, ad.id)

		switch {
		case err != nil:
			lastErr = err
		case status == http.StatusNoContent || status == http.StatusNotFound:
			return nil
		case status == http.StatusBadGateway ||
			status == http.StatusServiceUnavailable ||
			status == http.StatusGatewayTimeout:
			lastErr = fmt.Errorf("статус %d", status)
		default:
			return fmt.Errorf("неожиданный статус %d", status)
		}

		utils.LogWithLabelAndTimeStamp("Precondition",
			fmt.Sprintf("DELETE %s: попытка %d не удалась: %v", ad.id, attempt+1, lastErr))
	}

	return lastErr
}
func (f *AdvertisementFactory) Cleanup(t *testing.T) {
	var failed []createdAdvertisement

	for i, ad := range f.created {
		if i > 0 {
			time.Sleep(cleanupDelay)
		}

		utils.LogWithLabelAndTimeStamp("Precondition", "Удаляем созданное объявление с ID: "+ad.id)

		if err := deleteWithRetry(t, ad); err != nil {
			t.Errorf("не удалось удалить объявление %s: %v", ad.id, err)
			failed = append(failed, ad) // останется в списке, дочистим в TearDownSuite
		}
	}

	f.created = failed
}
func photoPath() string {
	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(file), "testdata", "advertisement.jpg")
}

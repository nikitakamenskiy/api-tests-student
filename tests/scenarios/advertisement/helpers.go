package advertisement

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// newAdvertisementPhotos возвращает пути к тестовым фотографиям
func newAdvertisementPhotos(dir string) []string {
	base := filepath.Join(dir, "images")

	return []string{
		filepath.Join(base, "advertisementImage1.jpg"),
		filepath.Join(base, "advertisementImage2.jpg"),
		filepath.Join(base, "advertisementImage3.jpg"),
	}
}

// newLargePhoto возвращает путь к файлу фото размером > 5MB
func newLargePhoto(dir string) []string {
	return []string{filepath.Join(dir, "images", "largeImage.png")}
}

// setFormField задаёт или удаляет поле формы. Падает, если поля нет среди параметров по умолчанию,
// чтобы опечатка в названии поля в кейсе не превращала тест в ложно-зелёный
func setFormField(t *testing.T, params map[string]string, field, value string, omit bool) {
	t.Helper()

	_, exists := params[field]
	require.True(t, exists, "в кейсе указано неизвестное поле %q", field)

	if omit {
		delete(params, field)
		return
	}

	params[field] = value
}

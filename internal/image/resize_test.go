package image_test

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	imagePreviewer "github.com/romangricuk/image-previewer/internal/image"
	"github.com/romangricuk/image-previewer/internal/logger"
)

func TestResizeImage(t *testing.T) {
	// Инициализация логгера для тестов
	log := logger.NewTestLogger()

	// Путь к тестовому изображению
	testImagePath := filepath.Join("..", "..", "testdata", "test_image.jpg")

	// Открываем файл изображения
	file, err := os.Open(testImagePath)
	if err != nil {
		t.Fatalf("Failed to open test image: %v", err)
	}
	defer file.Close()

	// Читаем данные изображения
	originalImageData := new(bytes.Buffer)
	_, err = originalImageData.ReadFrom(file)
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Определение размеров для изменения
	width := 100
	height := 100

	// Вызов функции ResizeImage
	resizedData, err := imagePreviewer.ResizeImage(context.Background(), originalImageData.Bytes(), width, height, log)
	if err != nil {
		t.Fatalf("ResizeImage failed: %v", err)
	}

	// Проверка, что данные не пустые
	if len(resizedData) == 0 {
		t.Fatalf("Resized image data is empty")
	}

	// Декодирование полученного изображения
	img, _, err := image.Decode(bytes.NewReader(resizedData))
	if err != nil {
		t.Fatalf("Failed to decode resized image: %v", err)
	}

	// Проверка размеров изображения
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("Expected image size %dx%d, got %dx%d", width, height, img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestResizeImageWithCancelledContext(t *testing.T) {
	// Инициализация логгера для тестов
	log := logger.NewTestLogger()

	// Путь к тестовому изображению
	testImagePath := filepath.Join("..", "..", "testdata", "test_image.jpg")

	// Открываем файл изображения
	file, err := os.Open(testImagePath)
	if err != nil {
		t.Fatalf("Failed to open test image: %v", err)
	}
	defer file.Close()

	// Читаем данные изображения
	originalImageData := new(bytes.Buffer)
	_, err = originalImageData.ReadFrom(file)
	if err != nil {
		t.Fatalf("Failed to read test image: %v", err)
	}

	// Создание отмененного контекста
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Вызов функции ResizeImage с отмененным контекстом
	_, err = imagePreviewer.ResizeImage(ctx, originalImageData.Bytes(), 100, 100, log)
	if err == nil {
		t.Fatalf("Expected error due to cancelled context, but got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled error, got %v", err)
	}
}

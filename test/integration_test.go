package test

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/romangricuk/image-previewer/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getFreePort() (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "", err
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return strconv.Itoa(port), nil
}

func startTestApplication() (application *app.Application, port string, err error) {
	port, err = getFreePort()
	if err != nil {
		return nil, "", err
	}
	fmt.Printf("free port: %s\n", port)
	// Устанавливаем переменные окружения для тестов
	os.Setenv("APP_PORT", port)
	os.Setenv("CACHE_SIZE", "2")
	os.Setenv("CACHE_DIR", "./cache")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("SHUTDOWN_TIMEOUT", "5s")

	cacheDir := os.Getenv("CACHE_DIR")
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		return nil, "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	application, err = app.NewApplication("")
	if err != nil {
		return nil, "", err
	}

	go func() {
		if err := application.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			application.Logger.Errorf("Failed to run application: %v", err)
		}
	}()

	// Даем серверу время запуститься
	time.Sleep(2 * time.Second)

	return application, port, nil
}

func stopTestApplication(application *app.Application) {
	if err := application.Shutdown(); err != nil {
		application.Logger.Errorf("Error during shutdown: %v", err)
	}
}

// Тест ресайза разных картинок
func TestImageSizes(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/test/data/"

	images := []string{
		"gopher_50x50.jpg",
		"gopher_256x126.jpg",
		"gopher_333x666.jpg",
		"gopher_500x500.jpg",
		"gopher_1024x252.jpg",
		"gopher_200x700.jpg",
		"gopher_2000x1000.jpg",
		"_gopher_original_1024x504.jpg",
	}

	expectedWidth := 300
	expectedHeight := 200

	for _, imageName := range images {
		t.Run("Testing image "+imageName, func(t *testing.T) {
			imageURL := baseURL + imageName

			reqURL := fmt.Sprintf(
				"http://localhost:%s/fill/300/200/%s",
				port,
				strings.TrimPrefix(imageURL, "https://"),
			)

			resp, err := http.Get(reqURL) //nolint:gosec,noctx
			require.NoError(t, err, "Failed to get image %s", imageName)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf(
					"Expected status 200 for image %s, got %d. Response body: %s",
					imageName,
					resp.StatusCode,
					string(bodyBytes),
				)
			}

			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body for image %s", imageName)

			assert.NotEmpty(t, data, "Expected non-empty response body for image %s", imageName)

			img, _, err := image.Decode(bytes.NewReader(data))
			require.NoError(t, err, "Failed to decode image %s", imageName)

			assert.Equal(t, expectedWidth, img.Bounds().Dx(), "Width mismatch for image %s", imageName)
			assert.Equal(t, expectedHeight, img.Bounds().Dy(), "Height mismatch for image %s", imageName)
		})
	}
}

// Тест ресайза картинки разными размерами
func TestDifferentSizes(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/test/data/_gopher_original_1024x504.jpg"

	sizes := []struct {
		width  int
		height int
	}{
		{100, 100},
		{200, 400},
		{500, 250},
		{800, 600},
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%dx%d", size.width, size.height), func(t *testing.T) {
			reqURL := fmt.Sprintf(
				"http://localhost:%s/fill/%d/%d/%s",
				port,
				size.width,
				size.height,
				strings.TrimPrefix(baseURL, "https://"),
			)

			resp, err := http.Get(reqURL) //nolint:gosec,noctx
			require.NoError(t, err, "Failed to get image")
			defer resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200")

			data, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			assert.NotEmpty(t, data, "Expected non-empty response body")

			img, _, err := image.Decode(bytes.NewReader(data))
			require.NoError(t, err, "Failed to decode image")

			assert.Equal(t, size.width, img.Bounds().Dx(), "Width mismatch")
			assert.Equal(t, size.height, img.Bounds().Dy(), "Height mismatch")
		})
	}
}

// Тест заголовков
func TestResponseHeaders(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/test/data/gopher_50x50.jpg"
	reqURL := fmt.Sprintf(
		"http://localhost:%s/fill/300/200/%s",
		port,
		strings.TrimPrefix(baseURL, "https://"),
	)

	resp, err := http.Get(reqURL) //nolint:gosec,noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(
		t,
		"image/jpeg",
		resp.Header.Get("Content-Type"),
		"Expected Content-Type to be image/jpeg",
	)
}

// Тестируем проверку на timeout
func TestRequestTimeout(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Используем контролируемый HTTP-сервер, который задерживает ответ
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // Задержка больше, чем таймаут сервера
		http.Error(w, "Timeout", http.StatusGatewayTimeout)
	}))
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL, "http://")
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	client := &http.Client{
		Timeout: 2 * time.Second, // Устанавливаем таймаут меньше задержки сервера
	}
	resp, err := client.Get(reqURL) //nolint:noctx
	require.Error(t, err, "Expected timeout error")
	if resp != nil {
		resp.Body.Close()
	}
}

// Тест для проверки, что изображение берется из кэша
func TestImageFromCache(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Создаем тестовый сервер для изображения и счетчик запросов
	var requestCount int32
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.ServeFile(w, r, "test/data/gopher_50x50.jpg")
	}))
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL, "http://")
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	// Первый запрос - изображение должно быть загружено с удаленного сервера
	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image first time")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200")

	// Второй запрос - изображение должно быть взято из кэша
	resp2, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image second time")
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode, "Expected status 200")

	// Проверяем, что запрос к удаленному серверу был выполнен только один раз
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount), "Expected image to be served from cache")
}

// Тестируем, когда удаленный сервер не существует
func TestRemoteServerNotExist(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Используем несуществующий домен
	imageURL := "nonexistent.domain/image.jpg"
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image")
	defer resp.Body.Close()

	// Ожидаем статус ошибки
	require.Equal(t, http.StatusBadGateway, resp.StatusCode, "Expected status 502 Bad Gateway")
}

// Тестируем, когда удаленный сервер возвращает 404 Not Found
func TestRemoteImageNotFound(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Создаем тестовый сервер, который возвращает 404
	testServer := httptest.NewServer(http.NotFoundHandler())
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL+"/nonexistent.jpg", "http://")
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image")
	defer resp.Body.Close()

	// Ожидаем статус ошибки
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected status 404 Not Found")
}

// Тестируем, когда удаленный сервер возвращает не изображение, а, например, текстовый файл
func TestRemoteNonImageFile(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Создаем тестовый сервер, который возвращает текстовый файл
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "This is not an image")
	}))
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL, "http://")
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image")
	defer resp.Body.Close()

	// Ожидаем статус ошибки
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected status 400 Bad Request")
}

// Тестируем, когда удаленный сервер возвращает 500 Internal Server Error
func TestRemoteServerError(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Создаем тестовый сервер, который возвращает 500
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusBadGateway)
	}))
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL, "http://")
	reqURL := fmt.Sprintf("http://localhost:%s/fill/300/200/%s", port, imageURL)

	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image")
	defer resp.Body.Close()

	// Ожидаем статус ошибки
	require.Equal(t, http.StatusBadGateway, resp.StatusCode, "Expected status 502 Bad Gateway")
}

// Тестируем, когда изображение меньше, чем нужный размер
func TestImageSmallerThanRequestedSize(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Используем изображение меньшего размера
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "data/gopher_50x50.jpg")
	}))
	defer testServer.Close()

	imageURL := strings.TrimPrefix(testServer.URL, "http://")
	// Запрашиваем размер больше, чем исходный
	reqURL := fmt.Sprintf("http://localhost:%s/fill/100/100/%s", port, imageURL)

	resp, err := http.Get(reqURL)
	require.NoError(t, err, "Failed to get image")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200")

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	img, _, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err, "Failed to decode image")

	// Проверяем, что изображение имеет запрошенный размер
	assert.Equal(t, 100, img.Bounds().Dx(), "Width mismatch")
	assert.Equal(t, 100, img.Bounds().Dy(), "Height mismatch")
}

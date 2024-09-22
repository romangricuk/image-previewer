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
	os.Setenv("CACHE_DIR", "../cache")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("SHUTDOWN_TIMEOUT", "5s")
	//os.Setenv("DISABLE_LOGGING", "true")

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

func TestImageSizes(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/testdata/"

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

			//nolint:gosec,noctx
			resp, err := http.Get(reqURL)
			require.NoError(t, err, "Failed to get image %s", imageName)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("Expected status 200 for image %s, got %d. Response body: %s", imageName, resp.StatusCode, string(bodyBytes))
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

func TestDifferentSizes(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	//nolint:lll
	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/testdata/_gopher_original_1024x504.jpg"

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

func TestResponseHeaders(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	baseURL := "https://raw.githubusercontent.com/romangricuk/image-previewer/master/testdata/gopher_50x50.jpg"
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

func TestRequestTimeout(t *testing.T) {
	application, port, err := startTestApplication()
	require.NoError(t, err)
	defer stopTestApplication(application)

	// Используем контролируемый HTTP-сервер, который задерживает ответ
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // Задержка больше, чем таймаут сервера
		http.ServeFile(w, r, "test_image.jpg")
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

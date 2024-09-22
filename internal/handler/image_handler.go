package handler

import (
	"context"
	"crypto/md5" //nolint:gosec
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/romangricuk/image-previewer/internal/cache"
	"github.com/romangricuk/image-previewer/internal/config"
	"github.com/romangricuk/image-previewer/internal/image"
	"github.com/romangricuk/image-previewer/internal/logger"
	"github.com/romangricuk/image-previewer/internal/utils"
)

func NewImageHandler(cfg *config.Config, log logger.Logger) http.HandlerFunc {
	lruCache := cache.NewLRUCache(cfg.CacheSize, log)

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cacheDir := cfg.CacheDir

		// Парсинг параметров запроса
		width, height, imageURL, err := parseRequestParameters(r, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cacheKey := fmt.Sprintf("%d_%d_%s", width, height, imageURL)
		log.Infof("Processing request for image: %s with size %dx%d", imageURL, width, height)

		// Проверяем наличие в кэше
		if cachedPath, found := getFromCache(lruCache, cacheKey, log); found {
			http.ServeFile(w, r, cachedPath)
			return
		}

		// Загрузка изображения
		data, statusCode, err := fetchImage(ctx, r, imageURL, log)
		if err != nil {
			if statusCode == http.StatusOK {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else {
				w.WriteHeader(statusCode)
				w.Write(data)
			}
			return
		}

		// Проверка изображения
		if err := validateImage(data, log); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Изменение размера изображения
		resizedData, err := resizeImage(ctx, data, width, height, log)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Сохранение в кэш
		if err := saveToCache(cacheDir, cacheKey, resizedData, lruCache, log); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Отправка изображения клиенту
		sendImageResponse(w, resizedData)
	}
}

func parseRequestParameters(r *http.Request, log logger.Logger) (int, int, string, error) {
	parts := strings.SplitN(r.URL.Path, "/", 5)
	if len(parts) < 5 {
		log.Warn("Invalid URL format")
		return 0, 0, "", fmt.Errorf("invalid URL format")
	}

	width, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Warnf("Invalid width: %v", err)
		return 0, 0, "", fmt.Errorf("invalid width")
	}

	height, err := strconv.Atoi(parts[3])
	if err != nil {
		log.Warnf("Invalid height: %v", err)
		return 0, 0, "", fmt.Errorf("invalid height")
	}

	imageURL := parts[4]

	return width, height, imageURL, nil
}

func getFromCache(cache *cache.LRUCache, cacheKey string, log logger.Logger) (string, bool) {
	if cachedPath, found := cache.Get(cacheKey); found {
		log.Debugf("Cache hit for key: %s", cacheKey)
		return cachedPath, true
	}
	log.Debugf("Cache miss for key: %s", cacheKey)
	return "", false
}

func fetchImage(ctx context.Context, r *http.Request, imageURL string, log logger.Logger) ([]byte, int, error) {
	resp, err := utils.FetchImage(ctx, r, imageURL, log)
	if err != nil {
		log.Errorf("Failed to fetch image: %v", err)
		return nil, http.StatusBadGateway, fmt.Errorf("failed to fetch image")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("Remote server returned status code: %d", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		return body, resp.StatusCode, fmt.Errorf("remote server error")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read image data: %v", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to read image data")
	}

	return data, http.StatusOK, nil
}

func validateImage(data []byte, log logger.Logger) error {
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		log.Warnf("Fetched file is not an image: content type %s", contentType)
		return fmt.Errorf("fetched file is not an image")
	}
	return nil
}

func resizeImage(ctx context.Context, data []byte, width, height int, log logger.Logger) ([]byte, error) {
	resizedData, err := image.ResizeImage(ctx, data, width, height, log)
	if err != nil {
		log.Errorf("Failed to resize image: %v", err)
		return nil, fmt.Errorf("failed to resize image")
	}
	return resizedData, nil
}

func saveToCache(cacheDir, cacheKey string, data []byte, cache *cache.LRUCache, log logger.Logger) error {
	cacheFileName := fmt.Sprintf("%x.jpg", md5.Sum([]byte(cacheKey))) //nolint:gosec
	cachePath := filepath.Join(cacheDir, cacheFileName)

	if err := os.WriteFile(cachePath, data, 0o600); err != nil {
		log.Errorf("Failed to save image to cache: %v", err)
		return fmt.Errorf("failed to save image to cache")
	}

	cache.Put(cacheKey, cachePath)
	log.Debugf("Image saved to cache: %s", cachePath)
	return nil
}

func sendImageResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

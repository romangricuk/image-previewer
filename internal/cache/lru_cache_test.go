package cache_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/romangricuk/image-previewer/internal/cache"
	"github.com/romangricuk/image-previewer/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUCache(t *testing.T) {
	// Инициализация логгера для тестов
	testLogger := logger.NewTestLogger()

	cacheDir := "./test_cache"
	os.Mkdir(cacheDir, 0o755)
	defer os.RemoveAll(cacheDir)

	c := cache.NewLRUCache(2, testLogger)
	cachePath1 := filepath.Join(cacheDir, "file1")
	cachePath2 := filepath.Join(cacheDir, "file2")
	cachePath3 := filepath.Join(cacheDir, "file3")

	// Создаем тестовые файлы
	os.WriteFile(cachePath1, []byte("data1"), 0o600)
	os.WriteFile(cachePath2, []byte("data2"), 0o600)
	os.WriteFile(cachePath3, []byte("data3"), 0o600)

	c.Put("key1", cachePath1)
	c.Put("key2", cachePath2)
	_, found := c.Get("key1")
	require.True(t, found, "Expected to find key1")

	c.Put("key3", cachePath3)
	_, found = c.Get("key2")
	assert.False(t, found, "Expected key2 to be evicted")

	// Проверяем, что файл key2 был удален
	_, err := os.Stat(cachePath2)
	assert.True(t, os.IsNotExist(err), "Expected file for key2 to be deleted")

	// Проверяем, что файлы key1 и key3 существуют
	_, err = os.Stat(cachePath1)
	assert.False(t, os.IsNotExist(err), "Expected file for key1 to exist")
	_, err = os.Stat(cachePath3)
	assert.False(t, os.IsNotExist(err), "Expected file for key3 to exist")
}

func TestLRUCache_ZeroCapacity(t *testing.T) {
	log := logger.NewTestLogger()
	c := cache.NewLRUCache(0, log)

	c.Put("key1", "path1")
	c.Put("key2", "path2")

	// Проверяем, что элементы не были добавлены
	if _, found := c.Get("key1"); found {
		t.Error("Expected not to find key1 in cache with zero capacity")
	}
	if _, found := c.Get("key2"); found {
		t.Error("Expected not to find key2 in cache with zero capacity")
	}
}

func TestLRUCache_RemoveNonexistentFile(_ *testing.T) {
	// Инициализация логгера для тестов
	log := logger.NewTestLogger()

	cacheDir := "./test_cache_nonexistent"
	os.Mkdir(cacheDir, 0o755)
	defer os.RemoveAll(cacheDir)

	c := cache.NewLRUCache(1, log)
	cachePath := filepath.Join(cacheDir, "file")

	// Не создаем файл на диске

	// Добавляем элемент в кэш
	c.Put("key", cachePath)

	// Добавляем еще один элемент, чтобы вызвать удаление предыдущего
	anotherCachePath := filepath.Join(cacheDir, "file2")
	os.WriteFile(anotherCachePath, []byte("data2"), 0o600)
	c.Put("key2", anotherCachePath)

	// Проверяем, что ошибок не произошло при попытке удалить несуществующий файл
}

func TestLRUCache_UpdateExistingItem(t *testing.T) {
	log := logger.NewTestLogger()
	c := cache.NewLRUCache(2, log)

	c.Put("key1", "path1")
	c.Put("key2", "path2")
	c.Put("key1", "new_path1")

	path, found := c.Get("key1")
	require.True(t, found, "Expected to find key1")
	assert.Equal(t, "new_path1", path, "Expected updated path for key1")

	// Поскольку key1 был обновлен, key2 должен быть следующим для удаления
	c.Put("key3", "path3")
	_, found = c.Get("key2")
	assert.False(t, found, "Expected key2 to be evicted after updating key1")
}

func TestLRUCache_GetNonexistentKey(t *testing.T) {
	log := logger.NewTestLogger()
	c := cache.NewLRUCache(2, log)

	c.Put("key1", "path1")
	_, found := c.Get("nonexistent")
	assert.False(t, found, "Expected not to find nonexistent key")
}

func TestLRUCache_Order(t *testing.T) {
	log := logger.NewTestLogger()
	c := cache.NewLRUCache(3, log)

	c.Put("key1", "path1")
	c.Put("key2", "path2")
	c.Put("key3", "path3")

	// Доступ к key1, чтобы сделать его недавно использованным
	_, found := c.Get("key1")
	require.True(t, found, "Expected to find key1")

	// Добавление нового элемента должно привести к удалению key2
	c.Put("key4", "path4")

	_, found = c.Get("key2")
	assert.False(t, found, "Expected key2 to be evicted")
	_, found = c.Get("key1")
	require.True(t, found, "Expected to find key1")
	_, found = c.Get("key3")
	require.True(t, found, "Expected to find key3")
	_, found = c.Get("key4")
	require.True(t, found, "Expected to find key4")
}

func TestLRUCache_ConcurrentAccess(t *testing.T) {
	log := logger.NewTestLogger()
	c := cache.NewLRUCache(100, log)
	var wg sync.WaitGroup

	// Горутины для записи
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			path := fmt.Sprintf("path%d", i)
			c.Put(key, path)
		}(i)
	}

	// Горутины для чтения
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			c.Get(key)
		}(i)
	}

	wg.Wait()

	// Проверяем, что все ключи находятся в кэше
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key%d", i)
		_, found := c.Get(key)
		assert.True(t, found, "Expected to find %s in cache", key)
	}
}

func TestLRUCache_FileDeletion(t *testing.T) {
	log := logger.NewTestLogger()
	cacheDir := "./test_cache_file_deletion"
	os.Mkdir(cacheDir, 0o755)
	defer os.RemoveAll(cacheDir)

	c := cache.NewLRUCache(2, log)
	cachePath1 := filepath.Join(cacheDir, "file1")
	cachePath2 := filepath.Join(cacheDir, "file2")
	cachePath3 := filepath.Join(cacheDir, "file3")

	// Создаем тестовые файлы
	os.WriteFile(cachePath1, []byte("data1"), 0o600)
	os.WriteFile(cachePath2, []byte("data2"), 0o600)
	os.WriteFile(cachePath3, []byte("data3"), 0o600)

	c.Put("key1", cachePath1)
	c.Put("key2", cachePath2)
	c.Put("key3", cachePath3)

	// Проверяем, что файл для key1 был удален
	_, err := os.Stat(cachePath1)
	assert.True(t, os.IsNotExist(err), "Expected file for key1 to be deleted")

	// Проверяем, что файлы для key2 и key3 существуют
	_, err = os.Stat(cachePath2)
	assert.False(t, os.IsNotExist(err), "Expected file for key2 to exist")
	_, err = os.Stat(cachePath3)
	assert.False(t, os.IsNotExist(err), "Expected file for key3 to exist")
}

package cache

import (
	"container/list"
	"os"
	"sync"

	"github.com/romangricuk/image-previewer/internal/logger"
)

type cacheItem struct {
	Key  string
	Path string
}

type LRUCache struct {
	capacity int
	items    map[string]*list.Element
	order    *list.List
	mutex    sync.Mutex
	log      logger.Logger
}

func NewLRUCache(capacity int, log logger.Logger) *LRUCache {
	if capacity <= 0 {
		log.Warn("Cache capacity must be greater than zero. Setting capacity to 0.")
		capacity = 0
	}

	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
		log:      log,
	}
}

func (c *LRUCache) Get(key string) (string, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.capacity == 0 {
		return "", false
	}

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		return elem.Value.(*cacheItem).Path, true
	}
	return "", false
}

func (c *LRUCache) Put(key, path string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Если емкость кэша равна 0, не добавляем новые элементы
	if c.capacity == 0 {
		c.log.Debugf("Cache capacity is zero. Skipping adding key: %s", key)
		return
	}

	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		elem.Value.(*cacheItem).Path = path
		c.log.Debugf("Updated cache item for key: %s", key)
		return
	}

	if c.order.Len() >= c.capacity {
		// Удаляем последний элемент
		elem := c.order.Back()
		if elem != nil {
			c.order.Remove(elem)
			item := elem.Value.(*cacheItem)
			delete(c.items, item.Key)
			// Удаляем файл с диска
			err := os.Remove(item.Path)
			if err != nil {
				c.log.Errorf("Failed to remove file from cache: %v", err)
			}
			c.log.Debugf("Evicted cache item for key: %s", item.Key)
		}
	}

	item := &cacheItem{Key: key, Path: path}
	elem := c.order.PushFront(item)
	c.items[key] = elem
	c.log.Debugf("Added new cache item for key: %s", key)
}

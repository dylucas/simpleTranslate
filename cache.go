package main

import (
	"container/list"
	"sync"
)

// cacheEntry 缓存条目值，存放翻译结果或识别结果
type cacheEntry struct {
	key   string
	value string
}

// lruCache 简单的 LRU 缓存，线程安全。
// 用于缓存翻译结果与语种识别结果，避免重复调用云 API。
type lruCache struct {
	mu        sync.Mutex
	capacity  int
	maxBytes  int
	usedBytes int
	items     map[string]*list.Element
	order     *list.List // front = 最近使用，back = 最久未用
}

// newLRUCache 创建指定容量的 LRU 缓存
func newLRUCache(capacity int, byteBudget ...int) *lruCache {
	if capacity < 1 {
		capacity = 1
	}
	maxBytes := 0
	if len(byteBudget) > 0 && byteBudget[0] > 0 {
		maxBytes = byteBudget[0]
	}
	return &lruCache{
		capacity: capacity,
		maxBytes: maxBytes,
		items:    make(map[string]*list.Element, capacity),
		order:    list.New(),
	}
}

// get 读取缓存，命中时将该条目移到队首。未命中返回 ("", false)
func (c *lruCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*cacheEntry).value, true
	}
	return "", false
}

// set 写入缓存。若达到容量上限，淘汰最久未使用的条目。
func (c *lruCache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entryBytes := len(key) + len(value)
	if c.maxBytes > 0 && entryBytes > c.maxBytes {
		if el, ok := c.items[key]; ok {
			c.removeElement(el)
		}
		return
	}
	if el, ok := c.items[key]; ok {
		c.usedBytes -= len(el.Value.(*cacheEntry).value)
		el.Value.(*cacheEntry).value = value
		c.usedBytes += len(value)
		c.order.MoveToFront(el)
		c.evict()
		return
	}
	el := c.order.PushFront(&cacheEntry{key: key, value: value})
	c.items[key] = el
	c.usedBytes += entryBytes
	c.evict()
}

func (c *lruCache) evict() {
	for c.order.Len() > c.capacity || (c.maxBytes > 0 && c.usedBytes > c.maxBytes) {
		if oldest := c.order.Back(); oldest != nil {
			c.removeElement(oldest)
		} else {
			return
		}
	}
}

func (c *lruCache) removeElement(el *list.Element) {
	entry := el.Value.(*cacheEntry)
	c.usedBytes -= len(entry.key) + len(entry.value)
	c.order.Remove(el)
	delete(c.items, entry.key)
}

// clear 清空缓存（主要用于凭据变更后失效旧结果）
func (c *lruCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element, c.capacity)
	c.order.Init()
	c.usedBytes = 0
}

func (c *lruCache) bytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.usedBytes
}

// len 返回当前缓存条目数（主要用于测试与调试）
func (c *lruCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

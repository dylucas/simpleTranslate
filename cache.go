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
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List // front = 最近使用，back = 最久未用
}

// newLRUCache 创建指定容量的 LRU 缓存
func newLRUCache(capacity int) *lruCache {
	if capacity < 1 {
		capacity = 1
	}
	return &lruCache{
		capacity: capacity,
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
	if el, ok := c.items[key]; ok {
		el.Value.(*cacheEntry).value = value
		c.order.MoveToFront(el)
		return
	}
	el := c.order.PushFront(&cacheEntry{key: key, value: value})
	c.items[key] = el
	if c.order.Len() > c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
}

// clear 清空缓存（主要用于凭据变更后失效旧结果）
func (c *lruCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element, c.capacity)
	c.order.Init()
}

// len 返回当前缓存条目数（主要用于测试与调试）
func (c *lruCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

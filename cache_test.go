package main

import (
	"strings"
	"testing"
)

// TestLRUCache_GetSet 基本读写命中
func TestLRUCache_GetSet(t *testing.T) {
	c := newLRUCache(3)
	c.set("a", "1")
	c.set("b", "2")
	if v, ok := c.get("a"); !ok || v != "1" {
		t.Errorf("期望命中 a=1，得到 %q,%v", v, ok)
	}
	if _, ok := c.get("missing"); ok {
		t.Error("未写入的 key 不应命中")
	}
}

func TestLRUCache_ByteBudget(t *testing.T) {
	c := newLRUCache(10, 6)
	c.set("a", "12")
	c.set("b", "345")
	if _, ok := c.get("a"); ok {
		t.Fatal("oldest entry should be evicted when byte budget is exceeded")
	}
	if c.bytes() > 6 {
		t.Fatalf("cache bytes = %d, want <= 6", c.bytes())
	}
}

func TestLRUCache_RejectsOversizedAndTracksUpdates(t *testing.T) {
	c := newLRUCache(10, 5)
	c.set("a", "123")
	if c.bytes() != 4 {
		t.Fatalf("bytes = %d, want 4", c.bytes())
	}
	c.set("a", "1")
	if c.bytes() != 2 {
		t.Fatalf("updated bytes = %d, want 2", c.bytes())
	}
	c.set("huge", "12")
	if _, ok := c.get("huge"); ok {
		t.Fatal("oversized entry must not be cached")
	}
}

// TestLRUCache_Eviction 达到容量上限时淘汰最久未使用
func TestLRUCache_Eviction(t *testing.T) {
	c := newLRUCache(2)
	c.set("a", "1")
	c.set("b", "2")
	// 访问 a，使 b 成为最久未用
	c.get("a")
	c.set("c", "3") // 应淘汰 b
	if _, ok := c.get("b"); ok {
		t.Error("b 应被淘汰")
	}
	if _, ok := c.get("a"); !ok {
		t.Error("a 应保留（最近访问过）")
	}
	if _, ok := c.get("c"); !ok {
		t.Error("c 应存在")
	}
}

// TestLRUCache_Update 已有 key 更新值并移到队首
func TestLRUCache_Update(t *testing.T) {
	c := newLRUCache(2)
	c.set("a", "1")
	c.set("b", "2")
	c.set("a", "updated") // a 移到队首，b 成为最久未用
	c.set("c", "3")       // 淘汰 b
	if v, _ := c.get("a"); v != "updated" {
		t.Errorf("a 期望 updated，得到 %q", v)
	}
	if _, ok := c.get("b"); ok {
		t.Error("b 应被淘汰")
	}
}

// TestLRUCache_Clear 清空缓存
func TestLRUCache_Clear(t *testing.T) {
	c := newLRUCache(3)
	c.set("a", "1")
	c.set("b", "2")
	c.clear()
	if c.len() != 0 {
		t.Errorf("清空后长度期望 0，得到 %d", c.len())
	}
	if _, ok := c.get("a"); ok {
		t.Error("清空后不应命中")
	}
}

// TestLRUCache_Concurrent 并发读写不 panic（基础竞态检测）
func TestLRUCache_Concurrent(t *testing.T) {
	c := newLRUCache(10)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			c.set("k", "v")
			c.get("k")
		}
	}()
	for i := 0; i < 100; i++ {
		c.set("k2", "v2")
		c.get("k2")
	}
	<-done
}

// TestCacheKey 用分隔符避免拼接歧义
func TestCacheKey(t *testing.T) {
	k1 := cacheKey("a", "b", "c")
	k2 := cacheKey("ab", "c")
	if k1 == k2 {
		t.Errorf("不同分组应生成不同 key: %q == %q", k1, k2)
	}
}

func TestCacheKey_IsFixedDigestAndFieldSensitive(t *testing.T) {
	text := "secret source text"
	key := cacheKey("tencent", "en", "zh", text)
	if len(key) != 64 {
		t.Fatalf("digest length = %d, want 64", len(key))
	}
	if key == cacheKey("tencent", "en", "fr", text) || key == cacheKey("tencent", "en", "zh", text+"!") {
		t.Fatal("changing route or source text must change the digest")
	}
}

// BenchmarkCacheKey 测量缓存键计算的分配开销（优化后使用 sync.Pool + strconv）。
func BenchmarkCacheKey(b *testing.B) {
	text := strings.Repeat("x", 6000) // 模拟最大输入
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = versionedCacheKey(1, "baidu", "general", "zh", "en", text)
	}
}

// BenchmarkLRUCache_GetSet 模拟翻译结果缓存的读写压力。
func BenchmarkLRUCache_GetSet(b *testing.B) {
	c := newLRUCache(64, 512<<10)
	text := strings.Repeat("x", 500)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := cacheKey("tencent", "en", "zh", text)
		c.set(key, text)
		c.get(key)
	}
}

// BenchmarkLRUCache_ConcurrentStress 并发压力测试：验证缓存在线程安全下的稳定性。
func BenchmarkLRUCache_ConcurrentStress(b *testing.B) {
	c := newLRUCache(64, 512<<10)
	text := strings.Repeat("x", 500)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := cacheKey("tencent", "en", "zh", text)
			c.set(key, text)
			c.get(key)
		}
	})
}

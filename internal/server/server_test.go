package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/TBro1998/ArkPlayerPortal/internal/snapshot"
)

var testWeb = fstest.MapFS{
	"index.html":       {Data: []byte("<!doctype html><title>portal</title>")},
	"assets/app-1.js":  {Data: []byte("console.log(1)")},
	"assets/app-1.css": {Data: []byte("body{}")},
}

func newTest(t *testing.T) (*Server, *snapshot.Store, string) {
	t.Helper()
	dir := t.TempDir()
	store := snapshot.NewStore(dir)
	return New(store, testWeb, nil), store, dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func do(h http.Handler, method, target string, header map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct{ Code string }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("错误响应不是 JSON：%s", rec.Body.String())
	}
	return body.Code
}

func TestPortalAndShop(t *testing.T) {
	s, store, dir := newTest(t)
	h := s.Handler()

	if rec := do(h, "GET", "/api/v1/portal", nil); rec.Code != 503 || errCode(t, rec) != "not_ready" {
		t.Fatalf("没有快照时应当 503 not_ready：%d %s", rec.Code, rec.Body)
	}

	writeFile(t, dir, "portal.json", `{"site":{"name":"x"}}`)
	writeFile(t, dir, "shops/3.json", `{"items":[]}`)
	store.Scan()

	rec := do(h, "GET", "/api/v1/portal", nil)
	if rec.Code != 200 || rec.Body.String() != `{"site":{"name":"x"}}` {
		t.Fatalf("portal：%d %s", rec.Code, rec.Body)
	}
	if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("响应头不对：%v", rec.Header())
	}
	etag := rec.Header().Get("ETag")
	if rec := do(h, "GET", "/api/v1/portal", map[string]string{"If-None-Match": etag}); rec.Code != 304 || rec.Body.Len() != 0 {
		t.Fatalf("ETag 相同时应当 304：%d", rec.Code)
	}
	if rec := do(h, "GET", "/api/v1/portal", map[string]string{"If-None-Match": `"other", W/` + etag}); rec.Code != 304 {
		t.Fatalf("If-None-Match 里有一个匹配就应当 304：%d", rec.Code)
	}

	if rec := do(h, "GET", "/api/v1/shops/3", nil); rec.Code != 200 || rec.Body.String() != `{"items":[]}` {
		t.Fatalf("shop：%d %s", rec.Code, rec.Body)
	}
	for _, target := range []string{"/api/v1/shops/4", "/api/v1/shops/abc", "/api/v1/shops/..%2F3"} {
		if rec := do(h, "GET", target, nil); rec.Code != 404 || errCode(t, rec) != "shop_not_found" {
			t.Fatalf("%s 应当 404 shop_not_found：%d %s", target, rec.Code, rec.Body)
		}
	}
	if rec := do(h, "GET", "/api/v2/whatever", nil); rec.Code != 404 || errCode(t, rec) != "not_found" {
		t.Fatalf("不存在的接口应当 404 JSON：%d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "HEAD", "/api/v1/portal", nil); rec.Code != 200 || rec.Body.Len() != 0 {
		t.Fatalf("HEAD：%d %q", rec.Code, rec.Body)
	}
}

func TestMethodAndHeaders(t *testing.T) {
	s, _, _ := newTest(t)
	h := s.Handler()
	for _, m := range []string{"POST", "PUT", "DELETE", "OPTIONS"} {
		rec := do(h, m, "/api/v1/portal", nil)
		if rec.Code != 405 || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Fatalf("%s 应当 405：%d", m, rec.Code)
		}
	}
	rec := do(h, "GET", "/", nil)
	for _, k := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "X-Frame-Options"} {
		if rec.Header().Get(k) == "" {
			t.Errorf("缺少响应头 %s", k)
		}
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("不应当有 CORS 头")
	}
}

func TestWeb(t *testing.T) {
	s, _, _ := newTest(t)
	h := s.Handler()
	for _, target := range []string{"/", "/index.html", "/shop/3", "/assets/missing.js"} {
		rec := do(h, "GET", target, nil)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<title>portal</title>") {
			t.Fatalf("%s 应当返回 index.html：%d %s", target, rec.Code, rec.Body)
		}
		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("index.html 不应当缓存")
		}
	}
	rec := do(h, "GET", "/assets/app-1.js", nil)
	if rec.Code != 200 || rec.Body.String() != "console.log(1)" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("静态文件：%d %v", rec.Code, rec.Header())
	}

	empty := New(snapshot.NewStore(t.TempDir()), fstest.MapFS{}, nil).Handler()
	if rec := do(empty, "GET", "/", nil); rec.Code != 503 {
		t.Fatalf("没有构建网页时应当 503：%d", rec.Code)
	}
}

func TestRateLimit(t *testing.T) {
	s, _, _ := newTest(t)
	h := s.Handler()
	limited := 0
	for range requestBurst + 10 {
		rec := do(h, "GET", "/api/v1/portal", nil)
		if rec.Code == 429 {
			limited++
			if rec.Header().Get("Retry-After") == "" || errCode(t, rec) != "rate_limited" {
				t.Fatal("429 应当带 Retry-After 和 rate_limited")
			}
		}
	}
	if limited == 0 {
		t.Fatal("超过上限应当 429")
	}
	if rec := do(h, "GET", "/healthz", nil); rec.Code != 200 {
		t.Fatal("/healthz 不受限流影响")
	}

	l := newLimiter(1, 2)
	ip := netip.MustParseAddr("192.0.2.1")
	now := time.Unix(1000, 0)
	l.allow(ip, now)
	l.allow(ip, now)
	if _, ok := l.allow(ip, now); ok {
		t.Fatal("桶空了应当拒绝")
	}
	if _, ok := l.allow(ip, now.Add(time.Second)); !ok {
		t.Fatal("一秒后应当补充一个")
	}
	l.allow(netip.MustParseAddr("192.0.2.2"), now.Add(2*time.Minute))
	if len(l.buckets) != 1 {
		t.Fatalf("补满的桶应当清理掉，还剩 %d 个", len(l.buckets))
	}
}

func TestClientIP(t *testing.T) {
	trusted, err := ParseTrustedProxies("127.0.0.1, 10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTrustedProxies("nope"); err == nil {
		t.Fatal("无效的地址应当报错")
	}
	s := New(snapshot.NewStore(t.TempDir()), testWeb, trusted)
	cases := []struct {
		remote, xff, want string
	}{
		{"203.0.113.5:1234", "198.51.100.1", "203.0.113.5"},                   // 不是受信任的代理，忽略 XFF
		{"127.0.0.1:1234", "198.51.100.1", "198.51.100.1"},                    // 经过代理
		{"127.0.0.1:1234", "6.6.6.6, 198.51.100.1, 10.1.2.3", "198.51.100.1"}, // 从右往左第一个不受信任的
		{"127.0.0.1:1234", "", "127.0.0.1"},                                   // 没有 XFF
		{"127.0.0.1:1234", "garbage", "127.0.0.1"},                            // 解析不了时停在代理
		{"[::ffff:203.0.113.5]:1234", "", "203.0.113.5"},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = c.remote
		if c.xff != "" {
			req.Header.Set("X-Forwarded-For", c.xff)
		}
		if got := s.clientIP(req).String(); got != c.want {
			t.Errorf("%s + %q：得到 %s，应当是 %s", c.remote, c.xff, got, c.want)
		}
	}
}

func TestEvents(t *testing.T) {
	s, store, dir := newTest(t)
	s.pingInterval = 50 * time.Millisecond
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("SSE：%d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}

	lines := make(chan string, 100)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	waitFor := func(want string) {
		t.Helper()
		deadline := time.After(3 * time.Second)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatalf("连接断开，没有等到 %q", want)
				}
				if l == want {
					return
				}
			case <-deadline:
				t.Fatalf("没有等到 %q", want)
			}
		}
	}
	waitFor(": connected")
	waitFor(": ping")

	writeFile(t, dir, "shops/7.json", `{}`)
	store.Scan()
	waitFor("event: shop")
	waitFor(`data: {"key":"7"}`)

	writeFile(t, dir, "portal.json", `{}`)
	store.Scan()
	waitFor("event: portal")
	waitFor("data: {}")
}

func TestStreamLimit(t *testing.T) {
	s, _, _ := newTest(t)
	ip := netip.MustParseAddr("192.0.2.1")
	for range maxStreamsPerIP {
		if !s.acquireStream(ip) {
			t.Fatal("没到上限不应当拒绝")
		}
	}
	if s.acquireStream(ip) {
		t.Fatal("超过每个 IP 的上限应当拒绝")
	}
	if !s.acquireStream(netip.MustParseAddr("192.0.2.2")) {
		t.Fatal("别的 IP 不受影响")
	}
	s.releaseStream(ip)
	if !s.acquireStream(ip) {
		t.Fatal("释放后应当可以再连")
	}

	h := s.Handler()
	req := httptest.NewRequest("GET", "/api/v1/events", nil)
	req.RemoteAddr = "192.0.2.1:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 503 || errCode(t, rec) != "too_many_connections" {
		t.Fatalf("超过上限应当 503 too_many_connections：%d", rec.Code)
	}
}

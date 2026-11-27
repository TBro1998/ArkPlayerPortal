// Package server 玩家站的 HTTP 服务：快照接口、SSE 通知和网页（见 docs/api.md）
package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/netip"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/TBro1998/ArkPlayerPortal/internal/snapshot"
)

const (
	// 按来源 IP 的请求频率：每秒补充 requestRate 个，最多攒 requestBurst 个。
	// 联机房的玩家在玩家站看来来自同一个地址，所以比较宽
	requestRate  = 20
	requestBurst = 120
	// SSE 连接数上限：每个来源 IP、全站
	maxStreamsPerIP = 64
	maxStreams      = 1000
	pingInterval    = 25 * time.Second
)

const contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; " +
	"connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// Server 玩家站的 HTTP 服务
type Server struct {
	store   *snapshot.Store
	web     fs.FS // 网页的静态文件（Vite 的构建结果），根目录有 index.html
	trusted []netip.Prefix
	limiter *limiter

	streamMu    sync.Mutex
	streams     int
	streamsByIP map[netip.Addr]int

	// pingInterval 测试里改短
	pingInterval time.Duration
}

// New web 是网页的静态文件，trusted 是受信任的反向代理地址
func New(store *snapshot.Store, web fs.FS, trusted []netip.Prefix) *Server {
	return &Server{
		store:        store,
		web:          web,
		trusted:      trusted,
		limiter:      newLimiter(requestRate, requestBurst),
		streamsByIP:  map[netip.Addr]int{},
		pingInterval: pingInterval,
	}
}

// Handler 所有路由，外面套上方法检查、安全响应头和限流
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/v1/portal", s.handlePortal)
	mux.HandleFunc("GET /api/v1/shops/{key}", s.handleShop)
	mux.HandleFunc("GET /api/v1/events", s.handleEvents)
	mux.HandleFunc("GET /api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "接口不存在")
	})
	mux.Handle("GET /", s.webHandler())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			h.Set("Allow", "GET, HEAD")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "只支持 GET 请求")
			return
		}
		if r.URL.Path != "/healthz" {
			if wait, ok := s.limiter.allow(s.clientIP(r), time.Now()); !ok {
				h.Set("Retry-After", fmt.Sprint(int(wait/time.Second)+1))
				writeError(w, http.StatusTooManyRequests, "rate_limited", "请求过于频繁，请稍后再试")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) handlePortal(w http.ResponseWriter, r *http.Request) {
	doc := s.store.Portal()
	if doc == nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", "服主还没有发布玩家站的内容")
		return
	}
	writeDoc(w, r, doc)
}

func (s *Server) handleShop(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var doc *snapshot.Doc
	if snapshot.ValidKey(key) {
		doc = s.store.Shop(key)
	}
	if doc == nil {
		writeError(w, http.StatusNotFound, "shop_not_found", "这台服务器没有商店")
		return
	}
	writeDoc(w, r, doc)
}

// writeDoc 返回快照文件的内容，带 ETag，内容没变时 304
func writeDoc(w http.ResponseWriter, r *http.Request, doc *snapshot.Doc) {
	h := w.Header()
	h.Set("ETag", doc.ETag)
	h.Set("Cache-Control", "no-cache")
	if etagMatches(r.Header.Get("If-None-Match"), doc.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("Content-Length", fmt.Sprint(len(doc.Body)))
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = w.Write(doc.Body)
	}
}

func etagMatches(header, etag string) bool {
	for v := range strings.SplitSeq(header, ",") {
		v = strings.TrimSpace(v)
		if v == "*" || strings.TrimPrefix(v, "W/") == etag {
			return true
		}
	}
	return false
}

// handleEvents SSE：快照变化时推送 portal / shop 通知，每 25 秒一行注释保持连接
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	ip := s.clientIP(r)
	if !s.acquireStream(ip) {
		w.Header().Set("Retry-After", "30")
		writeError(w, http.StatusServiceUnavailable, "too_many_connections", "连接数太多，请稍后再试")
		return
	}
	defer s.releaseStream(ip)

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // 长连接，不受服务器的写超时限制
	events, cancel := s.store.Subscribe()
	defer cancel()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no") // 反向代理（nginx）不缓冲
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil || rc.Flush() != nil {
		return
	}

	ping := time.NewTicker(s.pingInterval)
	defer ping.Stop()
	for {
		var msg string
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			msg = ": ping\n\n"
		case e, ok := <-events:
			if !ok {
				return // 积压太多被断开，客户端重连后重新请求
			}
			data := []byte("{}")
			if e.Kind == "shop" {
				data, _ = json.Marshal(map[string]string{"key": e.Key})
			}
			msg = "event: " + e.Kind + "\ndata: " + string(data) + "\n\n"
		}
		if _, err := fmt.Fprint(w, msg); err != nil || rc.Flush() != nil {
			return
		}
	}
}

func (s *Server) acquireStream(ip netip.Addr) bool {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if s.streams >= maxStreams || s.streamsByIP[ip] >= maxStreamsPerIP {
		return false
	}
	s.streams++
	s.streamsByIP[ip]++
	return true
}

func (s *Server) releaseStream(ip netip.Addr) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	s.streams--
	if s.streamsByIP[ip]--; s.streamsByIP[ip] <= 0 {
		delete(s.streamsByIP, ip)
	}
}

// webHandler 网页：有这个文件就返回它，否则返回 index.html（单页应用的路由由前端处理）
func (s *Server) webHandler() http.Handler {
	files := http.FileServerFS(s.web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(s.web, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite 构建的文件名带内容哈希
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(s.web, "index.html")
		if err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("网页还没有构建"))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method != http.MethodHead {
			_, _ = w.Write(index)
		}
	})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code})
}

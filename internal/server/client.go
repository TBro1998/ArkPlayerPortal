package server

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// ParseTrustedProxies 解析 PORTAL_TRUSTED_PROXIES：逗号分隔的 IP 或 CIDR
func ParseTrustedProxies(v string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for part := range strings.SplitSeq(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if p, err := netip.ParsePrefix(part); err == nil {
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("PORTAL_TRUSTED_PROXIES 里的 %q 不是 IP 或 CIDR", part)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

func (s *Server) isTrusted(a netip.Addr) bool {
	for _, p := range s.trusted {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// clientIP 来源 IP：连接的地址；它是受信任的反向代理时，取 X-Forwarded-For 里从右往左第一个不受信任的地址
func (s *Server) clientIP(r *http.Request) netip.Addr {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}
	}
	addr = addr.Unmap()
	if !s.isTrusted(addr) {
		return addr
	}
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			break
		}
		a = a.Unmap()
		addr = a
		if !s.isTrusted(a) {
			break
		}
	}
	return addr
}

// limiter 按来源 IP 的令牌桶
type limiter struct {
	rate  float64
	burst float64

	mu        sync.Mutex
	buckets   map[netip.Addr]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(rate, burst float64) *limiter {
	return &limiter{rate: rate, burst: burst, buckets: map[netip.Addr]*bucket{}}
}

// allow 能不能处理这个请求；不能时返回要等多久
func (l *limiter) allow(ip netip.Addr, now time.Time) (time.Duration, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastSweep) > time.Minute {
		// 桶已经补满的 IP 不用再记
		for k, b := range l.buckets {
			if b.tokens+now.Sub(b.last).Seconds()*l.rate >= l.burst {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[ip] = b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens < 1 {
		return time.Duration((1 - b.tokens) / l.rate * float64(time.Second)), false
	}
	b.tokens--
	return 0, true
}

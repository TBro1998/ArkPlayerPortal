// Package snapshot 读取 Commander 写在数据目录里的快照文件（见 docs/snapshot.md），
// 定期检查是否变化，变化时通知订阅者。
package snapshot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	portalFile = "portal.json"
	shopsDir   = "shops"
	// maxFileSize 单个快照文件的上限，超过的当作无效
	maxFileSize = 8 << 20
	// subscriberBuffer 每个订阅者最多积压的通知，满了就断开它（客户端重连后会重新请求）
	subscriberBuffer = 32
)

// shopFilePattern 商店文件名：服务器 ID（数字）+ .json
var shopFilePattern = regexp.MustCompile(`^([0-9]{1,10})\.json$`)

// ValidKey 是不是合法的服务器 key（商店文件名里的数字）
func ValidKey(key string) bool {
	return shopFilePattern.MatchString(key + ".json")
}

// Doc 一个快照文件的内容
type Doc struct {
	Body []byte
	ETag string // 带引号，可以直接放进 ETag 响应头
}

// Event 快照变化的通知。Kind 是 "portal" 或 "shop"，商店的 Key 是服务器 key
type Event struct {
	Kind string
	Key  string
}

// fileStat 用来判断文件是否变化
type fileStat struct {
	modTime time.Time
	size    int64
}

func statOf(info fs.FileInfo) fileStat {
	return fileStat{modTime: info.ModTime(), size: info.Size()}
}

// Store 内存里的快照，Scan 从数据目录更新
type Store struct {
	dir string

	mu     sync.RWMutex
	portal *Doc
	shops  map[string]*Doc
	seen   map[string]fileStat // 相对路径 → 上次读取时的状态（读到无效内容也记，文件不变就不再读）

	subMu sync.Mutex
	subs  map[chan Event]struct{}
}

// NewStore dir 是快照目录（容器里的 /data）
func NewStore(dir string) *Store {
	return &Store{
		dir:   dir,
		shops: map[string]*Doc{},
		seen:  map[string]fileStat{},
		subs:  map[chan Event]struct{}{},
	}
}

// Portal 当前的 portal.json，还没有时返回 nil
func (s *Store) Portal() *Doc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.portal
}

// Shop 某台服务器的商店，没有时返回 nil
func (s *Store) Shop(key string) *Doc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shops[key]
}

// Subscribe 订阅快照变化。返回的通道在 cancel 之后，或者订阅者积压太多通知时关闭
func (s *Store) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)
	s.subMu.Lock()
	s.subs[ch] = struct{}{}
	s.subMu.Unlock()
	return ch, func() {
		s.subMu.Lock()
		defer s.subMu.Unlock()
		if _, ok := s.subs[ch]; ok {
			delete(s.subs, ch)
			close(ch)
		}
	}
}

func (s *Store) publish(events []Event) {
	if len(events) == 0 {
		return
	}
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for ch := range s.subs {
		if !deliver(ch, events) {
			// 积压太多，断开这个订阅者
			delete(s.subs, ch)
			close(ch)
		}
	}
}

func deliver(ch chan Event, events []Event) bool {
	for _, e := range events {
		select {
		case ch <- e:
		default:
			return false
		}
	}
	return true
}

// Watch 每隔 interval 检查一次数据目录，直到 ctx 结束
func (s *Store) Watch(ctx context.Context, interval time.Duration) {
	s.Scan()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Scan()
		}
	}
}

// Scan 检查一次数据目录，读取变化了的文件，通知订阅者
func (s *Store) Scan() {
	root, err := os.OpenRoot(s.dir)
	if err != nil {
		// 目录不存在（Commander 还没写）时当作空的
		s.publish(s.clearAll())
		return
	}
	defer root.Close()

	var events []Event
	switch r, doc := s.check(root, portalFile); r {
	case updated, removed:
		s.mu.Lock()
		s.portal = doc
		s.mu.Unlock()
		events = append(events, Event{Kind: "portal"})
	}

	present := map[string]bool{}
	if entries, err := fs.ReadDir(root.FS(), shopsDir); err == nil {
		for _, e := range entries {
			m := shopFilePattern.FindStringSubmatch(e.Name())
			if m == nil || !e.Type().IsRegular() {
				continue
			}
			key := m[1]
			present[key] = true
			if r, doc := s.check(root, shopsDir+"/"+e.Name()); r == updated {
				s.mu.Lock()
				s.shops[key] = doc
				s.mu.Unlock()
				events = append(events, Event{Kind: "shop", Key: key})
			}
		}
	}
	s.mu.Lock()
	for key := range s.shops {
		if !present[key] {
			delete(s.shops, key)
			events = append(events, Event{Kind: "shop", Key: key})
		}
	}
	// 已经删掉的商店文件不再记状态，之后重新出现时一定重新读取
	for name := range s.seen {
		if key, ok := strings.CutPrefix(name, shopsDir+"/"); ok && !present[strings.TrimSuffix(key, ".json")] {
			delete(s.seen, name)
		}
	}
	s.mu.Unlock()

	s.publish(events)
}

// clearAll 数据目录不在了：清空所有内容
func (s *Store) clearAll() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	var events []Event
	if s.portal != nil {
		events = append(events, Event{Kind: "portal"})
	}
	for key := range s.shops {
		events = append(events, Event{Kind: "shop", Key: key})
	}
	s.portal = nil
	s.shops = map[string]*Doc{}
	s.seen = map[string]fileStat{}
	return events
}

type checkResult int

const (
	unchanged checkResult = iota
	updated               // 读到了新的合法内容
	removed               // 文件不在了（之前读到过）
	invalid               // 变了但读不出合法的 JSON，保留旧内容
)

// check 文件和上次读取时相比是否变化，变化时读取
func (s *Store) check(root *os.Root, name string) (checkResult, *Doc) {
	info, err := root.Stat(name)
	if err != nil {
		s.mu.Lock()
		_, had := s.seen[name]
		delete(s.seen, name)
		s.mu.Unlock()
		if had && errors.Is(err, fs.ErrNotExist) {
			return removed, nil
		}
		return unchanged, nil
	}
	st := statOf(info)
	s.mu.RLock()
	prev, had := s.seen[name]
	s.mu.RUnlock()
	if had && prev == st {
		return unchanged, nil
	}

	// 读到无效内容也记下状态：文件不再变化就不重复读取和记日志
	body, err := readLimited(root, name)
	s.mu.Lock()
	s.seen[name] = st
	s.mu.Unlock()
	if err != nil {
		log.Printf("读取 %s 失败：%v", name, err)
		return invalid, nil
	}
	if !json.Valid(body) {
		log.Printf("%s 不是合法的 JSON，继续使用上一次的内容", name)
		return invalid, nil
	}
	sum := sha256.Sum256(body)
	return updated, &Doc{Body: body, ETag: `"` + hex.EncodeToString(sum[:12]) + `"`}
}

func readLimited(root *os.Root, name string) ([]byte, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := io.ReadAll(io.LimitReader(f, maxFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxFileSize {
		return nil, fmt.Errorf("文件超过 %d MB，不读取", maxFileSize>>20)
	}
	return body, nil
}

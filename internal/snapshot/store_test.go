package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// write 先写临时文件再改名，和 Commander 的写法一样；修改时间往后推，保证能看出变化
func write(t *testing.T, dir, name, content string, age int) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Now().Add(time.Duration(age) * time.Second)
	if err := os.Chtimes(tmp, mt, mt); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, p); err != nil {
		t.Fatal(err)
	}
}

func drain(ch <-chan Event) []Event {
	var out []Event
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, e)
		default:
			return out
		}
	}
}

func TestScanPortal(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "missing"))
	events, cancel := s.Subscribe()
	defer cancel()

	s.Scan()
	if s.Portal() != nil || len(drain(events)) != 0 {
		t.Fatal("目录不存在时应当没有内容、没有通知")
	}

	s = NewStore(dir)
	events, cancel2 := s.Subscribe()
	defer cancel2()
	write(t, dir, "portal.json", `{"a":1}`, 1)
	s.Scan()
	doc := s.Portal()
	if doc == nil || string(doc.Body) != `{"a":1}` || doc.ETag == "" {
		t.Fatalf("没有读到 portal.json：%+v", doc)
	}
	if got := drain(events); len(got) != 1 || got[0].Kind != "portal" {
		t.Fatalf("通知不对：%+v", got)
	}

	// 没变化：不通知
	s.Scan()
	if got := drain(events); len(got) != 0 {
		t.Fatalf("没有变化不应当通知：%+v", got)
	}

	// 无效的 JSON：保留旧内容，不通知；之后修好了照常读取
	write(t, dir, "portal.json", `{"a":`, 2)
	s.Scan()
	if string(s.Portal().Body) != `{"a":1}` || len(drain(events)) != 0 {
		t.Fatal("无效内容应当保留旧内容")
	}
	write(t, dir, "portal.json", `{"a":2}`, 3)
	s.Scan()
	if string(s.Portal().Body) != `{"a":2}` || s.Portal().ETag == doc.ETag {
		t.Fatal("修好后应当读到新内容和新的 ETag")
	}
	drain(events)

	// 删除：清空并通知
	if err := os.Remove(filepath.Join(dir, "portal.json")); err != nil {
		t.Fatal(err)
	}
	s.Scan()
	if s.Portal() != nil {
		t.Fatal("删除后应当没有内容")
	}
	if got := drain(events); len(got) != 1 || got[0].Kind != "portal" {
		t.Fatalf("删除后应当通知：%+v", got)
	}
}

func TestScanShops(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	events, cancel := s.Subscribe()
	defer cancel()

	write(t, dir, "shops/3.json", `{"items":[]}`, 1)
	write(t, dir, "shops/abc.json", `{}`, 1)   // 不是数字，忽略
	write(t, dir, "shops/5.json.tmp", `{}`, 1) // 临时文件，忽略
	s.Scan()
	if s.Shop("3") == nil || s.Shop("abc") != nil {
		t.Fatal("应当只读数字命名的商店文件")
	}
	if got := drain(events); len(got) != 1 || got[0] != (Event{Kind: "shop", Key: "3"}) {
		t.Fatalf("通知不对：%+v", got)
	}

	if err := os.Remove(filepath.Join(dir, "shops", "3.json")); err != nil {
		t.Fatal(err)
	}
	s.Scan()
	if s.Shop("3") != nil {
		t.Fatal("删除后应当没有商店")
	}
	if got := drain(events); len(got) != 1 || got[0] != (Event{Kind: "shop", Key: "3"}) {
		t.Fatalf("删除后应当通知：%+v", got)
	}

	// 重新出现（修改时间、大小和删除前一样）也要重新读取
	write(t, dir, "shops/3.json", `{"items":[]}`, 1)
	s.Scan()
	if s.Shop("3") == nil {
		t.Fatal("重新出现的商店文件应当读取")
	}
}

func TestTooLarge(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	big := make([]byte, maxFileSize+10)
	for i := range big {
		big[i] = ' '
	}
	big[0] = '1'
	write(t, dir, "portal.json", string(big), 1)
	s.Scan()
	if s.Portal() != nil {
		t.Fatal("超过上限的文件不应当读取")
	}
}

func TestSlowSubscriberDropped(t *testing.T) {
	s := NewStore(t.TempDir())
	events, cancel := s.Subscribe()
	defer cancel()
	for range subscriberBuffer + 5 {
		s.publish([]Event{{Kind: "portal"}})
	}
	n := 0
	for range events {
		n++
	}
	if n != subscriberBuffer {
		t.Fatalf("积压满了应当断开订阅者，收到 %d 条", n)
	}
	cancel() // 已经断开时再取消不会 panic
}

func TestValidKey(t *testing.T) {
	for key, want := range map[string]bool{"3": true, "1234567890": true, "": false, "a": false, "../3": false, "12345678901": false} {
		if ValidKey(key) != want {
			t.Errorf("ValidKey(%q) 应当是 %v", key, want)
		}
	}
}

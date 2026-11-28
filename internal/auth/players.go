package auth

import (
	"encoding/json"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"sync"
)

// 玩家自己的数据：Commander 写的 players/<服务器 key>.json（{"players": {"<SteamID>": {...}}}），不公开，
// 只按登录玩家的 SteamID 取出他自己的那一条。文件变了才重新读

const maxPlayersFile = 64 << 20

var playersFilePattern = regexp.MustCompile(`^([0-9]{1,10})\.json$`)

// Players 读取 players/ 目录
type Players struct {
	dir string

	mu    sync.Mutex
	files map[string]playersFile // key → 文件内容
}

type playersFile struct {
	stat    fileStat
	players map[string]json.RawMessage
}

// NewPlayers dir 是快照目录
func NewPlayers(dir string) *Players { return &Players{dir: dir, files: map[string]playersFile{}} }

// For 这个玩家在每台服务器上的数据（key → 原样的 JSON 对象），没有时返回空 map
func (p *Players) For(steamID string) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	root, err := os.OpenRoot(p.dir)
	if err != nil {
		return out
	}
	defer root.Close()
	entries, err := fs.ReadDir(root.FS(), "players")
	if err != nil {
		return out
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := map[string]bool{}
	for _, e := range entries {
		m := playersFilePattern.FindStringSubmatch(e.Name())
		if m == nil || !e.Type().IsRegular() {
			continue
		}
		key := m[1]
		seen[key] = true
		info, err := e.Info()
		if err != nil || info.Size() > maxPlayersFile {
			continue
		}
		st := fileStat{info.ModTime(), info.Size()}
		f, ok := p.files[key]
		if !ok || f.stat != st {
			data, err := fs.ReadFile(root.FS(), "players/"+e.Name())
			if err != nil {
				continue
			}
			var doc struct {
				Players map[string]json.RawMessage `json:"players"`
			}
			if json.Unmarshal(data, &doc) != nil {
				continue // 读到一半（Commander 正在替换）时下次再读
			}
			f = playersFile{stat: st, players: doc.Players}
			p.files[key] = f
		}
		if v, ok := f.players[steamID]; ok && strings.HasPrefix(strings.TrimSpace(string(v)), "{") {
			out[key] = v
		}
	}
	for key := range p.files {
		if !seen[key] {
			delete(p.files, key)
		}
	}
	return out
}

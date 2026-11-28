// Package auth 玩家登录：网页用 Steam OpenID 登录，联机助手用 arkfgf.com 签发的令牌换会话（不用再登录一次）。
// 登录后只能看自己的数据（ArkShop 点数等，Commander 写在快照目录的 players/ 里）。协议见 docs/api.md 的「登录」
package auth

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	identityFile = "identity.key"
	sessionTTL   = 30 * 24 * time.Hour
	// identityContext 玩家站证明身份时签名的内容前缀，和别的签名用途区分开
	identityContext = "arkfgf-portal-identity\n"
)

var steamIDPattern = regexp.MustCompile(`^[0-9]{17}$`)

// User 登录的玩家
type User struct {
	SteamID string `json:"steam_id"`
	Name    string `json:"name"`
	Avatar  string `json:"avatar"`
}

// identity 玩家站的身份密钥（Commander 生成，见 docs/snapshot.md）
type identity struct {
	priv       ed25519.PrivateKey
	pub        ed25519.PublicKey
	audience   string // 公钥 SHA-256 的十六进制，arkfgf.com 签发的令牌只对它有效
	sessionKey []byte
}

// Identity 管理身份密钥：文件变了（Commander 重新生成）就重新读取
type Identity struct {
	dir string

	mu    sync.Mutex
	stat  fileStat
	ident *identity
}

type fileStat struct {
	mod  time.Time
	size int64
}

// NewIdentity dir 是快照目录
func NewIdentity(dir string) *Identity { return &Identity{dir: dir} }

var errNoIdentity = errors.New("玩家站的身份密钥还没有生成")

func (i *Identity) get() (*identity, error) {
	root, err := os.OpenRoot(i.dir)
	if err != nil {
		return nil, errNoIdentity
	}
	defer root.Close()
	info, err := root.Stat(identityFile)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errNoIdentity
	}
	st := fileStat{info.ModTime(), info.Size()}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.ident != nil && i.stat == st {
		return i.ident, nil
	}
	data, err := fs.ReadFile(root.FS(), identityFile)
	if err != nil {
		return nil, errNoIdentity
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("玩家站的身份密钥无效")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	sum := sha256.Sum256(pub)
	sk := sha256.Sum256(append([]byte("arkfgf-portal-session\n"), seed...))
	i.ident = &identity{priv: priv, pub: pub, audience: hex.EncodeToString(sum[:]), sessionKey: sk[:]}
	i.stat = st
	return i.ident, nil
}

// Proof 证明玩家站持有这个身份：公钥、audience 和对 nonce 的签名（base64url）
type Proof struct {
	PublicKey string `json:"public_key"`
	Audience  string `json:"audience"`
	Signature string `json:"signature"`
}

// Prove 对联机助手给的 nonce 签名
func (i *Identity) Prove(nonce string) (Proof, error) {
	id, err := i.get()
	if err != nil {
		return Proof{}, err
	}
	sig := ed25519.Sign(id.priv, []byte(identityContext+nonce))
	return Proof{
		PublicKey: base64.RawURLEncoding.EncodeToString(id.pub),
		Audience:  id.audience,
		Signature: base64.RawURLEncoding.EncodeToString(sig),
	}, nil
}

// Audience 这个玩家站的身份
func (i *Identity) Audience() (string, error) {
	id, err := i.get()
	if err != nil {
		return "", err
	}
	return id.audience, nil
}

type sessionClaims struct {
	User
	Exp int64 `json:"exp"`
}

// IssueSession 签发玩家站自己的会话（HMAC，密钥从身份密钥派生；身份密钥换了以后旧会话作废）
func (i *Identity) IssueSession(u User, now time.Time) (string, time.Time, error) {
	id, err := i.get()
	if err != nil {
		return "", time.Time{}, err
	}
	exp := now.Add(sessionTTL)
	payload, _ := json.Marshal(sessionClaims{User: u, Exp: exp.Unix()})
	body := "v1." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, id.sessionKey)
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), exp, nil
}

// ParseSession 校验会话，返回玩家
func (i *Identity) ParseSession(token string, now time.Time) (User, bool) {
	id, err := i.get()
	if err != nil || len(token) > 2048 {
		return User{}, false
	}
	dot := strings.LastIndexByte(token, '.')
	if dot < 0 || !strings.HasPrefix(token, "v1.") {
		return User{}, false
	}
	body, sigPart := token[:dot], token[dot+1:]
	sig, err := base64.RawURLEncoding.DecodeString(sigPart)
	if err != nil {
		return User{}, false
	}
	mac := hmac.New(sha256.New, id.sessionKey)
	mac.Write([]byte(body))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return User{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(body, "v1."))
	if err != nil {
		return User{}, false
	}
	var c sessionClaims
	if json.Unmarshal(payload, &c) != nil || !steamIDPattern.MatchString(c.SteamID) || now.Unix() >= c.Exp {
		return User{}, false
	}
	return c.User, true
}

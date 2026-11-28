package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// arkfgf.com 签发的玩家站令牌（联机助手登录用）：EdDSA 签名的 JWT，iss=arkfgf.com、aud=玩家站的 audience、sub=SteamID64。
// 公钥从 PORTAL_ARKFGF_KEYS_URL 取（JWK），缓存 1 小时，遇到不认识的 kid 时最多每分钟重新取一次

const (
	tokenIssuer    = "arkfgf.com"
	keysCacheTTL   = time.Hour
	keysRefetchGap = time.Minute
)

// ArkfgfKeys 取 arkfgf.com 的公钥并校验令牌
type ArkfgfKeys struct {
	URL  string
	HTTP *http.Client

	mu      sync.Mutex
	keys    map[string]ed25519.PublicKey
	fetched time.Time
	tried   time.Time
}

// Enabled 配置了公钥地址
func (k *ArkfgfKeys) Enabled() bool { return k != nil && k.URL != "" }

func (k *ArkfgfKeys) key(ctx context.Context, kid string, now time.Time) (ed25519.PublicKey, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if pub, ok := k.keys[kid]; ok && now.Sub(k.fetched) < keysCacheTTL {
		return pub, nil
	}
	if now.Sub(k.tried) < keysRefetchGap {
		if pub, ok := k.keys[kid]; ok {
			return pub, nil
		}
		return nil, errors.New("arkfgf.com 的公钥里没有这个令牌的 kid")
	}
	k.tried = now
	keys, err := k.fetch(ctx)
	if err != nil {
		if pub, ok := k.keys[kid]; ok {
			return pub, nil // 取不到新的时继续用旧的
		}
		return nil, err
	}
	k.keys, k.fetched = keys, now
	if pub, ok := keys[kid]; ok {
		return pub, nil
	}
	return nil, errors.New("arkfgf.com 的公钥里没有这个令牌的 kid")
}

func (k *ArkfgfKeys) fetch(ctx context.Context) (map[string]ed25519.PublicKey, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.URL, nil)
	if err != nil {
		return nil, err
	}
	client := k.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("取 arkfgf.com 的公钥失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("取 arkfgf.com 的公钥失败：%d", resp.StatusCode)
	}
	var body struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
		return nil, err
	}
	keys := map[string]ed25519.PublicKey{}
	for _, j := range body.Keys {
		x, err := base64.RawURLEncoding.DecodeString(j.X)
		if j.Kty == "OKP" && j.Crv == "Ed25519" && err == nil && len(x) == ed25519.PublicKeySize {
			keys[j.Kid] = ed25519.PublicKey(x)
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("arkfgf.com 没有返回可用的公钥")
	}
	return keys, nil
}

// Verify 校验 arkfgf.com 签发的令牌：签名、签发方、audience、有效期，返回玩家
func (k *ArkfgfKeys) Verify(ctx context.Context, token, audience string, now time.Time) (User, error) {
	parts := strings.Split(token, ".")
	if len(token) > 4096 || len(parts) != 3 {
		return User{}, errors.New("令牌格式不对")
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if raw, err := base64.RawURLEncoding.DecodeString(parts[0]); err != nil || json.Unmarshal(raw, &header) != nil || header.Alg != "EdDSA" {
		return User{}, errors.New("令牌的签名算法不对")
	}
	pub, err := k.key(ctx, header.Kid, now)
	if err != nil {
		return User{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
		return User{}, errors.New("令牌的签名无效")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return User{}, errors.New("令牌格式不对")
	}
	var c struct {
		Iss    string `json:"iss"`
		Aud    string `json:"aud"`
		Sub    string `json:"sub"`
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
		Exp    int64  `json:"exp"`
	}
	if json.Unmarshal(raw, &c) != nil {
		return User{}, errors.New("令牌格式不对")
	}
	switch {
	case c.Iss != tokenIssuer:
		return User{}, errors.New("令牌不是 arkfgf.com 签发的")
	case c.Aud != audience:
		return User{}, errors.New("令牌不是给这个玩家站的")
	case now.Unix() >= c.Exp:
		return User{}, errors.New("令牌已过期")
	case !steamIDPattern.MatchString(c.Sub):
		return User{}, errors.New("令牌里的 SteamID 无效")
	}
	return User{SteamID: c.Sub, Name: limitRunes(c.Name, 64), Avatar: safeAvatar(c.Avatar)}, nil
}

func limitRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

// safeAvatar 只接受 Steam 头像地址（https，Steam 的 CDN），其他的丢掉
func safeAvatar(u string) string {
	if len(u) < 512 && (strings.HasPrefix(u, "https://avatars.steamstatic.com/") || strings.HasPrefix(u, "https://avatars.akamai.steamstatic.com/") ||
		strings.HasPrefix(u, "https://avatars.cloudflare.steamstatic.com/")) {
		return u
	}
	return ""
}

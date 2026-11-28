package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TBro1998/ArkPlayerPortal/internal/auth"
	"github.com/TBro1998/ArkPlayerPortal/internal/snapshot"
)

func signEdToken(priv ed25519.PrivateKey, claims map[string]any) string {
	h, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": "k1"})
	p, _ := json.Marshal(claims)
	body := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	return body + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(body)))
}

func newLoginTest(t *testing.T, keysURL string) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 5
	if err := os.WriteFile(filepath.Join(dir, "identity.key"), []byte(base64.StdEncoding.EncodeToString(seed)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "players"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "players", "3.json"), []byte(`{"players":{"76561198000000001":{"points":120}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	login := &Login{
		Identity: auth.NewIdentity(dir),
		Players:  auth.NewPlayers(dir),
		Arkfgf:   &auth.ArkfgfKeys{URL: keysURL},
		Steam:    &auth.Steam{},
	}
	return New(snapshot.NewStore(dir), testWeb, nil).WithLogin(login).Handler(), dir
}

func TestLinkLoginFlow(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	seed[7] = 1
	site := ed25519.NewKeyFromSeed(seed)
	keys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kid": "k1", "kty": "OKP", "crv": "Ed25519",
			"x": base64.RawURLEncoding.EncodeToString(site.Public().(ed25519.PublicKey))}}})
	}))
	defer keys.Close()
	h, _ := newLoginTest(t, keys.URL)

	// 1. 身份证明
	rec := do(h, "GET", "/api/v1/identity?nonce=0123456789abcdef", nil)
	var proof auth.Proof
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &proof) != nil || proof.Audience == "" {
		t.Fatalf("identity: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/api/v1/identity?nonce=short", nil); rec.Code != 400 {
		t.Fatalf("短 nonce 应当 400: %d", rec.Code)
	}

	// 2. 用 arkfgf.com 签的令牌换会话
	claims := map[string]any{"iss": "arkfgf.com", "aud": proof.Audience, "sub": "76561198000000001", "name": "玩家", "exp": time.Now().Add(time.Minute).Unix()}
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	rec = post(`{"token":"` + signEdToken(site, claims) + `"}`)
	var out struct {
		Session string    `json:"session"`
		User    auth.User `json:"user"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &out) != nil || out.Session == "" || out.User.SteamID != "76561198000000001" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Error("令牌登录不应当设置 cookie")
	}
	claims["aud"] = "someone-else"
	if rec := post(`{"token":"` + signEdToken(site, claims) + `"}`); rec.Code != 401 || errCode(t, rec) != "invalid_token" {
		t.Fatalf("别的玩家站的令牌应当 401: %d %s", rec.Code, rec.Body)
	}

	// 3. 用会话看自己的数据
	rec = do(h, "GET", "/api/v1/me", map[string]string{"Authorization": "Bearer " + out.Session})
	var me struct {
		User *auth.User                 `json:"user"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &me) != nil || me.User == nil || string(me.Data["3"]) != `{"points":120}` {
		t.Fatalf("me: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, "GET", "/api/v1/me", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"user":null`) || !strings.Contains(rec.Body.String(), `"link_login":true`) {
		t.Fatalf("没登录时 user 为 null: %s", rec.Body)
	}
	if rec := do(h, "GET", "/api/v1/me", map[string]string{"Authorization": "Bearer bogus"}); !strings.Contains(rec.Body.String(), `"user":null`) {
		t.Fatalf("无效会话: %s", rec.Body)
	}

	// 其他路径仍然不接受 POST
	req := httptest.NewRequest("POST", "/api/v1/portal", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 405 {
		t.Fatalf("POST /api/v1/portal 应当 405: %d", w.Code)
	}
}

func TestTokenLoginDisabled(t *testing.T) {
	h, _ := newLoginTest(t, "")
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(`{"token":"x"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 404 || errCode(t, w) != "link_login_disabled" {
		t.Fatalf("没有配置公钥地址时应当 404: %d %s", w.Code, w.Body)
	}
}

func TestSteamLoginRedirectAndState(t *testing.T) {
	h, _ := newLoginTest(t, "")
	req := httptest.NewRequest("GET", "http://1.2.3.4:8090/auth/steam/login", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	loc, _ := url.Parse(w.Header().Get("Location"))
	if w.Code != 302 || loc.Host != "steamcommunity.com" || !strings.HasPrefix(loc.Query().Get("openid.return_to"), "http://1.2.3.4:8090/auth/steam/callback?state=") {
		t.Fatalf("login redirect: %d %s", w.Code, w.Header().Get("Location"))
	}
	if !strings.Contains(w.Header().Get("Set-Cookie"), "portal_login_state=") {
		t.Fatal("应当设置 state cookie")
	}
	// 没有 state cookie 的回调直接失败，不联系 Steam
	req = httptest.NewRequest("GET", "http://1.2.3.4:8090/auth/steam/callback?state=abc&openid.mode=id_res", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 302 || !strings.HasPrefix(w.Header().Get("Location"), "/#/me?error=") {
		t.Fatalf("callback without state: %d %s", w.Code, w.Header().Get("Location"))
	}
}

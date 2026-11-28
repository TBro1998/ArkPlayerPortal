package auth

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// WriteIdentity 在 dir 里写一个身份密钥，返回私钥
func writeIdentity(t *testing.T, dir string, seedByte byte) ed25519.PrivateKey {
	t.Helper()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = seedByte
	if err := os.WriteFile(filepath.Join(dir, identityFile), []byte(base64.StdEncoding.EncodeToString(seed)), 0o600); err != nil {
		t.Fatal(err)
	}
	return ed25519.NewKeyFromSeed(seed)
}

func TestIdentityAndSession(t *testing.T) {
	dir := t.TempDir()
	id := NewIdentity(dir)
	if _, err := id.Prove("0123456789abcdef"); err == nil {
		t.Fatal("没有身份密钥时应当报错")
	}
	priv := writeIdentity(t, dir, 1)
	proof, err := id.Prove("0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := base64.RawURLEncoding.DecodeString(proof.PublicKey)
	sig, _ := base64.RawURLEncoding.DecodeString(proof.Signature)
	if !ed25519.Verify(pub, []byte(identityContext+"0123456789abcdef"), sig) {
		t.Fatal("身份证明的签名应当能验证")
	}
	sum := sha256.Sum256(priv.Public().(ed25519.PublicKey))
	if proof.Audience != hex.EncodeToString(sum[:]) {
		t.Fatal("audience 应当是公钥的 SHA-256")
	}

	now := time.Unix(1_800_000_000, 0)
	tok, _, err := id.IssueSession(User{SteamID: "76561198000000001", Name: "玩家"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if u, ok := id.ParseSession(tok, now.Add(time.Hour)); !ok || u.SteamID != "76561198000000001" || u.Name != "玩家" {
		t.Fatalf("session: %+v %v", u, ok)
	}
	if _, ok := id.ParseSession(tok, now.Add(sessionTTL+time.Second)); ok {
		t.Error("过期的会话不应当通过")
	}
	if _, ok := id.ParseSession(tok[:len(tok)-2]+"xx", now); ok {
		t.Error("签名不对的会话不应当通过")
	}
	// 身份密钥换了，旧会话作废
	time.Sleep(10 * time.Millisecond)
	writeIdentity(t, dir, 2)
	_ = os.Chtimes(filepath.Join(dir, identityFile), time.Now().Add(time.Minute), time.Now().Add(time.Minute))
	if _, ok := id.ParseSession(tok, now); ok {
		t.Error("身份密钥换了以后旧会话应当作废")
	}
}

// signToken 用 arkfgf.com 的方式签一个令牌
func signToken(priv ed25519.PrivateKey, kid string, claims map[string]any) string {
	h, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": kid})
	p, _ := json.Marshal(claims)
	body := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	return body + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, []byte(body)))
}

// FakeArkfgf 提供公钥的假 arkfgf.com
func fakeArkfgf(t *testing.T, pub ed25519.PublicKey, kid string, hits *int) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kid": kid, "kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "x": base64.RawURLEncoding.EncodeToString(pub)}}})
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestArkfgfVerify(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	seed[3] = 9
	site := ed25519.NewKeyFromSeed(seed)
	hits := 0
	ts := fakeArkfgf(t, site.Public().(ed25519.PublicKey), "k1", &hits)
	keys := &ArkfgfKeys{URL: ts.URL}
	now := time.Unix(1_800_000_000, 0)
	good := map[string]any{"iss": "arkfgf.com", "aud": "aud1", "sub": "76561198000000001", "name": "玩家",
		"avatar": "https://avatars.steamstatic.com/a.jpg", "exp": now.Add(time.Minute).Unix()}
	u, err := keys.Verify(context.Background(), signToken(site, "k1", good), "aud1", now)
	if err != nil || u.SteamID != "76561198000000001" || u.Avatar == "" {
		t.Fatalf("verify: %+v %v", u, err)
	}
	// 公钥有缓存
	if _, err := keys.Verify(context.Background(), signToken(site, "k1", good), "aud1", now); err != nil || hits != 1 {
		t.Fatalf("keys should be cached: %v %d", err, hits)
	}

	other := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	bad := func(mod func(map[string]any)) map[string]any {
		c := map[string]any{}
		for k, v := range good {
			c[k] = v
		}
		mod(c)
		return c
	}
	cases := map[string]string{
		"别的 audience": signToken(site, "k1", bad(func(c map[string]any) { c["aud"] = "aud2" })),
		"过期":          signToken(site, "k1", bad(func(c map[string]any) { c["exp"] = now.Add(-time.Second).Unix() })),
		"别的签发方":       signToken(site, "k1", bad(func(c map[string]any) { c["iss"] = "evil" })),
		"SteamID 无效":  signToken(site, "k1", bad(func(c map[string]any) { c["sub"] = "123" })),
		"签名不对":        signToken(other, "k1", good),
		"格式不对":        "a.b",
	}
	for name, tok := range cases {
		if _, err := keys.Verify(context.Background(), tok, "aud1", now); err == nil {
			t.Errorf("%s 应当校验失败", name)
		}
	}
	// HS256 之类的算法一律拒绝
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","kid":"k1"}`))
	if _, err := keys.Verify(context.Background(), h+".e30.x", "aud1", now); err == nil {
		t.Error("不是 EdDSA 的令牌应当拒绝")
	}
	// 头像只接受 Steam 的
	u, _ = keys.Verify(context.Background(), signToken(site, "k1", bad(func(c map[string]any) { c["avatar"] = "javascript:alert(1)" })), "aud1", now)
	if u.Avatar != "" {
		t.Errorf("不是 Steam 的头像应当丢掉: %q", u.Avatar)
	}
}

func TestSteamVerify(t *testing.T) {
	valid := true
	var got url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		if valid {
			_, _ = w.Write([]byte("ns:http://specs.openid.net/auth/2.0\nis_valid:true\n"))
		} else {
			_, _ = w.Write([]byte("is_valid:false\n"))
		}
	}))
	defer ts.Close()
	s := &Steam{Endpoint: ts.URL}
	returnTo := "http://1.2.3.4:8090/auth/steam/callback?state=abc"
	params := url.Values{
		"openid.mode":        {"id_res"},
		"openid.return_to":   {returnTo},
		"openid.op_endpoint": {steamOpenID},
		"openid.claimed_id":  {"https://steamcommunity.com/openid/id/76561198000000001"},
		"openid.identity":    {"https://steamcommunity.com/openid/id/76561198000000001"},
		"openid.sig":         {"x"},
	}
	id, err := s.Verify(context.Background(), params, returnTo)
	if err != nil || id != "76561198000000001" || got.Get("openid.mode") != "check_authentication" {
		t.Fatalf("verify: %s %v %v", id, err, got)
	}
	if _, err := s.Verify(context.Background(), params, "http://evil/auth/steam/callback?state=abc"); err == nil {
		t.Error("回调地址不对应当失败")
	}
	valid = false
	if _, err := s.Verify(context.Background(), params, returnTo); err == nil {
		t.Error("Steam 不确认时应当失败")
	}
	if !strings.Contains(s.LoginURL(returnTo, "http://1.2.3.4:8090/"), "openid.return_to=") {
		t.Error("login url")
	}
}

func TestPlayers(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "players"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, "players", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("3.json", `{"players":{"76561198000000001":{"points":120},"76561198000000002":{"points":5}}}`)
	write("5.json", `{"players":{"76561198000000002":{"points":7}}}`)
	write("x.json", `{"players":{"76561198000000001":{"points":999}}}`)
	p := NewPlayers(dir)
	got := p.For("76561198000000001")
	if len(got) != 1 || string(got["3"]) != `{"points":120}` {
		t.Fatalf("players: %v", got)
	}
	if got := p.For("76561198000000002"); len(got) != 2 {
		t.Fatalf("players: %v", got)
	}
	_ = os.Remove(filepath.Join(dir, "players", "3.json"))
	if got := p.For("76561198000000001"); len(got) != 0 {
		t.Fatalf("删除后应当没有: %v", got)
	}
}

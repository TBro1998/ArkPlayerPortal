package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/TBro1998/ArkPlayerPortal/internal/auth"
)

// 登录（见 docs/api.md 的「登录」）：
//   - 网页：/auth/steam/login 跳到 Steam，/auth/steam/callback 回来后设置 HttpOnly cookie
//   - 联机助手：GET /api/v1/identity 拿玩家站的身份证明，向 arkfgf.com 换令牌，POST /api/v1/login 换会话（Bearer）
//   - GET /api/v1/me：当前玩家和他自己的数据，没登录时 user 为 null

const (
	sessionCookie = "portal_session"
	stateCookie   = "portal_login_state"
	maxLoginBody  = 8 << 10
)

var noncePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{16,128}$`)

// Login 登录需要的依赖
type Login struct {
	Identity  *auth.Identity
	Players   *auth.Players
	Arkfgf    *auth.ArkfgfKeys // 没有配置时联机助手不能登录
	Steam     *auth.Steam
	PublicURL string // PORTAL_PUBLIC_URL，为空时按请求的 Host 算
	Now       func() time.Time
}

func (l *Login) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

func (s *Server) registerLogin(mux *http.ServeMux) {
	l := s.login
	if l == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/identity", s.handleIdentity)
	mux.HandleFunc("POST /api/v1/login", s.handleTokenLogin)
	mux.HandleFunc("POST /api/v1/logout", s.handleLogout)
	mux.HandleFunc("GET /api/v1/me", s.handleMe)
	mux.HandleFunc("GET /auth/steam/login", s.handleSteamLogin)
	mux.HandleFunc("GET /auth/steam/callback", s.handleSteamCallback)
}

// postAllowed 可以用 POST 的路径
func postAllowed(path string) bool {
	return path == "/api/v1/login" || path == "/api/v1/logout"
}

func (s *Server) handleIdentity(w http.ResponseWriter, r *http.Request) {
	nonce := r.URL.Query().Get("nonce")
	if !noncePattern.MatchString(nonce) {
		writeError(w, http.StatusBadRequest, "invalid_nonce", "nonce 无效")
		return
	}
	proof, err := s.login.Identity.Prove(nonce)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, proof)
}

func (s *Server) handleTokenLogin(w http.ResponseWriter, r *http.Request) {
	if !s.login.Arkfgf.Enabled() {
		writeError(w, http.StatusNotFound, "link_login_disabled", "这个玩家站没有开启联机助手登录")
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, maxLoginBody)).Decode(&in); err != nil || in.Token == "" {
		writeError(w, http.StatusBadRequest, "invalid_input", "缺少令牌")
		return
	}
	audience, err := s.login.Identity.Audience()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", err.Error())
		return
	}
	user, err := s.login.Arkfgf.Verify(r.Context(), in.Token, audience, s.login.now())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", err.Error())
		return
	}
	session, exp, err := s.login.Identity.IssueSession(user, s.login.now())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_ready", err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"session": session, "expires_at": exp.Unix(), "user": user})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, s.cookie(r, sessionCookie, "", -1))
	w.WriteHeader(http.StatusNoContent)
}

// currentUser 会话来自 Authorization: Bearer（联机助手）或 cookie（网页）
func (s *Server) currentUser(r *http.Request) (auth.User, bool) {
	token := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimPrefix(h, "Bearer ")
	} else if c, err := r.Cookie(sessionCookie); err == nil {
		token = c.Value
	}
	if token == "" {
		return auth.User{}, false
	}
	return s.login.Identity.ParseSession(token, s.login.now())
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out := map[string]any{"user": nil, "steam_login": s.login.Steam != nil, "link_login": s.login.Arkfgf.Enabled()}
	if u, ok := s.currentUser(r); ok {
		out["user"] = u
		out["data"] = s.login.Players.For(u.SteamID)
	}
	writeJSON(w, http.StatusOK, out)
}

// baseURL 玩家访问玩家站用的地址：PORTAL_PUBLIC_URL，没有时按请求算（受信任的反向代理给的 X-Forwarded-Proto）
func (s *Server) baseURL(r *http.Request) (string, bool) {
	if s.login.PublicURL != "" {
		return strings.TrimRight(s.login.PublicURL, "/"), true
	}
	host := r.Host
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = net.JoinHostPort(h, p)
	}
	if host == "" || strings.ContainsAny(host, "/\\@ ") {
		return "", false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if s.fromTrustedProxy(r) && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + host, true
}

func (s *Server) secure(r *http.Request) bool {
	base, _ := s.baseURL(r)
	return strings.HasPrefix(base, "https://")
}

func (s *Server) cookie(r *http.Request, name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.secure(r),
	}
}

func (s *Server) handleSteamLogin(w http.ResponseWriter, r *http.Request) {
	base, ok := s.baseURL(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_host", "无法确定玩家站的地址")
		return
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	state := hex.EncodeToString(b[:])
	http.SetCookie(w, s.cookie(r, stateCookie, state, 600))
	returnTo := base + "/auth/steam/callback?state=" + state
	http.Redirect(w, r, s.login.Steam.LoginURL(returnTo, base+"/"), http.StatusFound)
}

func (s *Server) handleSteamCallback(w http.ResponseWriter, r *http.Request) {
	base, ok := s.baseURL(r)
	q := r.URL.Query()
	state := q.Get("state")
	c, err := r.Cookie(stateCookie)
	http.SetCookie(w, s.cookie(r, stateCookie, "", -1))
	if !ok || err != nil || state == "" || c.Value != state {
		s.loginFailed(w, r, "登录已过期，请重新登录")
		return
	}
	steamID, err := s.login.Steam.Verify(r.Context(), q, base+"/auth/steam/callback?state="+url.QueryEscape(state))
	if err != nil {
		s.loginFailed(w, r, err.Error())
		return
	}
	name, avatar := s.login.Steam.Profile(r.Context(), steamID)
	session, _, err := s.login.Identity.IssueSession(auth.User{SteamID: steamID, Name: name, Avatar: avatar}, s.login.now())
	if err != nil {
		s.loginFailed(w, r, err.Error())
		return
	}
	http.SetCookie(w, s.cookie(r, sessionCookie, session, int((30*24*time.Hour)/time.Second)))
	http.Redirect(w, r, "/#/me", http.StatusFound)
}

// loginFailed 回到网页并带上原因（网页在 #/me 显示）
func (s *Server) loginFailed(w http.ResponseWriter, r *http.Request, msg string) {
	http.Redirect(w, r, "/#/me?error="+url.QueryEscape(msg), http.StatusFound)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

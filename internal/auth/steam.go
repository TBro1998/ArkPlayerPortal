package auth

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// 网页的 Steam OpenID 登录：跳到 Steam 登录，回来后向 Steam 核验（check_authentication），拿到 SteamID64。
// 昵称和头像取 Steam 公开资料（不需要 API Key），取不到时只有 SteamID

const steamOpenID = "https://steamcommunity.com/openid/login"

var claimedIDPattern = regexp.MustCompile(`^https://steamcommunity\.com/openid/id/([0-9]{17})$`)

// Steam 核验 Steam 的登录结果
type Steam struct {
	HTTP *http.Client
	// Endpoint 测试时换成本地地址
	Endpoint string
	// ProfileURL 公开资料地址的前缀（测试时换），后面接 SteamID
	ProfileURL string
}

func (s *Steam) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *Steam) endpoint() string {
	if s.Endpoint != "" {
		return s.Endpoint
	}
	return steamOpenID
}

// LoginURL Steam 登录页地址，登录后回到 returnTo
func (s *Steam) LoginURL(returnTo, realm string) string {
	q := url.Values{}
	q.Set("openid.ns", "http://specs.openid.net/auth/2.0")
	q.Set("openid.mode", "checkid_setup")
	q.Set("openid.return_to", returnTo)
	q.Set("openid.realm", realm)
	q.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
	q.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")
	return steamOpenID + "?" + q.Encode()
}

// Verify 核验回调参数：return_to 必须是 expectedReturnTo，再请 Steam 确认签名，返回 SteamID64
func (s *Steam) Verify(ctx context.Context, params url.Values, expectedReturnTo string) (string, error) {
	if params.Get("openid.mode") != "id_res" {
		return "", errors.New("Steam 登录没有完成")
	}
	if params.Get("openid.return_to") != expectedReturnTo {
		return "", errors.New("Steam 登录的回调地址不对")
	}
	if params.Get("openid.op_endpoint") != steamOpenID {
		return "", errors.New("不是 Steam 的登录结果")
	}
	m := claimedIDPattern.FindStringSubmatch(params.Get("openid.claimed_id"))
	if m == nil || params.Get("openid.identity") != params.Get("openid.claimed_id") {
		return "", errors.New("Steam 登录结果里的 SteamID 无效")
	}
	check := url.Values{}
	for k, v := range params {
		if strings.HasPrefix(k, "openid.") && len(v) > 0 {
			check.Set(k, v[0])
		}
	}
	check.Set("openid.mode", "check_authentication")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint(), strings.NewReader(check.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client().Do(req)
	if err != nil {
		return "", errors.New("连不上 Steam，请稍后再试")
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	if !strings.Contains(string(body), "is_valid:true") {
		return "", errors.New("Steam 没有确认这次登录，请重新登录")
	}
	return m[1], nil
}

// Profile 取 Steam 公开资料里的昵称和头像，取不到时返回空
func (s *Steam) Profile(ctx context.Context, steamID string) (name, avatar string) {
	base := s.ProfileURL
	if base == "" {
		base = "https://steamcommunity.com/profiles/"
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+steamID+"/?xml=1", nil)
	if err != nil {
		return "", ""
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	var p struct {
		Name   string `xml:"steamID"`
		Avatar string `xml:"avatarMedium"`
	}
	if xml.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&p) != nil {
		return "", ""
	}
	return limitRunes(p.Name, 64), safeAvatar(strings.TrimSpace(p.Avatar))
}

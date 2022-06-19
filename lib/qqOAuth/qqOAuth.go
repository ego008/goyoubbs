package qqOAuth

import (
	"errors"
	"github.com/ego008/goutils/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

var Logging bool
var openIDRegex = regexp.MustCompile("\\((.*)\\)")

const (
	AuthURL        = "https://graph.qq.com/oauth2.0/authorize"
	AccessTokenURL = "https://graph.qq.com/oauth2.0/token"
	OpenIDURL      = "https://graph.qq.com/oauth2.0/me"
	UserInfoURL    = "https://graph.qq.com/user/get_user_info"
)

type OAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type OAuthToken struct {
	AccessToken  string
	ExpiresIn    string
	RefreshToken string
	Code         string
	Msg          string
}

type UserInfo struct {
	Ret      int    `json:"ret"`
	Message  string `json:"msg"`
	Gender   string `json:"gender"`
	Nickname string `json:"nickname"`
	City     string `json:"city"`
	Province string `json:"province"`
	Avatar   string `json:"figureurl_qq_2"`
}

type OpenID struct {
	ClientID string `json:"client_id"`
	OpenID   string `json:"openid"`
}

func NewQQOAuth(clientID, clientSecret, redirectURL string) (*OAuth, error) {
	if len(clientID) == 0 {
		return nil, errors.New("clientID cannot be empty")
	}
	if len(clientSecret) == 0 {
		return nil, errors.New("clientSecret cannot be empty")
	}
	if len(redirectURL) == 0 {
		return nil, errors.New("redirectURL cannot be empty")
	}

	oauth := &OAuth{}
	oauth.ClientID = clientID
	oauth.ClientSecret = clientSecret
	oauth.RedirectURL = redirectURL
	return oauth, nil
}

func (oauth *OAuth) GetAuthorizationURL(state string) (string, error) {
	if len(state) == 0 {
		return "", errors.New("state cannot be empty")
	}
	qs := url.Values{
		"client_id":     {oauth.ClientID},
		"redirect_uri":  {oauth.RedirectURL},
		"state":         {state},
		"response_type": {"code"}}

	urlStr := AuthURL + "?" + qs.Encode()
	return urlStr, nil
}

func (oauth *OAuth) GetAccessToken(code string) (*OAuthToken, error) {
	if len(code) == 0 {
		return nil, errors.New("code cannot be empty")
	}

	qs := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {oauth.ClientID},
		"client_secret": {oauth.ClientSecret},
		"code":          {code},
		"redirect_uri":  {oauth.RedirectURL}}
	reqURL := AccessTokenURL + "?" + qs.Encode()
	if Logging {
		logReq("GET: " + reqURL)
	}

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)

	if Logging {
		logResp(bodyStr)
	}

	bodyQs, err := url.ParseQuery(bodyStr)
	if err != nil {
		return nil, err
	}

	token := &OAuthToken{}
	token.AccessToken = bodyQs.Get("access_token")
	token.ExpiresIn = bodyQs.Get("expires_in")
	token.RefreshToken = bodyQs.Get("refresh_token")
	token.Code = bodyQs.Get("code")
	token.Msg = bodyQs.Get("msg")
	return token, err
}

func (oauth *OAuth) GetOpenID(accessToken string) (*OpenID, error) {
	if len(accessToken) == 0 {
		return nil, errors.New("accessToken cannot be empty")
	}
	qs := url.Values{
		"access_token": {accessToken}}
	reqURL := OpenIDURL + "?" + qs.Encode()
	if Logging {
		logReq("GET: " + reqURL)
	}

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)

	if Logging {
		logResp(bodyStr)
	}

	regexRes := openIDRegex.FindStringSubmatch(bodyStr)
	if regexRes == nil || len(regexRes) < 2 {
		return nil, errors.New("invalid content ")
	}

	bodyJSON := regexRes[1]
	openID := &OpenID{}
	err = json.Unmarshal([]byte(bodyJSON), openID)
	if err != nil {
		return nil, err
	}
	return openID, err
}

func (oauth *OAuth) GetUserInfo(accessToken string, openID string) (*UserInfo, error) {
	if len(accessToken) == 0 {
		return nil, errors.New("accessToken cannot be nil")
	}
	if len(openID) == 0 {
		return nil, errors.New("openID cannot be nil")
	}
	qs := url.Values{
		"oauth_consumer_key": {oauth.ClientID},
		"access_token":       {accessToken},
		"openid":             {openID}}
	urlStr := UserInfoURL + "?" + qs.Encode()
	if Logging {
		logReq("GET: " + urlStr)
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(bodyBytes)
	if Logging {
		logResp(bodyStr)
	}

	ret := &UserInfo{}
	err = json.Unmarshal(bodyBytes, ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func logReq(content string) {
	log.Print("Request: " + content)
}
func logResp(content string) {
	log.Print("Response: " + content)
}

package weiboOAuth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var Logging bool

const (
	AuthURL        = "https://api.weibo.com/oauth2/authorize"
	AccessTokenURL = "https://api.weibo.com/oauth2/access_token"
	UserInfoURL    = "https://api.weibo.com/2/users/show.json"
)

type OAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type OAuthToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	RemindIn    string `json:"remind_in"`
	UIDString   string `json:"uid"`

	Error        string `json:"error"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_description"`
}

type UserInfo struct {
	UID         int64  `json:"id"`
	Name        string `json:"name"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Avatar      string `json:"profile_image_url"`
	URL         string `json:"url"`
	Gender      string `json:"gender"`
}

func NewWeiboOAuth(clientID, clientSecret, redirectURL string) (*OAuth, error) {
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
	qs := url.Values{
		"client_id":    {oauth.ClientID},
		"redirect_uri": {oauth.RedirectURL},
		"state":        {state}}
	urlStr := AuthURL + "?" + qs.Encode()
	return urlStr, nil
}

func (oauth *OAuth) GetAccessToken(code string) (*OAuthToken, error) {
	if len(code) == 0 {
		return nil, errors.New("code cannot be empty")
	}
	if Logging {
		logReq("POST: " + AccessTokenURL)
	}
	resp, err := http.PostForm(AccessTokenURL,
		url.Values{"client_id": {oauth.ClientID},
			"client_secret": {oauth.ClientSecret},
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {oauth.RedirectURL}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if Logging {
		logResp(string(body))
	}

	token := &OAuthToken{}
	err = json.Unmarshal(body, token)
	if err != nil {
		return nil, err
	}
	if token.ErrorCode != 0 {
		return nil, errors.New(token.ErrorMessage)
	}
	return token, err
}

func (oauth *OAuth) GetUserInfo(accessToken, uid string) (*UserInfo, error) {
	if len(accessToken) == 0 {
		return nil, errors.New("accessToken cannot be empty")
	}
	qs := url.Values{"access_token": {accessToken},
		"uid": {uid}}
	urlStr := UserInfoURL + "?" + qs.Encode()
	if Logging {
		logReq("GET: " + urlStr)
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if Logging {
		logResp(string(body))
	}

	ret := &UserInfo{}
	err = json.Unmarshal(body, ret)
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

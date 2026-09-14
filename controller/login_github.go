package controller

import (
	"context"
	"goyoubbs/model"
	"goyoubbs/util"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/rs/xid"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

func (h *BaseHandler) GithubOauthHandler(c *gin.Context) {
	scf := h.App.Cf.Site

	// https://github.com/settings/developers

	oauthConf := &oauth2.Config{
		ClientID:     scf.GithubClientID,
		ClientSecret: scf.GithubClientSecret,
		//Scopes:       []string{"user:email", "repo"},
		RedirectURL: scf.MainDomain + "/oauth/github/callback",
		Endpoint:    githuboauth.Endpoint,
	}

	now := time.Now().UTC().Unix()
	githubUrlState := strconv.FormatInt(now, 10)[6:]

	_ = h.SetCookie(c, "githubUrlState", githubUrlState, 1)

	gotoUrl := oauthConf.AuthCodeURL(githubUrlState, oauth2.AccessTypeOnline)
	c.Redirect(302, gotoUrl)
}

func (h *BaseHandler) GithubOauthCallback(c *gin.Context) {
	githubUrlState := h.GetCookie(c, "githubUrlState")
	if len(githubUrlState) == 0 {
		c.String(200, `githubUrlState cookie missed`)
		return
	}

	code := c.Query("code")
	if code == "" {
		c.String(200, "Invalid code")
		return
	}

	state := c.Query("state")
	if state != githubUrlState {
		c.String(200, "Invalid state")
		return
	}

	scf := h.App.Cf.Site
	oauthConf := &oauth2.Config{
		ClientID:     scf.GithubClientID,
		ClientSecret: scf.GithubClientSecret,
		//Scopes:       []string{"user:email", "repo"},
		RedirectURL: scf.MainDomain + "/oauth/github/callback",
		Endpoint:    githuboauth.Endpoint,
	}

	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	oauthClient := oauthConf.Client(context.Background(), token)
	client := github.NewClient(oauthClient)
	githubUser, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		c.String(200, err.Error())
		return
	}

	githubIdStr := strconv.FormatInt(*githubUser.ID, 10)

	timeStamp := uint64(time.Now().UTC().Unix())

	db := h.App.Db
	jb, _ := json.Marshal(githubUser)
	_ = db.Hset("github_user_info", []byte(githubIdStr), jb)

	next := h.GetCookie(c, "next")

	authorKey := "gh:" + githubIdStr
	rs := db.Hget("oauth2user", []byte(authorKey))
	if rs.OK() {
		// login
		log.Print("authorKey ok", rs.String())
		obj := model.AuthInfo{}
		_ = json.Unmarshal(rs.Data[0], &obj)
		if obj.Uid > 0 {
			// 已绑定用户名则直接登录
			uObj, _ := model.UserGetById(db, obj.Uid)
			if uObj.ID == 0 {
				c.String(200, "uid not found")
				return
			}
			sessionId := xid.New().String()
			uObj.LastLoginTime = timeStamp
			uObj.Session = sessionId
			jb, _ := json.Marshal(uObj)
			_ = db.Hset(model.UserTbName, sdb.I2b(uObj.ID), jb)
			_ = h.SetCookie(c, "SessionID", strconv.FormatUint(uObj.ID, 10)+":"+sessionId, 365)

			if len(next) > 0 {
				h.DelCookie(c, "next")
				c.Redirect(302, scf.MainDomain+next)
				return
			}
			c.Redirect(302, scf.MainDomain+"/")
			return
		}
	} else {
		log.Println("oauth2user", authorKey, "not exist - go to reg")
	}

	jb, _ = json.Marshal(model.AuthInfo{Openid: githubIdStr})
	_ = db.Hset("oauth2user", sdb.S2b(authorKey), jb)

	// 绑定用户名，跳到注册页面，填写默认登录名

	if scf.CloseReg {
		c.String(200, `stop to new register`)
		return
	}

	// 保存 openid ，以便在 注册 时取出可用登录名及注册成功后自动获取头像
	_ = h.SetCookie(c, "openid", authorKey, 1)

	// 获取用户名和头像
	name := util.RemoveCharacter(*githubUser.Login)
	name = strings.TrimSpace(strings.Replace(name, " ", "", -1))
	if len(name) > 0 {
		nameLow := strings.ToLower(name)
		if db.Hget("user_name2uid", []byte(nameLow)).OK() {
			name = ""
		}
	}

	uUrl := *githubUser.HTMLURL
	if len(*githubUser.Blog) > 0 {
		uUrl = *githubUser.Blog
	}
	jb, _ = json.Marshal(model.AuthProfileInfo{
		LoginBy: "github",
		OpenId:  githubIdStr,
		Name:    name,
		Avatar:  *githubUser.AvatarURL,
		Agent:   c.Request.UserAgent(),
		About:   "",
		Url:     uUrl,
	})
	_ = db.Hset("oauth_tmp_info", sdb.S2b(authorKey), jb)

	c.Redirect(302, scf.MainDomain+"/register")
}

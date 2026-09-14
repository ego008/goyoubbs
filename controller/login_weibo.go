package controller

import (
	"goyoubbs/lib/weiboOAuth"
	"goyoubbs/model"
	"goyoubbs/util"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
)

func (h *BaseHandler) WeiboOauthHandler(c *gin.Context) {
	scf := h.App.Cf.Site
	weibo, err := weiboOAuth.NewWeiboOAuth(scf.WeiboClientID, scf.WeiboClientSecret, scf.MainDomain+"/oauth/wb/callback")
	if err != nil {
		c.String(200, err.Error())
		return
	}
	// weiboOAuth.Logging = true

	now := time.Now().UTC().Unix()
	WeiboUrlState := strconv.FormatInt(now, 10)[6:]

	urlStr, err := weibo.GetAuthorizationURL(WeiboUrlState)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	_ = h.SetCookie(c, "WeiboUrlState", WeiboUrlState, 1)
	c.Redirect(302, urlStr)
}

func (h *BaseHandler) WeiboOauthCallback(c *gin.Context) {
	WeiboUrlState := h.GetCookie(c, "WeiboUrlState")
	if len(WeiboUrlState) == 0 {
		c.String(200, `WeiboUrlState cookie missed`)
		return
	}

	scf := h.App.Cf.Site
	weibo, err := weiboOAuth.NewWeiboOAuth(scf.WeiboClientID, scf.WeiboClientSecret, scf.MainDomain+"/oauth/wb/callback")
	if err != nil {
		c.String(200, err.Error())
		return
	}
	// weiboOAuth.Logging = true

	code := c.Query("code")
	if code == "" {
		c.String(200, "Invalid code")
		return
	}

	state := c.Query("state")
	if state != WeiboUrlState {
		c.String(200, "Invalid state")
		return
	}

	token, err := weibo.GetAccessToken(code)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	wbUserID := token.UIDString

	timeStamp := uint64(time.Now().UTC().Unix())
	next := h.GetCookie(c, "next")

	db := h.App.Db
	authorKey := "wb:" + wbUserID
	rs := db.Hget("oauth2user", []byte(authorKey))
	if rs.OK() {
		// login
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
	}

	jb, _ := json.Marshal(model.AuthInfo{Openid: wbUserID})
	_ = db.Hset("oauth2user", sdb.S2b(authorKey), jb)
	// 绑定用户名，跳到注册页面，填写默认登录名

	if scf.CloseReg {
		c.String(200, `stop to new register`)
		return
	}

	// 保存 openid ，以便在 注册 时取出可用登录名及注册成功后自动获取头像
	_ = h.SetCookie(c, "openid", authorKey, 1)

	// 获取用户名和头像
	profile, err := weibo.GetUserInfo(token.AccessToken, wbUserID)
	if err == nil {
		name := util.RemoveCharacter(profile.Name)
		name = strings.TrimSpace(strings.Replace(name, " ", "", -1))
		if len(name) > 0 {
			nameLow := strings.ToLower(name)
			if db.Hget("user_name2uid", []byte(nameLow)).OK() {
				name = ""
			}
		}

		jb, _ := json.Marshal(model.AuthProfileInfo{
			LoginBy: "weibo",
			OpenId:  wbUserID,
			Name:    name,
			Avatar:  profile.Avatar,
			Agent:   c.Request.UserAgent(),
			About:   profile.Description,
			Url:     profile.URL,
		})
		_ = db.Hset("oauth_tmp_info", sdb.S2b(authorKey), jb)
	}

	c.Redirect(302, scf.MainDomain+"/register")
}

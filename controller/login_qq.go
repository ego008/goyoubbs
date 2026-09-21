package controller

import (
	"goyoubbs/lib/qqOAuth"
	"goyoubbs/model"
	"goyoubbs/util"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) QQOauthHandler(c *gin.Context) {
	scf := h.App.Cf.Site
	qq, err := qqOAuth.NewQQOAuth(scf.QQClientID, scf.QQClientSecret, scf.MainDomain+"/oauth/qq/callback")
	if err != nil {
		c.String(200, err.Error())
		return
	}
	// qqOAuth.Logging = true

	now := time.Now().UTC().Unix()
	qqUrlState := strconv.FormatInt(now, 10)[6:]

	urlStr, err := qq.GetAuthorizationURL(qqUrlState)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	_ = h.SetCookie(c, "QQUrlState", qqUrlState, 1)
	c.Redirect(302, urlStr)
}

func (h *BaseHandler) QQOauthCallback(c *gin.Context) {
	qqUrlState := h.GetCookie(c, "QQUrlState")
	if len(qqUrlState) == 0 {
		c.String(200, `qqUrlState cookie missed`)
		return
	}

	scf := h.App.Cf.Site
	qq, err := qqOAuth.NewQQOAuth(scf.QQClientID, scf.QQClientSecret, scf.MainDomain+"/oauth/qq/callback")
	if err != nil {
		c.String(200, err.Error())
		return
	}
	// qqOAuth.Logging = true

	code := c.Query("code")
	if code == "" {
		c.String(200, "Invalid code")
		return
	}

	state := c.Query("state")
	if state != qqUrlState {
		c.String(200, "Invalid state")
		return
	}

	token, err := qq.GetAccessToken(code)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	openid, err := qq.GetOpenID(token.AccessToken)
	if err != nil {
		c.String(200, err.Error())
		return
	}

	timeStamp := uint64(time.Now().UTC().Unix())
	next := h.GetCookie(c, "next")

	db := h.App.Db

	_ = db.Update(func(tx *bbolt.Tx) error {

		authorKey := "qq:" + openid.OpenID
		val := db.HGet(tx, "oauth2user", []byte(authorKey))
		if len(val) > 0 {
			// login
			obj := model.AuthInfo{}
			_ = json.Unmarshal(val, &obj)

			if obj.Uid > 0 {
				// 已绑定用户名则直接登录
				uObj, _ := model.UserGetById(db, tx, obj.Uid)
				if uObj.ID == 0 {
					c.String(200, "uid not found")
					return nil
				}
				sessionId := xid.New().String()
				uObj.LastLoginTime = timeStamp
				uObj.Session = sessionId
				jb, _ := json.Marshal(uObj)
				_ = db.HSet(tx, model.UserTbName, mdb.I2b(uObj.ID), jb)
				_ = h.SetCookie(c, "SessionID", strconv.FormatUint(uObj.ID, 10)+":"+sessionId, 365)

				if len(next) > 0 {
					h.DelCookie(c, "next")
					c.Redirect(302, scf.MainDomain+next)
					return nil
				}
				c.Redirect(302, scf.MainDomain+"/")
				return nil
			}
		}

		jb, _ := json.Marshal(model.AuthInfo{Openid: openid.OpenID})
		_ = db.HSet(tx, "oauth2user", mdb.S2b(authorKey), jb)

		// 绑定用户名，跳到注册页面，填写默认登录名

		if scf.CloseReg {
			c.String(200, `stop to new register`)
			return nil
		}

		// 保存 openid ，以便在 注册 时取出可用登录名及注册成功后自动获取头像
		_ = h.SetCookie(c, "openid", authorKey, 1)

		// 获取用户名和头像
		profile, err := qq.GetUserInfo(token.AccessToken, openid.OpenID)
		if err == nil {
			if profile.Ret == 0 {
				name := util.RemoveCharacter(profile.Nickname)
				name = strings.TrimSpace(strings.Replace(name, " ", "", -1))
				if len(name) > 0 {
					nameLow := strings.ToLower(name)
					if db.HKeyExist(tx, "user_name2uid", []byte(nameLow)) {
						name = ""
					}
				}

				jb, _ := json.Marshal(model.AuthProfileInfo{
					LoginBy: "qq",
					OpenId:  openid.OpenID,
					Name:    name,
					Avatar:  profile.Avatar,
					Agent:   c.Request.UserAgent(),
				})
				_ = db.HSet(tx, "oauth_tmp_info", mdb.S2b(authorKey), jb)
			}
		}

		return nil
	})

	c.Redirect(302, scf.MainDomain+"/register")
}

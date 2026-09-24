package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/captcha"
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	"go.etcd.io/bbolt"
)

// UserLoginPage 把这个页面放到这里为了解决验证码不显示的bug
func (h *BaseHandler) UserLoginPage(c *gin.Context) {

	scf := h.App.Cf.Site

	act := strings.TrimLeft(c.Request.URL.Path, "/")
	title := "登录"
	if act == "register" {
		title = "注册"
	}

	evn := &ybs.UserLogin{}
	evn.SiteCf = scf
	evn.Title = title
	evn.PageName = "user_login_register"

	//evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)

	evn.Act = act
	evn.CaptchaId = captcha.New()

	// 继承第三方登录信息
	openid := h.GetCookie(c, "openid")
	if openid != "" {
		_ = h.App.Db.View(func(tx *bbolt.Tx) error {
			_ = h.App.Db.HGetFunc(tx, "oauth_tmp_info", []byte(openid[3:]), func(val []byte) error {
				obj := model.AuthProfileInfo{}
				_ = json.Unmarshal(val, &obj)
				evn.DefaultName = obj.Name
				return nil
			})
			return nil
		})
	}

	if openid == "" { // 避免第三方登录循环
		if !scf.AllowNameReg && act == "register" {
			// 只允许第三方账户登录或已注册用户登录
			c.Redirect(302, scf.MainDomain+"/login")
			return
		}
		if scf.QQClientID != "" || scf.WeiboClientID != "" || scf.GithubClientID != "" {
			evn.HasOtherAuth = true
		}
	}

	token := h.GetCookie(c, "token")
	if len(token) == 0 {
		token = xid.New().String()
		_ = h.SetCookie(c, "token", token, 1)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) UserLoginPost(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	token := h.GetCookie(c, "token")
	if len(token) == 0 {
		c.String(200, `{"Code":400,"Msg":"token cookie missed"}`)
		return
	}

	act := strings.TrimLeft(c.Request.URL.Path, "/")

	type recForm struct {
		Name            string
		Password        string
		CaptchaId       string
		CaptchaSolution string
	}

	type response struct {
		model.NormalRsp
	}

	var rec recForm
	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	if len(rec.Name) == 0 || len(rec.Password) == 0 {
		c.String(200, `{"Code":400,"Msg":"name or pw is empty"}`)
		return
	}
	nameLow := strings.ToLower(strings.TrimSpace(rec.Name))
	if !util.IsNickname(nameLow) {
		c.String(200, `{"Code":400,"Msg":"name fmt err"}`)
		return
	}

	// 验证码校验
	if !captcha.VerifyString(rec.CaptchaId, rec.CaptchaSolution) {
		// 验证码校验后会注销，以下若有出错返回都要重新生成CaptchaId，错误代码405
		c.String(200, `{"Code":405,"Msg":"验证码错误","NewCaptchaId":"`+captcha.New()+`"}`)
		return
	}

	rec.Password = strings.ToLower(rec.Password) // 密码均为小写

	db := h.App.Db
	timeStamp := uint64(util.GetCNTM(model.TimeOffSet))

	var autoAvatar bool // 注册时自动生成头像
	var loggedUid uint64

	var showStr string
	_ = db.Update(func(tx *bbolt.Tx) error {

		if act == "login" {

			tbn := "user_login_pw_err"
			clientIp := ReadUserIP(c)
			if len(clientIp) == 0 {
				showStr = `{"Code":405,"Msg":"clientIp is empty","NewCaptchaId":"` + captcha.New() + `"}`
				return nil
			}

			var hasDelKey bool
			offSetSeconds := int64(120)
			nowTm := time.Now().Unix()
			if tm := db.HGetInt(tx, tbn, []byte(clientIp)); tm > 0 {
				if nowTm-int64(tm) < offSetSeconds {
					showStr = `{"Code":405,"Msg":"sleep 2 min","NewCaptchaId":"` + captcha.New() + `"}`
					return nil
				}
				hasDelKey = true
			}

			uobj, err := model.UserGetByName(db, tx, nameLow)
			if err != nil {
				showStr = `{"Code":405,"Msg":"用户不存在","NewCaptchaId":"` + captcha.New() + `"}`
				return nil
			}
			if uobj.Password != rec.Password {
				_ = db.HSet(tx, tbn, []byte(clientIp), mdb.I2b(uint64(nowTm)))
				showStr = `{"Code":405,"Msg":"name and pw not match","NewCaptchaId":"` + captcha.New() + `"}`
				return nil
			}
			if hasDelKey {
				_ = db.HDel(tx, tbn, []byte(clientIp))
			}

			sessionid := xid.New().String()
			uobj.LastLoginTime = timeStamp
			uobj.Session = sessionid
			jb, _ := json.Marshal(uobj)
			_ = db.HSet(tx, model.UserTbName, mdb.I2b(uobj.ID), jb)
			_ = h.SetCookie(c, "SessionID", strconv.FormatUint(uobj.ID, 10)+":"+sessionid, 365)

			loggedUid = uobj.ID
		} else {
			// register
			siteCf := h.App.Cf.Site

			if db.HGetInt(tx, model.CountTb, mdb.S2b(model.UserTbName)) > 0 {
				if siteCf.CloseReg {
					showStr = `{"Code":405,"Msg":"stop to new register","NewCaptchaId":"` + captcha.New() + `"}`
					return nil
				}
				if db.HKeyExist(tx, "user_name2uid", []byte(nameLow)) {
					showStr = `{"Code":405,"Msg":"name is exist","NewCaptchaId":"` + captcha.New() + `"}`
					return nil
				}
			}

			userId, _ := db.HIncr(tx, model.CountTb, mdb.S2b(model.UserTbName), 1)
			flag := 5
			if siteCf.RegReview {
				flag = 1
			}

			if userId == 1 {
				flag = 99
			}

			uobj := model.User{
				ID:            userId,
				Name:          rec.Name,
				Password:      rec.Password,
				Flag:          flag,
				RegTime:       timeStamp,
				LastLoginTime: timeStamp,
				Session:       xid.New().String(),
			}

			jb, _ := json.Marshal(uobj)
			_ = db.HSet(tx, model.UserTbName, mdb.I2b(uobj.ID), jb)
			_ = db.HSet(tx, "user_name2uid", []byte(nameLow), mdb.I2b(userId))
			_ = db.HSet(tx, "user_flag:"+strconv.Itoa(flag), mdb.I2b(uobj.ID), nil)

			//生成头像
			_ = util.GenAvatar(db, tx, uobj.ID, uobj.Name)
			autoAvatar = true

			_ = h.SetCookie(c, "SessionID", strconv.FormatUint(uobj.ID, 10)+":"+uobj.Session, 365)

			loggedUid = uobj.ID
		}

		// 绑定第三方登录信息
		openid := h.GetCookie(c, "openid") // authorKey
		if openid != "" {
			obj := model.AuthInfo{}
			var ok bool
			_ = db.HGetFunc(tx, "oauth2user", []byte(openid), func(val []byte) error {
				ok = true
				_ = json.Unmarshal(val, &obj)
				return nil
			})
			if ok {
				obj.Name = rec.Name
				obj.Uid = loggedUid
				jb, _ := json.Marshal(obj)
				_ = h.App.Db.HSet(tx, "oauth2user", []byte(openid), jb)
			}
			// 自动取头像
			if autoAvatar {
				obj := model.AuthProfileInfo{}
				_ = db.HGetFunc(tx, "oauth_tmp_info", []byte(openid), func(val []byte) error {
					_ = json.Unmarshal(val, &obj)
					return nil
				})
				if obj.Avatar != "" {
					jb, _ := json.Marshal(model.AvatarTask{
						Uid:      loggedUid,
						Name:     rec.Name,
						Avatar:   obj.Avatar,
						SavePath: "static/avatar/" + strconv.FormatUint(loggedUid, 10) + ".jpg",
						Agent:    obj.Agent,
					})
					_ = db.HSet(tx, "task_to_get_avatar", mdb.I2b(loggedUid), jb)
				}
			}
		}

		return nil
	})

	if len(showStr) > 0 {
		c.String(200, showStr)
		return
	}

	h.DelCookie(c, "openid")
	h.DelCookie(c, "token")

	rsp := response{}
	rsp.Code = 200
	_ = json.NewEncoder(c.Writer).Encode(rsp)
}

func (h *BaseHandler) UserLogout(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.ID > 0 {
		cks := []string{"SessionID", "token"}
		for _, k := range cks {
			h.DelCookie(c, k)
		}
	}
	c.Redirect(302, "/")
}

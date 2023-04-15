package controller

import (
	"github.com/ego008/captcha"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/rs/xid"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"strconv"
	"strings"
	"time"
)

// UserLoginPage 把这个页面放到这里为了解决验证码不显示的bug
func (h *BaseHandler) UserLoginPage(ctx *fasthttp.RequestCtx) {

	scf := h.App.Cf.Site

	act := strings.TrimLeft(sdb.B2s(ctx.RequestURI()), "/")
	title := "登录"
	if act == "register" {
		title = "注册"
	}

	evn := &ybs.UserLogin{}
	evn.SiteCf = scf
	evn.Title = title
	evn.PageName = "user_login_register"

	//evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	evn.Act = act
	evn.CaptchaId = captcha.New()

	// 继承第三方登录信息
	openid := h.GetCookie(ctx, "openid")
	if openid != "" {
		if rs := h.App.Db.Hget("oauth_tmp_info", []byte(openid[3:])); rs.OK() {
			obj := model.AuthProfileInfo{}
			_ = json.Unmarshal(rs.Bytes(), &obj)
			evn.DefaultName = obj.Name
		}
	}

	if openid == "" { // 避免第三方登录循环
		if !scf.AllowNameReg && act == "register" {
			// 只允许第三方账户登录或已注册用户登录
			ctx.Redirect(scf.MainDomain+"/login", 302)
			return
		}
		if scf.QQClientID != "" || scf.WeiboClientID != "" || scf.GithubClientID != "" {
			evn.HasOtherAuth = true
		}
	}

	token := h.GetCookie(ctx, "token")
	if len(token) == 0 {
		token = xid.New().String()
		_ = h.SetCookie(ctx, "token", token, 1)
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ybs.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) UserLoginPost(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=UTF-8")

	token := h.GetCookie(ctx, "token")
	if len(token) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"token cookie missed"}`)
		return
	}

	act := strings.TrimLeft(sdb.B2s(ctx.RequestURI()), "/")

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
	err := util.Bind(ctx, util.JSON, &rec)
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	if len(rec.Name) == 0 || len(rec.Password) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"name or pw is empty"}`)
		return
	}
	nameLow := strings.ToLower(strings.TrimSpace(rec.Name))
	if !util.IsNickname(nameLow) {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"name fmt err"}`)
		return
	}

	// 验证码校验
	if !captcha.VerifyString(rec.CaptchaId, rec.CaptchaSolution) {
		// 验证码校验后会注销，以下若有出错返回都要重新生成CaptchaId，错误代码405
		_, _ = ctx.WriteString(`{"Code":405,"Msg":"验证码错误","NewCaptchaId":"` + captcha.New() + `"}`)
		return
	}

	rec.Password = strings.ToLower(rec.Password) // 密码均为小写

	db := h.App.Db
	timeStamp := uint64(util.GetCNTM(model.TimeOffSet))

	var autoAvatar bool // 注册时自动生成头像
	var loggedUid uint64
	if act == "login" {

		tbn := "user_login_pw_err"
		clientIp := ReadUserIP(ctx)
		if len(clientIp) == 0 {
			_, _ = ctx.WriteString(`{"Code":405,"Msg":"clientIp is empty","NewCaptchaId":"` + captcha.New() + `"}`)
			return
		}

		var hasDelKey bool
		offSetSeconds := int64(120)
		nowTm := time.Now().Unix()
		if tm := db.HgetInt(tbn, []byte(clientIp)); tm > 0 {
			if nowTm-int64(tm) < offSetSeconds {
				_, _ = ctx.WriteString(`{"Code":405,"Msg":"sleep 2 min","NewCaptchaId":"` + captcha.New() + `"}`)
				return
			}
			hasDelKey = true
		}

		uobj, err := model.UserGetByName(db, nameLow)
		if err != nil {
			//_, _ = ctx.WriteString(`{"Code":405,"Msg":"json Decode err:` + err.Error() + `","NewCaptchaId":"` + captcha.New() + `"}`)
			_, _ = ctx.WriteString(`{"Code":405,"Msg":"用户不存在","NewCaptchaId":"` + captcha.New() + `"}`)
			return
		}
		if uobj.Password != rec.Password {
			_ = db.Hset(tbn, []byte(clientIp), sdb.I2b(uint64(nowTm)))
			_, _ = ctx.WriteString(`{"Code":405,"Msg":"name and pw not match","NewCaptchaId":"` + captcha.New() + `"}`)
			return
		}
		if hasDelKey {
			_ = db.Hdel(tbn, []byte(clientIp))
		}

		sessionid := xid.New().String()
		uobj.LastLoginTime = timeStamp
		uobj.Session = sessionid
		jb, _ := json.Marshal(uobj)
		_ = db.Hset(model.UserTbName, sdb.I2b(uobj.ID), jb)
		_ = h.SetCookie(ctx, "SessionID", strconv.FormatUint(uobj.ID, 10)+":"+sessionid, 365)

		loggedUid = uobj.ID
	} else {
		// register
		siteCf := h.App.Cf.Site

		if db.HgetInt(model.CountTb, sdb.S2b(model.UserTbName)) > 0 {
			if siteCf.CloseReg {
				_, _ = ctx.WriteString(`{"Code":405,"Msg":"stop to new register","NewCaptchaId":"` + captcha.New() + `"}`)
				return
			}
			if db.Hget("user_name2uid", []byte(nameLow)).State == "ok" {
				_, _ = ctx.WriteString(`{"Code":405,"Msg":"name is exist","NewCaptchaId":"` + captcha.New() + `"}`)
				return
			}
		}

		userId, _ := db.Hincr(model.CountTb, sdb.S2b(model.UserTbName), 1)
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
		_ = db.Hset(model.UserTbName, sdb.I2b(uobj.ID), jb)
		_ = db.Hset("user_name2uid", []byte(nameLow), sdb.I2b(userId))
		_ = db.Hset("user_flag:"+strconv.Itoa(flag), sdb.I2b(uobj.ID), nil)

		//生成头像
		_ = util.GenAvatar(db, uobj.ID, uobj.Name)
		autoAvatar = true

		_ = h.SetCookie(ctx, "SessionID", strconv.FormatUint(uobj.ID, 10)+":"+uobj.Session, 365)

		loggedUid = uobj.ID
	}

	// 绑定第三方登录信息
	openid := h.GetCookie(ctx, "openid") // authorKey
	if openid != "" {
		if rs := h.App.Db.Hget("oauth2user", []byte(openid)); rs.OK() {
			obj := model.AuthInfo{}
			_ = json.Unmarshal(rs.Bytes(), &obj)
			obj.Name = rec.Name
			obj.Uid = loggedUid
			jb, _ := json.Marshal(obj)
			_ = h.App.Db.Hset("oauth2user", []byte(openid), jb)
		}
		// 自动取头像
		if autoAvatar {
			if rs := h.App.Db.Hget("oauth_tmp_info", []byte(openid)); rs.OK() {
				obj := model.AuthProfileInfo{}
				_ = json.Unmarshal(rs.Bytes(), &obj)
				if obj.Avatar != "" {
					jb, _ := json.Marshal(model.AvatarTask{
						Uid:      loggedUid,
						Name:     rec.Name,
						Avatar:   obj.Avatar,
						SavePath: "static/avatar/" + strconv.FormatUint(loggedUid, 10) + ".jpg",
						Agent:    obj.Agent,
					})
					_ = db.Hset("task_to_get_avatar", sdb.I2b(loggedUid), jb)
				}
				// _ = db.Hdel("oauth_tmp_info", []byte(openid)) // 可删可不删
			}
		}
	}
	h.DelCookie(ctx, "openid")

	//
	h.DelCookie(ctx, "token")

	rsp := response{}
	rsp.Code = 200
	_ = json.NewEncoder(ctx).Encode(rsp)
}

func (h *BaseHandler) UserLogout(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID > 0 {
		cks := []string{"SessionID", "token"}
		for _, k := range cks {
			h.DelCookie(ctx, k)
		}
	}
	ctx.Redirect(h.App.Cf.Site.MainDomain+"/", fasthttp.StatusSeeOther)
}

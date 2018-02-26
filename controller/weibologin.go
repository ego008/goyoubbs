package controller

import (
	"encoding/json"
	"github.com/ego008/goyoubbs/lib/weiboOAuth"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/rs/xid"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *BaseHandler) WeiboOauthHandler(w http.ResponseWriter, r *http.Request) {
	scf := h.App.Cf.Site
	weibo, err := weiboOAuth.NewWeiboOAuth(strconv.Itoa(scf.WeiboClientID), scf.WeiboClientSecret, scf.MainDomain+"/oauth/wb/callback")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	// weiboOAuth.Logging = true

	now := time.Now().UTC().Unix()
	WeiboUrlState := strconv.FormatInt(now, 10)[6:]

	urlStr, err := weibo.GetAuthorizationURL(WeiboUrlState)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	h.SetCookie(w, "WeiboUrlState", WeiboUrlState, 1)
	http.Redirect(w, r, urlStr, http.StatusSeeOther)
}

func (h *BaseHandler) WeiboOauthCallback(w http.ResponseWriter, r *http.Request) {
	WeiboUrlState := h.GetCookie(r, "WeiboUrlState")
	if len(WeiboUrlState) == 0 {
		w.Write([]byte(`WeiboUrlState cookie missed`))
		return
	}

	scf := h.App.Cf.Site
	weibo, err := weiboOAuth.NewWeiboOAuth(strconv.Itoa(scf.WeiboClientID), scf.WeiboClientSecret, scf.MainDomain+"/oauth/wb/callback")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	// weiboOAuth.Logging = true

	code := r.FormValue("code")
	if code == "" {
		w.Write([]byte("Invalid code"))
		return
	}

	state := r.FormValue("state")
	if state != WeiboUrlState {
		w.Write([]byte("Invalid state"))
		return
	}

	token, err := weibo.GetAccessToken(code)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	wbUserID := token.UIDString

	timeStamp := uint64(time.Now().UTC().Unix())

	db := h.App.Db
	rs := db.Hget("oauth_weibo", []byte(wbUserID))
	if rs.State == "ok" {
		// login
		obj := model.QQ{}
		json.Unmarshal(rs.Data[0], &obj)
		uobj, err := model.UserGetById(db, obj.Uid)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		sessionid := xid.New().String()
		uobj.LastLoginTime = timeStamp
		uobj.Session = sessionid
		jb, _ := json.Marshal(uobj)
		db.Hset("user", youdb.I2b(uobj.Id), jb)
		h.SetCookie(w, "SessionID", strconv.FormatUint(uobj.Id, 10)+":"+sessionid, 365)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	profile, err := weibo.GetUserInfo(token.AccessToken, wbUserID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	// register

	siteCf := h.App.Cf.Site
	if siteCf.CloseReg {
		w.Write([]byte(`{"retcode":400,"retmsg":"stop to new register"}`))
		return
	}

	name := util.RemoveCharacter(profile.Name)
	name = strings.TrimSpace(strings.Replace(name, " ", "", -1))
	if len(name) == 0 {
		name = "wb"
	}
	var nameLow string
	i := 1
	for {
		nameLow = name + strconv.Itoa(i)
		if db.Hget("user_name2uid", []byte(nameLow)).State == "ok" {
			i++
		} else {
			name = nameLow
			break
		}
	}

	userId, _ := db.HnextSequence("user")
	flag := 5
	if siteCf.RegReview {
		flag = 1
	}
	if userId == 1 {
		flag = 99
	}

	gender := "female"
	if profile.Gender == "m" {
		gender = "male"
	}

	uobj := model.User{
		Id:            userId,
		Name:          name,
		About:         profile.Description,
		Url:           profile.URL,
		Gender:        gender,
		Flag:          flag,
		RegTime:       timeStamp,
		LastLoginTime: timeStamp,
		Session:       xid.New().String(),
	}

	uidStr := strconv.FormatUint(userId, 10)
	savePath := "static/avatar/" + uidStr + ".jpg"
	err = util.FetchAvatar(profile.Avatar, savePath, r.UserAgent())
	if err != nil {
		err = util.GenerateAvatar(gender, name, 73, 73, savePath)
	}
	if err != nil {
		uobj.Avatar = "0"
	} else {
		uobj.Avatar = uidStr
	}

	jb, _ := json.Marshal(uobj)
	db.Hset("user", youdb.I2b(uobj.Id), jb)
	db.Hset("user_name2uid", []byte(nameLow), youdb.I2b(userId))
	db.Hset("user_flag:"+strconv.Itoa(flag), youdb.I2b(uobj.Id), []byte(""))

	obj := model.WeiBo{
		Uid:    userId,
		Name:   name,
		Openid: wbUserID,
	}
	jb, _ = json.Marshal(obj)
	db.Hset("oauth_weibo", []byte(wbUserID), jb)

	h.SetCookie(w, "SessionID", strconv.FormatUint(uobj.Id, 10)+":"+uobj.Session, 365)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

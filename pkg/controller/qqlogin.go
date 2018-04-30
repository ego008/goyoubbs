/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"encoding/json"
	"github.com/ego008/youdb"
	"github.com/goxforum/xforum/pkg/lib/qqOAuth"
	"github.com/goxforum/xforum/pkg/model"
	"github.com/goxforum/xforum/pkg/util"
	"github.com/rs/xid"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *BaseHandler) QQOauthHandler(w http.ResponseWriter, r *http.Request) {
	scf := h.App.Cf.Site
	qq, err := qqOAuth.NewQQOAuth(strconv.Itoa(scf.QQClientID), scf.QQClientSecret, scf.MainDomain+"/oauth/qq/callback")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	// qqOAuth.Logging = true

	now := time.Now().UTC().Unix()
	qqUrlState := strconv.FormatInt(now, 10)[6:]

	urlStr, err := qq.GetAuthorizationURL(qqUrlState)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	h.SetCookie(w, "QQUrlState", qqUrlState, 1)
	http.Redirect(w, r, urlStr, http.StatusSeeOther)
}

func (h *BaseHandler) QQOauthCallback(w http.ResponseWriter, r *http.Request) {
	qqUrlState := h.GetCookie(r, "QQUrlState")
	if len(qqUrlState) == 0 {
		w.Write([]byte(`qqUrlState cookie missed`))
		return
	}

	scf := h.App.Cf.Site
	qq, err := qqOAuth.NewQQOAuth(strconv.Itoa(scf.QQClientID), scf.QQClientSecret, scf.MainDomain+"/oauth/qq/callback")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	// qqOAuth.Logging = true

	code := r.FormValue("code")
	if code == "" {
		w.Write([]byte("Invalid code"))
		return
	}

	state := r.FormValue("state")
	if state != qqUrlState {
		w.Write([]byte("Invalid state"))
		return
	}

	token, err := qq.GetAccessToken(code)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	openid, err := qq.GetOpenID(token.AccessToken)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	timeStamp := uint64(time.Now().UTC().Unix())

	db := h.App.Db
	rs := db.Hget("oauth_qq", []byte(openid.OpenID))
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

	profile, err := qq.GetUserInfo(token.AccessToken, openid.OpenID)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	if profile.Ret != 0 {
		w.Write([]byte(profile.Message))
		return
	}

	// register

	siteCf := h.App.Cf.Site
	if siteCf.CloseReg {
		w.Write([]byte(`{"retcode":400,"retmsg":"stop to new register"}`))
		return
	}

	name := util.RemoveCharacter(profile.Nickname)
	name = strings.TrimSpace(strings.Replace(name, " ", "", -1))
	if len(name) == 0 {
		name = "qq"
	}
	nameLow := strings.ToLower(name)
	i := 1
	for {
		if db.Hget("user_name2uid", []byte(nameLow)).State == "ok" {
			i++
			nameLow = name + strconv.Itoa(i)
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
	if profile.Gender == "男" {
		gender = "male"
	}

	uobj := model.User{
		Id:            userId,
		Name:          name,
		Flag:          flag,
		Gender:        gender,
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

	obj := model.QQ{
		Uid:    userId,
		Name:   name,
		Openid: openid.OpenID,
	}
	jb, _ = json.Marshal(obj)
	db.Hset("oauth_qq", []byte(openid.OpenID), jb)

	h.SetCookie(w, "SessionID", strconv.FormatUint(uobj.Id, 10)+":"+uobj.Session, 365)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

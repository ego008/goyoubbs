package controller

import (
	"bytes"
	"encoding/json"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/rs/xid"
	"goji.io/pat"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *BaseHandler) UserEdit(w http.ResponseWriter, r *http.Request) {
	uid := pat.Param(r, "uid")
	uidI, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"aid type err"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	db := h.App.Db

	uobj, err := model.UserGetById(db, uidI)
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	type pageData struct {
		PageData
		Uobj model.User
		Now  int64
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = "修改用户"
	evn.IsMobile = tpl == "mobile"
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "user_edit"

	evn.Uobj = uobj
	evn.Now = time.Now().UTC().Unix()

	h.SetCookie(w, "token", xid.New().String(), 1)
	h.Render(w, tpl, evn, "layout.html", "adminuseredit.html")
}

func (h *BaseHandler) UserEditPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	uid := pat.Param(r, "uid")
	uidI, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"aid type err"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	db := h.App.Db

	uobj, err := model.UserGetById(db, uidI)
	if err != nil {
		w.Write([]byte(`{"retcode":404,"retmsg":"` + err.Error() + `"}`))
		return
	}

	// r.ParseForm() // don't use ParseForm !important
	act := r.FormValue("act")
	if act == "avatar" {
		r.ParseMultipartForm(32 << 20)

		file, _, err := r.FormFile("avatar")
		defer file.Close()

		buff := make([]byte, 512)
		file.Read(buff)
		if len(util.CheckImageType(buff)) == 0 {
			w.Write([]byte(`{"retcode":400,"retmsg":"unknown image format"}`))
			return
		}

		var imgData bytes.Buffer
		file.Seek(0, 0)
		if fileSize, err := io.Copy(&imgData, file); err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"read image data err ` + err.Error() + `"}`))
			return
		} else {
			if fileSize > 5360690 {
				w.Write([]byte(`{"retcode":400,"retmsg":"image size too much"}`))
				return
			}
		}

		img, err := util.GetImageObj(&imgData)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"fail to get image obj ` + err.Error() + `"}`))
			return
		}

		err = util.AvatarResize(img, 73, 73, "static/avatar/"+uid+".jpg")
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"fail to resize avatar ` + err.Error() + `"}`))
			return
		}

		if uobj.Avatar == "0" || len(uobj.Avatar) == 0 {
			uobj.Avatar = uid
			model.UserUpdate(db, uobj)
		}

		http.Redirect(w, r, "/admin/user/edit/"+uid, http.StatusSeeOther)
		return
	}

	type recForm struct {
		Act      string `json:"act"`
		Flag     int    `json:"flag"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		Url      string `json:"url"`
		About    string `json:"about"`
		Password string `json:"password"`
		Hidden   string `json:"hidden"`
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err = decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	recAct := rec.Act
	if len(recAct) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"missed act "}`))
		return
	}

	var hidden bool
	if rec.Hidden == "1" {
		hidden = true
	}

	isChanged := false
	if recAct == "info" {
		oldName := uobj.Name
		nameLow := strings.ToLower(rec.Name)
		if !util.IsNickname(nameLow) {
			w.Write([]byte(`{"retcode":400,"retmsg":"name fmt err"}`))
			return
		}

		uobj.Email = rec.Email
		uobj.Url = rec.Url
		uobj.About = rec.About
		uobj.Hidden = hidden
		isChanged = true

		if oldName != rec.Name {
			if db.Hget("user_name2uid", []byte(nameLow)).State == "ok" {
				w.Write([]byte(`{"retcode":400,"retmsg":"name is exist"}`))
				return
			}
			db.Hdel("user_name2uid", []byte(strings.ToLower(oldName)))
			db.Hset("user_name2uid", []byte(nameLow), youdb.I2b(uobj.Id))
			uobj.Name = rec.Name
		}
	} else if recAct == "change_pw" {
		if len(rec.Password) == 0 {
			w.Write([]byte(`{"retcode":400,"retmsg":"missed args"}`))
			return
		}
		uobj.Password = rec.Password
		isChanged = true
	} else if recAct == "flag" {
		if rec.Flag != uobj.Flag {
			oldFlag := strconv.Itoa(uobj.Flag)
			isChanged = true
			uobj.Flag = rec.Flag

			db.Hset("user_flag:"+strconv.Itoa(rec.Flag), youdb.I2b(uobj.Id), []byte(""))
			db.Hdel("user_flag:"+oldFlag, youdb.I2b(uobj.Id))
		}
	}

	if isChanged {
		model.UserUpdate(db, uobj)
	}

	type response struct {
		normalRsp
	}

	rsp := response{}
	rsp.Retcode = 200
	rsp.Retmsg = "修改成功"
	json.NewEncoder(w).Encode(rsp)
}

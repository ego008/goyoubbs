package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"log"
	"net/http"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) UserSettingPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagReview {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.String(200, "403: forbidden")
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.UserSetting{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "个人设置"

	//
	evn.User = evn.CurrentUser

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) UserSettingPost(c *gin.Context) {

	curUser, _ := h.CurrentUser(c)
	if curUser.ID == 0 {
		c.String(200, `{"Code":403,"Msg":"author is none"}`)
		return
	}

	type recForm struct {
		Password0 string
		Password  string
		Url       string
		About     string
	}

	var rec recForm
	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		log.Println(err)
		c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	type response struct {
		model.NormalRsp
	}
	rsp := response{}

	db := h.App.Db

	// 编辑
	obj := *curUser

	if len(rec.Password) > 0 && len(rec.Password0) > 0 {
		if rec.Password0 != obj.Password {
			c.String(200, `{"Code":403,"Msg":"原密码不对"}`)
			return
		}
		obj.Password = rec.Password
	}

	obj.Url = rec.Url
	obj.About = rec.About

	obj = model.UserSet(db, obj)

	rsp.Code = 200
	rsp.Msg = "已成功更新"
	_ = json.NewEncoder(c.Writer).Encode(rsp)
}

package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"log"
)

func (h *BaseHandler) UserSettingPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagReview {
		if curUser.ID == 0 {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
			return
		}
		_, _ = ctx.WriteString("403: forbidden")
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.UserSetting{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "个人设置"

	//
	evn.User = curUser

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ybs.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")
}

func (h *BaseHandler) UserSettingPost(ctx *fasthttp.RequestCtx) {

	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID == 0 {
		_, _ = ctx.WriteString(`{"Code":403,"Msg":"author is none"}`)
		return
	}

	type recForm struct {
		Password0 string
		Password  string
		Url       string
		About     string
	}

	var rec recForm
	err := util.Bind(ctx, util.JSON, &rec)
	if err != nil {
		log.Println(err)
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	type response struct {
		model.NormalRsp
	}
	rsp := response{}

	db := h.App.Db

	// 编辑
	obj := curUser

	if len(rec.Password) > 0 && len(rec.Password0) > 0 {
		if rec.Password0 != obj.Password {
			_, _ = ctx.WriteString(`{"Code":403,"Msg":"原密码不对"}`)
			return
		}
		obj.Password = rec.Password
	}

	obj.Url = rec.Url
	obj.About = rec.About

	obj = model.UserSet(db, obj)

	rsp.Code = 200
	rsp.Msg = "已成功更新"
	_ = json.NewEncoder(ctx).Encode(rsp)
}

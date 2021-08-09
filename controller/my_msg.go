package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
)

func (h *BaseHandler) MyMsgPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	evn := &model.MyMsg{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "未读信息"

	evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
	evn.TopicPageInfo = model.GetMsgTopicList(db, curUser.ID)

	evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
	if curUser.Flag >= model.FlagAdmin {
		evn.HasTopicReview = model.CheckHasTopic2Review(db)
		evn.HasReplyReview = model.CheckHasComment2Review(db)
	}

	model.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")

	//_ = h.Render(ctx, evn, "admin/layout.html", "admin/my_msg.html")
}

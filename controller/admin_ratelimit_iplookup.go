package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/admin"
)

func (h *BaseHandler) AdminRateLimitIpLookup(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.IpLookup{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "Ip Lookup"
	evn.PageName = "admin_IpLookup"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, true)

	//todo

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}

func (h *BaseHandler) AdminRateLimitIpLookupPost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	// todo

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/ratelimit/iplookup", 302)
}

package controller

import (
	"github.com/valyala/fasthttp"
)

func (h *BaseHandler) AdminHomePage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/topic/add", 302)
	return

	/*
		scf := h.App.Cf.Site
		type PageData struct {
			model.BasePage
		}
		evn := &PageData{}
		evn.CurrentUser = curUser
		evn.SiteCf = scf
		evn.Title = scf.Name

		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
			evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)
		}

		_ = h.Render(ctx, evn, "admin/layout.html", "admin/home.html")

	*/
}

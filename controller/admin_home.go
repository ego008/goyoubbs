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
}

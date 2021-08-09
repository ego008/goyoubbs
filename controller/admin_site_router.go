package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"log"
	"strings"
)

func (h *BaseHandler) AdminSiteRouterPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &model.AdminSiteRouter{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "自定义路由"
	evn.PageName = "admin_site_router"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	evn.TypeLst = []string{
		"text/html; charset=utf-8",
		"text/plain; charset=utf-8",
	}
	evn.ObjLst = model.CustomRouterGetAll(h.App.Db)
	// 内容显示300字
	for i, v := range evn.ObjLst {
		if len(v.Content) > 300 {
			evn.ObjLst[i].Content = v.Content[:300] + " ......"
		}
	}

	//
	evn.Obj = model.CustomRouter{}
	key := ctx.FormValue("key")
	if len(key) > 0 {
		if sdb.B2s(ctx.FormValue("act")) == "del" {
			_ = h.App.Db.Hdel("custom_router", key)

			// fix del, reload mux
			RouterReload(h.App)

			ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/site/router", 302)
			return
		}
		evn.Obj = model.CustomRouterGetByKey(h.App.Db, key)
		if evn.Obj.Router == "" {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/site/router", 302)
			return
		}
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	model.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")
}

func (h *BaseHandler) AdminSiteRouterPost(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
		return
	}

	var isAdd bool
	obj := model.CustomRouter{}
	key := ctx.FormValue("key")
	if len(key) > 0 {
		// edit
		obj = model.CustomRouterGetByKey(h.App.Db, key)
		if obj.Router == "" {
			ctx.NotFound()
			return
		}
	} else {
		// add
		isAdd = true
		obj.Router = sdb.B2s(ctx.FormValue("Router"))
		if !strings.HasPrefix(obj.Router, "/") {
			obj.Router = "/" + obj.Router
		}
		// check 路径是否已存在
		for _, v := range h.App.Mux.List()["GET"] {
			if v == obj.Router {
				// 直接转
				log.Println("a handler is already registered for path", v)
				ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/site/router", 302)
				return
			}
		}
	}

	obj.MimeType = sdb.B2s(ctx.FormValue("MimeType"))
	obj.Content = sdb.B2s(ctx.FormValue("Content"))

	model.CustomRouterSet(h.App.Db, obj)

	// add new rooter
	if isAdd {
		h.App.Mux.GET(obj.Router, func(ctx *fasthttp.RequestCtx) {
			k := ctx.Path()
			rs2 := h.App.Db.Hget("custom_router", k)
			if !rs2.OK() {
				ctx.NotFound()
				return
			}
			obj2 := model.CustomRouter{}
			_ = json.Unmarshal(rs2.Bytes(), &obj2)
			if strings.HasPrefix(obj2.Content, "goto:") {
				ctx.Redirect(strings.TrimSpace(obj2.Content[5:]), 302)
				return
			}
			ctx.SetContentType(obj2.MimeType)
			_, _ = ctx.WriteString(obj2.Content)
		})
	}

	ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/site/router", 302)
}

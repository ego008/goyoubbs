package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"log"
	"net/http"
	"strings"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AdminSiteRouterPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.SiteRouter{}
	evn.CurrentUser = *curUser
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
	key := c.Query("key")
	if len(key) > 0 {
		if c.Query("act") == "del" {
			_ = h.App.Db.Hdel("custom_router", []byte(key))

			// fix del, reload mux
			RouterReload(h.App)

			c.Redirect(302, "/admin/site/router")
			return
		}
		evn.Obj = model.CustomRouterGetByKey(h.App.Db, []byte(key))
		if evn.Obj.Router == "" {
			c.Redirect(302, "/admin/site/router")
			return
		}
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) AdminSiteRouterPost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	var isAdd bool
	obj := model.CustomRouter{}
	key := c.Query("key")
	if len(key) > 0 {
		// edit
		obj = model.CustomRouterGetByKey(h.App.Db, []byte(key))
		if obj.Router == "" {
			c.Status(http.StatusNotFound)
			return
		}
	} else {
		// add
		isAdd = true
		obj.Router = c.PostForm("Router")
		if !strings.HasPrefix(obj.Router, "/") {
			obj.Router = "/" + obj.Router
		}
		// check 路径是否已存在
		for _, route := range h.App.Mux.Routes() {
			if route.Method == "GET" && route.Path == obj.Router {
				// 直接转
				log.Println("a handler is already registered for path", obj.Router)
				c.Redirect(302, "/admin/site/router")
				return
			}
		}
	}

	obj.MimeType = c.PostForm("MimeType")
	obj.Content = c.PostForm("Content")

	model.CustomRouterSet(h.App.Db, obj)

	// add new rooter
	if isAdd {
		h.App.Mux.GET(obj.Router, func(c *gin.Context) {
			k := c.Request.URL.Path
			rs2 := h.App.Db.Hget("custom_router", []byte(k))
			if !rs2.OK() {
				c.Status(http.StatusNotFound)
				return
			}
			obj2 := model.CustomRouter{}
			_ = json.Unmarshal(rs2.Bytes(), &obj2)
			if strings.HasPrefix(obj2.Content, "goto:") {
				c.Redirect(302, strings.TrimSpace(obj2.Content[5:]))
				return
			}
			c.Header("Content-Type", obj2.MimeType)
			c.String(200, obj2.Content)
		})
	}

	c.Redirect(302, "/admin/site/router")
}

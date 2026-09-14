package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/admin"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AdminNodePage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.Node{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "分区管理"
	evn.PageName = "admin_node"

	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	//
	evn.Act = "添加"
	evn.Node = model.Node{}
	_id := c.Query("id")
	if len(_id) > 0 {
		idi, _ := strconv.ParseUint(_id, 10, 64)
		evn.Node = model.Node{}
		node, code := model.NodeGetById(h.App.Db, idi)
		if code == 1 {
			evn.Node = node
			evn.Act = "编辑"
		}
	}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) AdminNodePost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	var id uint64
	_id := c.Query("id")
	if len(_id) > 0 {
		idI, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			id = idI
		}
	}

	obj := model.Node{}
	obj.ID = id
	obj.Name = c.PostForm("Name")
	obj.Score, _ = strconv.Atoi(c.PostForm("Score"))
	obj.About = c.PostForm("About")

	_, _ = model.NodeSet(h.App.Db, obj)

	// 删除缓存
	h.App.Mc.Del([]byte("NodeGetAll"))

	c.Redirect(302, "/admin/node")
}

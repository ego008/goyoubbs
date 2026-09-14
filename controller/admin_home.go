package controller

import (
	"goyoubbs/model"

	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AdminHomePage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	c.Redirect(302, "/admin/topic/add")
}

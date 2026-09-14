package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AdminImgPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	ts := util.TimeFmt(time.Now().Unix(), "20060102150405")
	zipName := "img_" + ts + ".zip"
	defer func() {
		_ = os.Remove(zipName)
	}()

	err := zipIt(h.App.Cf.Site.UploadDir, zipName)
	if err != nil {
		log.Println(err)
		return
	}

	c.Header("Content-Type", "application/zip")

	c.Header("Content-Disposition", "attachment; filename="+zipName)
	c.File(zipName)
	return
}

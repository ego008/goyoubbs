package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/util"
	"log"
	"os"
	"time"
)

func (h *BaseHandler) AdminImgPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.ID == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
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

	ctx.SetContentType("application/zip")

	ctx.Response.Header.Set("Content-Disposition", "attachment; filename="+zipName)
	ctx.SendFile(zipName)
	return
}

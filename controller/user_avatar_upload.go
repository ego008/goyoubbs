package controller

import (
	"bytes"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"image"
	"image/jpeg"
	"io"
	"strconv"
)

func (h *BaseHandler) AvatarUpload(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(ctx)

	if curUser.Flag == 0 {
		_, _ = ctx.WriteString(`{"Code":401,"Msg":"请先登录"}`)
		return
	}

	uid := sdb.B2s(ctx.FormValue("UserId"))
	uidI64, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"uid fmt err"}`)
		return
	}
	user, code := model.UserGetById(h.App.Db, uidI64)
	if code != 1 {
		_, _ = ctx.WriteString(`{"Code":404,"Msg":"user not found"}`)
		return
	}
	if curUser.Flag < model.FlagAdmin && user.ID != curUser.ID {
		_, _ = ctx.WriteString(`{"Code":403,"Msg":"can not set other member avatar"}`)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":500,"Msg":"` + err.Error() + `"}`)
		return
	}
	fileHandler, err := file.Open()
	if err != nil {
		_, _ = ctx.WriteString(`{"code":400,"msg":"` + err.Error() + `"}`)
		return
	}
	defer func() {
		_ = fileHandler.Close()
	}()

	var imgData bytes.Buffer
	if fileSize, err := io.Copy(&imgData, fileHandler); err != nil {
		_, _ = ctx.WriteString(`{"code":400,"msg":"` + err.Error() + `"}`)
		return
	} else {
		if fileSize > 5360690 {
			_, _ = ctx.WriteString(`{"code":400,"msg":"image size too much"}`)
			return
		}
	}

	buff := make([]byte, 512)
	if len(imgData.Bytes()) > 512 {
		buff = imgData.Bytes()[:512]
	}
	if len(util.CheckImageType(buff)) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"unknown image format"}`)
		return
	}

	var img image.Image
	img, err = util.GetImageObj(&imgData)
	imgData.Reset()
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	}

	dstImg := util.ImageResize(img, 119, 119)

	type response struct {
		model.NormalRsp
		Url string
	}

	rsp := response{}

	savePath := "static/avatar/" + uid + ".jpg"

	// save to db
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, dstImg, &jpeg.Options{Quality: 95})
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	}
	err = h.App.Db.Hset("user_avatar", sdb.I2b(uidI64), buf.Bytes())
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	}

	// save to local
	/*
		var f3 *os.File
		f3, err = os.Create(savePath)
		if err != nil {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}
		defer func() {
			_ = f3.Close()
		}()

		err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
		if err != nil {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}

	*/

	rsp.Code = 200
	rsp.Msg = "上传成功"
	rsp.Url = "/" + savePath

	_ = json.NewEncoder(ctx).Encode(rsp)
}

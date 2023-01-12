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
	"os"
	"strconv"
)

const (
	fileMaxSize = 10 << 20 // 10 MB
	imgMaxWidth = 1920
)

func (h *BaseHandler) FileUpload(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(ctx)

	if curUser.Flag == 0 {
		_, _ = ctx.WriteString(`{"Code":401,"Msg":"请先登录"}`)
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
		if fileSize > fileMaxSize {
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

	imgHashValue := util.Xxhash(imgData.Bytes())

	var img image.Image
	img, err = util.GetImageObj(&imgData)
	imgData.Reset()
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	}

	saveName := strconv.FormatUint(imgHashValue, 10) + ".jpg"
	showPath := "/static/upload/" + saveName
	saveFullPath := h.App.Cf.Site.UploadDir + "/" + saveName
	dstImg := util.ImageResize(img, imgMaxWidth, 0) // 1024

	type response struct {
		model.NormalRsp
		Url string
	}

	rsp := response{}

	db := h.App.Db

	if db.Hget("local_upload_md5_key", sdb.I2b(imgHashValue)).OK() {
		rsp.Code = 200
		rsp.Url = showPath
		rsp.Msg = "上传成功"
		_ = json.NewEncoder(ctx).Encode(rsp)
		return
	}

	var f3 *os.File
	f3, err = os.Create(saveFullPath)
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

	rsp.Code = 200
	rsp.Msg = "上传成功"
	rsp.Url = showPath

	// 保存hash值
	_ = db.Hset("local_upload_md5_key", sdb.I2b(imgHashValue), sdb.I2b(curUser.ID))

	_ = json.NewEncoder(ctx).Encode(rsp)
}

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
	"path"
	"strconv"
)

const (
	fileMaxSize = 2000 << 20 // 200 MB
	imgMaxWidth = 1920
)

type response struct {
	model.NormalRsp
	Url string
}

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
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	}
	defer func() {
		_ = fileHandler.Close()
	}()

	var imgData bytes.Buffer
	if fileSize, err := io.Copy(&imgData, fileHandler); err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
		return
	} else {
		if fileSize > fileMaxSize {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"image size too much"}`)
			return
		}
	}

	// check image
	buff := make([]byte, 512)
	if len(imgData.Bytes()) > 512 {
		buff = imgData.Bytes()[:512]
	}
	var fileIsImage bool
	imgType := util.CheckImageType(buff)
	if len(imgType) > 0 {
		fileIsImage = true
	}

	imgHashValue := util.Xxhash(imgData.Bytes())
	imgKeyS := strconv.FormatUint(imgHashValue, 10)
	imgKeyB := sdb.I2b(imgHashValue)

	var fileSuffix, saveName, showPath string

	if fileIsImage {
		if imgType == "gif" {
			saveName = imgKeyS + ".gif"
		} else {
			saveName = imgKeyS + ".jpg"
		}
		if h.App.Cf.Site.SaveImg2db {
			showPath = "/dbi/" + saveName
		} else {
			showPath = "/static/upload/" + saveName
		}
	} else {
		fileSuffix = path.Ext(file.Filename) // source file suffix
		saveName = imgKeyS + fileSuffix
		showPath = "/static/upload/" + saveName
	}

	saveFullPath := h.App.Cf.Site.UploadDir + "/" + saveName

	rsp := response{}

	db := h.App.Db

	if db.Hget("local_upload_md5_key", imgKeyB).OK() {
		rsp.Code = 200
		rsp.Url = showPath
		rsp.Msg = "上传成功"
		_ = json.NewEncoder(ctx).Encode(rsp)

		// fix
		if fileSuffix == ".mp4" {
			_ = db.Hset(model.TbnV2DecMp4, []byte(saveFullPath), nil)
		} else if fileSuffix == ".mp3" {
			model.Mp3InfoSet(db, saveFullPath)
		}

		return
	}

	if imgType == "gif" {
		if h.App.Cf.Site.SaveImg2db {
			// db
			if err = db.Hset(model.TbnDbImg, imgKeyB, imgData.Bytes()); err != nil {
				_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
				imgData.Reset()
				return
			}
		} else {
			// local
			if err = os.WriteFile(saveFullPath, imgData.Bytes(), 0644); err != nil {
				_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
				imgData.Reset()
				return
			}
		}
		imgData.Reset()
		rsp.Code = 200
		rsp.Msg = "上传成功"
		rsp.Url = showPath

		// 保存hash值
		_ = db.Hset("local_upload_md5_key", imgKeyB, sdb.I2b(curUser.ID))

		_ = json.NewEncoder(ctx).Encode(rsp)
		return
	}

	if fileIsImage {
		var img image.Image
		img, err = util.GetImageObj(&imgData)
		imgData.Reset()
		if err != nil {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}

		dstImg := util.ImageResize(img, imgMaxWidth, 0) // 1024

		buf := new(bytes.Buffer)
		if err = jpeg.Encode(buf, dstImg, &jpeg.Options{Quality: 95}); err != nil {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}

		if h.App.Cf.Site.SaveImg2db {
			// db
			if err = db.Hset(model.TbnDbImg, imgKeyB, buf.Bytes()); err != nil {
				_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
				return
			}
		} else {
			// local
			if err = os.WriteFile(saveFullPath, buf.Bytes(), 0644); err != nil {
				_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
				return
			}
		}
	} else {
		// local
		if err = os.WriteFile(saveFullPath, imgData.Bytes(), 0644); err != nil {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"` + err.Error() + `"}`)
			imgData.Reset()
			return
		}
		imgData.Reset()
		if fileSuffix == ".mp4" {
			_ = db.Hset(model.TbnV2DecMp4, []byte(saveFullPath), nil)
		} else if fileSuffix == ".mp3" {
			model.Mp3InfoSet(db, saveFullPath)
		}
	}

	rsp.Code = 200
	rsp.Msg = "上传成功"
	rsp.Url = showPath

	// 保存hash值
	_ = db.Hset("local_upload_md5_key", imgKeyB, sdb.I2b(curUser.ID))

	_ = json.NewEncoder(ctx).Encode(rsp)
}

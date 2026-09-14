package controller

import (
	"bytes"
	"goyoubbs/model"
	"goyoubbs/util"
	"image"
	"image/jpeg"
	"io"
	"strconv"

	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) AvatarUpload(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(c)

	if curUser.Flag == 0 {
		c.String(200, `{"Code":401,"Msg":"请先登录"}`)
		return
	}

	uid := c.Query("UserId")
	uidI64, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"uid fmt err"}`)
		return
	}
	user, code := model.UserGetById(h.App.Db, uidI64)
	if code != 1 {
		c.String(200, `{"Code":404,"Msg":"user not found"}`)
		return
	}
	if curUser.Flag < model.FlagAdmin && user.ID != curUser.ID {
		c.String(200, `{"Code":403,"Msg":"can not set other member avatar"}`)
		return
	}
	// 1. 直接获取上传的文件 Header
	fileHeader, err := c.FormFile("image")
	if err != nil {
		// 获取失败（如未提供文件或表单解析失败）
		c.String(200, `{"Code":500,"Msg":"`+err.Error()+`"}`)
		return
	}

	// 2. 打开文件流
	fileHandler, err := fileHeader.Open()
	if err != nil {
		c.String(200, `{"code":400,"msg":"`+err.Error()+`"}`)
		return
	}
	defer fileHandler.Close()

	var imgData bytes.Buffer
	if fileSize, err := io.Copy(&imgData, fileHandler); err != nil {
		c.String(200, `{"code":400,"msg":"`+err.Error()+`"}`)
		return
	} else {
		if fileSize > 5360690 {
			c.String(200, `{"code":400,"msg":"image size too much"}`)
			return
		}
	}

	buff := make([]byte, 512)
	if len(imgData.Bytes()) > 512 {
		buff = imgData.Bytes()[:512]
	}
	if len(util.CheckImageType(buff)) == 0 {
		c.String(200, `{"Code":400,"Msg":"unknown image format"}`)
		return
	}

	var img image.Image
	img, err = util.GetImageObj(&imgData)
	imgData.Reset()
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"`+err.Error()+`"}`)
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
		c.String(200, `{"Code":400,"Msg":"`+err.Error()+`"}`)
		return
	}
	err = h.App.Db.Hset("user_avatar", sdb.I2b(uidI64), buf.Bytes())
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"`+err.Error()+`"}`)
		return
	}

	// save to local
	/*
		var f3 *os.File
		f3, err = os.Create(savePath)
		if err != nil {
			c.String(200, `{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}
		defer func() {
			_ = f3.Close()
		}()

		err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
		if err != nil {
			c.String(200, `{"Code":400,"Msg":"` + err.Error() + `"}`)
			return
		}

	*/

	rsp.Code = 200
	rsp.Msg = "上传成功"
	rsp.Url = "/" + savePath

	_ = json.NewEncoder(c.Writer).Encode(rsp)
}

package controller

import (
	"bytes"
	"fmt"
	"goyoubbs/model"
	"goyoubbs/util"
	"image"
	"image/jpeg"
	"io"
	"os"
	"strconv"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

const (
	fileMaxSize = 2000 << 20 // 200 MB
	imgMaxWidth = 1920
)

type response struct {
	model.NormalRsp
	Url string
}

func (h *BaseHandler) FileUpload(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(c)

	if curUser.Flag == 0 {
		c.String(200, `{"Code":401,"Msg":"请先登录"}`)
		return
	}

	// 1. 直接获取上传的文件 Header
	fileHeader, err := c.FormFile("file")
	if err != nil {
		// 获取失败（如未提供文件或表单解析失败）
		fmt.Println(err)
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
		c.String(200, `{"Code":400,"Msg":"`+err.Error()+`"}`)
		return
	} else {
		if fileSize > fileMaxSize {
			c.String(200, `{"Code":400,"Msg":"image size too much"}`)
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
	imgKeyB := mdb.I2b(imgHashValue)

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
		// is mp3 or mp4
		mediaType := util.CheckMediaType(buff)
		if len(mediaType) > 0 {
			fileSuffix = "." + mediaType
			saveName = imgKeyS + fileSuffix
			showPath = "/static/upload/" + saveName
		} else {
			c.String(200, `{"Code":400,"Msg":"unknown image or media format"}`)
			return
		}
		//fileSuffix = path.Ext(file.Filename) // source file suffix
		//saveName = imgKeyS + fileSuffix
		//showPath = "/static/upload/" + saveName
	}

	saveFullPath := h.App.Cf.Site.UploadDir + "/" + saveName

	rsp := response{}

	db := h.App.Db

	var showStr string

	_ = db.Update(func(tx *bbolt.Tx) error {
		if db.HKeyExist(tx, "local_upload_md5_key", imgKeyB) {
			// fix
			if fileSuffix == ".mp4" {
				_ = db.HSet(tx, model.TbnV2DecMp4, []byte(saveFullPath), nil)
			} else if fileSuffix == ".mp3" {
				model.Mp3InfoSet(db, tx, saveFullPath)
			}
			return nil
		}

		if imgType == "gif" {
			if h.App.Cf.Site.SaveImg2db {
				// db
				if err = db.HSet(tx, model.TbnDbImg, imgKeyB, imgData.Bytes()); err != nil {
					showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
					imgData.Reset()
					return nil
				}
			} else {
				// local
				if err = os.WriteFile(saveFullPath, imgData.Bytes(), 0644); err != nil {
					showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
					imgData.Reset()
					return nil
				}
			}
			imgData.Reset()

			// 保存hash值
			_ = db.HSet(tx, "local_upload_md5_key", imgKeyB, mdb.I2b(curUser.ID))

			return nil
		}

		if fileIsImage {
			var img image.Image
			img, err = util.GetImageObj(&imgData)
			imgData.Reset()
			if err != nil {
				showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
				return nil
			}

			dstImg := util.ImageResize(img, imgMaxWidth, 0) // 1024

			buf := new(bytes.Buffer)
			if err = jpeg.Encode(buf, dstImg, &jpeg.Options{Quality: 95}); err != nil {
				showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
				return nil
			}

			if h.App.Cf.Site.SaveImg2db {
				// db
				if err = db.HSet(tx, model.TbnDbImg, imgKeyB, buf.Bytes()); err != nil {
					showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
					return nil
				}
			} else {
				// local
				if err = os.WriteFile(saveFullPath, buf.Bytes(), 0644); err != nil {
					showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
					return nil
				}
			}
		} else {
			// local
			if err = os.WriteFile(saveFullPath, imgData.Bytes(), 0644); err != nil {
				showStr = `{"Code":400,"Msg":"` + err.Error() + `"}`
				imgData.Reset()
				return nil
			}
			imgData.Reset()
			if fileSuffix == ".mp4" {
				_ = db.HSet(tx, model.TbnV2DecMp4, []byte(saveFullPath), nil)
			} else if fileSuffix == ".mp3" {
				model.Mp3InfoSet(db, tx, saveFullPath)
			}
		}

		// 保存hash值
		_ = db.HSet(tx, "local_upload_md5_key", imgKeyB, mdb.I2b(curUser.ID))

		return nil
	})

	if len(showStr) > 0 {
		c.String(200, showStr)
		return
	}

	rsp.Code = 200
	rsp.Msg = "上传成功"
	rsp.Url = showPath

	_ = json.NewEncoder(c.Writer).Encode(rsp)
}

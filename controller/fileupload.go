package controller

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/ego008/goyoubbs/lib/upyun"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"gopkg.in/h2non/filetype.v1"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"strings"
)

func (h *BaseHandler) FileUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 5 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	r.ParseMultipartForm(32 << 20)

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		w.Write([]byte(`{"retcode":500,"retmsg":"` + err.Error() + `"}`))
		return
	}
	defer file.Close()

	scf := h.App.Cf.Site

	buff := make([]byte, 512)
	file.Read(buff)
	imgType := util.CheckImageType(buff)
	if scf.UploadImgOnly && len(imgType) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"unknown image format"}`))
		return
	}

	var fileData bytes.Buffer
	file.Seek(0, 0)
	if fileSize, err := io.Copy(&fileData, file); err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
		return
	} else {
		if fileSize > scf.UploadMaxSizeByte {
			w.Write([]byte(`{"retcode":400,"retmsg":"image size too much"}`))
			return
		}
	}

	var fileEx string
	kind, unknown := filetype.Match(buff)
	if unknown == nil {
		fileEx = kind.Extension
	} else {
		tokens := strings.Split(fileHeader.Filename, ".")
		if len(tokens) > 1 {
			fileEx = tokens[len(tokens)-1]
		} else {
			fileEx = "unknown"
		}
	}

	// check Suffix
	if len(scf.UploadSuffix) > 2 {
		var allow bool
		for _, v := range strings.Split(scf.UploadSuffix, ",") {
			if strings.TrimSpace(v) == fileEx {
				allow = true
				break
			}
		}
		if !allow {
			w.Write([]byte(`{"retcode":403,"retmsg":"UploadSuffix limited ` + scf.UploadSuffix + `"}`))
			return
		}
	}

	hash := md5.Sum(fileData.Bytes())
	fileMd5 := hex.EncodeToString(hash[:])
	savePath := "static/upload/" + fileMd5 + "." + fileEx

	//
	var dstImg *image.NRGBA

	// defer fileData.Reset()
	var imgTransform bool

	if scf.UploadImgResize && len(imgType) > 0 && imgType != "gif" {
		imgTransform = true
	}

	if imgTransform {
		img, err := util.GetImageObj(&fileData)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}

		dstImg = util.ImageResize(img, 1024, 0)
	}

	type response struct {
		normalRsp
		Url string `json:"url"`
	}

	rsp := response{}

	db := h.App.Db

	if len(scf.QiniuSecretKey) > 0 && len(scf.QiniuAccessKey) > 0 {
		// qiniu

		if db.Hget("qiniu_upload_md5_key", []byte(fileMd5)).State == "ok" {
			rsp.Retcode = 200
			rsp.Url = scf.QiniuDomain + "/" + savePath
			rsp.Retmsg = "上传成功"
			json.NewEncoder(w).Encode(rsp)
			return
		}

		qmac := qbox.NewMac(scf.QiniuAccessKey, scf.QiniuSecretKey)
		putPolicy := storage.PutPolicy{
			//Scope: scf.QiniuBucket,
			Scope: scf.QiniuBucket + ":" + savePath,
		}
		upToken := putPolicy.UploadToken(qmac)
		cfg := storage.Config{}

		if h.App.QnZone == nil {
			zone, err := storage.GetZone(scf.QiniuAccessKey, scf.QiniuBucket)
			if err != nil {
				w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
				return
			}
			h.App.QnZone = zone
		}

		cfg.Zone = h.App.QnZone
		cfg.UseHTTPS = false                          // 是否使用https域名
		cfg.UseCdnDomains = false                     // 上传是否使用CDN上传加速
		formUploader := storage.NewFormUploader(&cfg) // 构建表单上传的对象
		ret := storage.PutRet{}

		if imgTransform {
			var newImgData bytes.Buffer
			err = jpeg.Encode(&newImgData, dstImg, &jpeg.Options{Quality: 95})
			if err != nil {
				w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
				return
			}
			err = formUploader.Put(context.Background(), &ret, upToken, savePath, bytes.NewReader(newImgData.Bytes()), int64(newImgData.Len()), nil)
			newImgData.Reset()
		} else {
			err = formUploader.Put(context.Background(), &ret, upToken, savePath, bytes.NewReader(fileData.Bytes()), int64(fileData.Len()), nil)
		}
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}

		db.Hset("qiniu_upload_md5_key", []byte(fileMd5), youdb.I2b(currentUser.Id))

		rsp.Url = scf.QiniuDomain + "/" + savePath
	} else if len(scf.UpyunUser) > 0 && len(scf.UpyunPw) > 0 {
		// upyun

		if db.Hget("upyun_upload_md5_key", []byte(fileMd5)).State == "ok" {
			rsp.Retcode = 200
			rsp.Url = scf.UpyunDomain + "/" + savePath
			rsp.Retmsg = "上传成功"
			json.NewEncoder(w).Encode(rsp)
			return
		}

		upy := upyun.NewUpYun(scf.UpyunBucket, scf.UpyunUser, scf.UpyunPw)

		if imgTransform {
			var newImgData bytes.Buffer
			err = jpeg.Encode(&newImgData, dstImg, &jpeg.Options{Quality: 95})
			if err != nil {
				w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
				return
			}
			err = upy.WriteFile(savePath, "", true, newImgData.Bytes())
			newImgData.Reset()
		} else {
			err = upy.WriteFile(savePath, "", true, fileData.Bytes())
		}
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}
		db.Hset("upyun_upload_md5_key", []byte(fileMd5), youdb.I2b(currentUser.Id))

		rsp.Url = scf.UpyunDomain + "/" + savePath
	} else {
		// local

		if db.Hget("local_upload_md5_key", []byte(fileMd5)).State == "ok" {
			rsp.Retcode = 200
			rsp.Url = scf.MainDomain + "/" + savePath
			rsp.Retmsg = "上传成功"
			json.NewEncoder(w).Encode(rsp)
			return
		}

		f3, err := os.Create(savePath)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}
		defer f3.Close()

		if imgTransform {
			err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
		} else {
			_, err = io.Copy(f3, &fileData)
		}
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}

		db.Hset("local_upload_md5_key", []byte(fileMd5), youdb.I2b(currentUser.Id))

		rsp.Url = scf.MainDomain + "/" + savePath
	}

	rsp.Retcode = 200
	rsp.Retmsg = "上传成功"
	json.NewEncoder(w).Encode(rsp)
}

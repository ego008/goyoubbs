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
	"image/jpeg"
	"io"
	"net/http"
	"os"
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

	file, _, err := r.FormFile("file")
	if err != nil {
		w.Write([]byte(`{"retcode":500,"retmsg":"` + err.Error() + `"}`))
		return
	}
	defer file.Close()

	buff := make([]byte, 512)
	file.Read(buff)
	if len(util.CheckImageType(buff)) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"unknown image format"}`))
		return
	}

	var imgData bytes.Buffer
	file.Seek(0, 0)
	if fileSize, err := io.Copy(&imgData, file); err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
		return
	} else {
		if fileSize > 5360690 {
			w.Write([]byte(`{"retcode":400,"retmsg":"image size too much"}`))
			return
		}
	}

	hash := md5.Sum(imgData.Bytes())
	fileMd5 := hex.EncodeToString(hash[:])

	img, err := util.GetImageObj(&imgData)
	imgData.Reset()
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
		return
	}

	savePath := "static/upload/" + fileMd5 + ".jpg"
	dstImg := util.ImageResize(img, 1024, 0)

	type response struct {
		normalRsp
		Url string `json:"url"`
	}

	rsp := response{}

	db := h.App.Db
	scf := h.App.Cf.Site

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

		var newImgData bytes.Buffer
		err = jpeg.Encode(&newImgData, dstImg, &jpeg.Options{Quality: 95})
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}
		err = formUploader.Put(context.Background(), &ret, upToken, savePath, bytes.NewReader(newImgData.Bytes()), int64(newImgData.Len()), nil)
		newImgData.Reset()
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

		var newImgData bytes.Buffer
		err = jpeg.Encode(&newImgData, dstImg, &jpeg.Options{Quality: 95})
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"` + err.Error() + `"}`))
			return
		}

		upy := upyun.NewUpYun(scf.UpyunBucket, scf.UpyunUser, scf.UpyunPw)
		err = upy.WriteFile(savePath, "", true, newImgData.Bytes())
		newImgData.Reset()
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

		err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
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

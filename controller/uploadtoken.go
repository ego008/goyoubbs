package controller

import (
	"encoding/json"
	"net/http"

	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"github.com/rs/xid"
)

func (h *BaseHandler) GetUploadImgToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	scf := h.App.Cf.Site

	qmac := qbox.NewMac(scf.QiniuAccessKey, scf.QiniuSecretKey)

	fileSaveName := "youbbs/upload/" + xid.New().String() + ".jpg"
	policy := storage.PutPolicy{
		Scope:   scf.QiniuBucket,
		SaveKey: fileSaveName,
	}
	token := policy.UploadToken(qmac)

	rsp := struct {
		normalRsp
		Domain   string `json:"domain"`
		Token    string `json:"token"`
		UpToken  string `json:"uptoken"`
		Filepath string `json:"filepath"`
		Fullurl  string `json:"fullurl"`
	}{}

	rsp.Retcode = 200
	rsp.Domain = scf.QiniuDomain
	rsp.Token = token
	rsp.UpToken = token
	rsp.Filepath = fileSaveName
	rsp.Fullurl = scf.QiniuDomain + "/" + fileSaveName

	json.NewEncoder(w).Encode(rsp)
}

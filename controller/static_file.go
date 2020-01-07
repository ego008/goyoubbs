package controller

import (
	"goji.io/pat"
	"io/ioutil"
	"net/http"
)

func (h *BaseHandler) StaticFile(w http.ResponseWriter, r *http.Request) {
	filePath := pat.Param(r, "filepath")
	buf, err := ioutil.ReadFile("static/" + filePath)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	fileType := http.DetectContentType(buf)
	w.Header().Set("Content-Type", fileType)
	_, _ = w.Write(buf)
}

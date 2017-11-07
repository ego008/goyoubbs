package controller

import (
	"io/ioutil"
	"net/http"
)

var defaultRobots = `User-agent: *
Disallow: /login
Disallow: /logout
Disallow: /admin
`

func (h *BaseHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	buf, err := ioutil.ReadFile("static/robots.txt")
	if err != nil {
		w.Write([]byte(defaultRobots))
		return
	}
	w.Write(buf)
}

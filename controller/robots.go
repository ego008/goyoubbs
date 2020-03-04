package controller

import (
	"bytes"
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
		_, _ = w.Write([]byte(defaultRobots + "\nSitemap: " + h.App.Cf.Site.MainDomain + "/sitemap.xml"))
		return
	}
	if !bytes.Contains(buf, []byte("Sitemap:")) {
		buf = append(buf, []byte("\nSitemap: "+h.App.Cf.Site.MainDomain+"/sitemap.xml")...)
	}
	_, _ = w.Write(buf)
}

package controller

import (
	"bytes"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var defaultRobots = `User-agent: *
Disallow: /admin
`

func (h *BaseHandler) Robots(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	buf, err := os.ReadFile("static/robots.txt")
	if err != nil {
		c.String(200, defaultRobots+"\nSitemap: "+h.App.Cf.Site.MainDomain+"/sitemap.xml")
		return
	}
	if !bytes.Contains(buf, []byte("Sitemap:")) {
		buf = append(buf, []byte("\nSitemap: "+h.App.Cf.Site.MainDomain+"/sitemap.xml")...)
	}
	c.String(200, string(buf))
}

func (h *BaseHandler) Ads(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	buf, err := os.ReadFile("static/ads.txt")
	if err != nil {
		c.Status(http.StatusNotFound)
		c.String(200, "404: not found")
		return
	}
	c.String(200, string(buf))
}

func (h *BaseHandler) StaticFile(c *gin.Context) {
	filePath := c.Param("filepath")
	buf, err := os.ReadFile("static/" + filePath)
	if err != nil {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Status(http.StatusNotFound)
		c.String(200, err.Error())
		return
	}
	fileType := http.DetectContentType(buf)
	//c.Header("Content-Type", fileType)
	//c.String(200, string(buf))
	c.Data(200, fileType, buf)
}

func (h *BaseHandler) ShowIcon(c *gin.Context) {
	c.Header("Content-Type", "image/png")
	buf, err := os.ReadFile("static/logo.png")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.String(200, string(buf))
}

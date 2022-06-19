package controller

import (
	"errors"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/app"
	"goyoubbs/model"
	"net"
	"strings"
	"time"
)

var rangeTopicLst []model.TopicLi // 边栏显示最近被浏览的文章

type (
	BaseHandler struct {
		App *app.Application
	}
)

func ReadUserIP(ctx *fasthttp.RequestCtx) string {
	clientIp := string(ctx.Request.Header.Peek("X-Real-Ip"))
	netIP := net.ParseIP(clientIp)
	if netIP != nil {
		return clientIp
	}

	ips := string(ctx.Request.Header.Peek(fasthttp.HeaderXForwardedFor))
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP = net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	ips = string(ctx.Request.Header.Peek("X-FORWARDED-FOR"))
	splitIps = strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP = net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String())
	if err != nil {
		return ""
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	return ""
}

func (h *BaseHandler) CurrentUser(ctx *fasthttp.RequestCtx) (model.User, error) {
	var user model.User
	ssValue := h.GetCookie(ctx, "SessionID")
	if len(ssValue) == 0 {
		return user, errors.New("SessionID cookie not found ")
	}
	z := strings.Split(ssValue, ":")
	uid := z[0]
	sessionID := z[1]

	rs := h.App.Db.Hget(model.UserTbName, sdb.DS2b(uid))
	if rs.State == "ok" {
		_ = json.Unmarshal(rs.Data[0], &user)
		if sessionID == user.Session {
			_ = h.SetCookie(ctx, "SessionID", ssValue, 365)
			return user, nil
		}
	}

	return user, errors.New("user not found")
}

func (h *BaseHandler) SetCookie(ctx *fasthttp.RequestCtx, name, value string, days int) error {
	encoded, err := h.App.Sc.Encode(name, value)
	if err != nil {
		return err
	}

	var c fasthttp.Cookie
	c.SetKey(name)
	c.SetValue(encoded)
	c.SetPath("/")
	c.SetSecure(false)
	c.SetHTTPOnly(true)
	c.SetExpire(time.Now().UTC().AddDate(0, 0, days))
	ctx.Response.Header.SetCookie(&c)

	return err
}

func (h *BaseHandler) GetCookie(ctx *fasthttp.RequestCtx, name string) string {
	if cookieByte := ctx.Request.Header.Cookie(name); len(cookieByte) > 0 {
		var value string
		if err := h.App.Sc.Decode(name, sdb.B2s(cookieByte), &value); err == nil {
			return value
		}
	}
	return ""
}

func (h *BaseHandler) DelCookie(ctx *fasthttp.RequestCtx, name string) {
	if len(name) > 0 {
		ctx.Response.Header.DelClientCookie(name)
	}
}

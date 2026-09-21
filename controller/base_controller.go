package controller

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"goyoubbs/model"
	"goyoubbs/util"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

var rangeTopicLst []model.TopicLi // 边栏显示最近被浏览的文章

type (
	BaseHandler struct {
		App *model.Application
	}
)

// ReadUserIP 获取客户端 IP 地址
func ReadUserIP(c *gin.Context) string {
	// Gin 自带 c.ClientIP() 已经能很好地处理 X-Forwarded-For 和 X-Real-IP
	// 此处保留原代码的匹配逻辑，若需优先遵循原有自定义规则：
	clientIp := c.GetHeader("X-Real-Ip")
	netIP := net.ParseIP(clientIp)
	if netIP != nil {
		return clientIp
	}

	ips := c.GetHeader("X-Forwarded-For")
	splitIps := util.StringSplit(ips, ",")
	for _, ip := range splitIps {
		netIP = net.ParseIP(strings.TrimSpace(ip))
		if netIP != nil {
			return strings.TrimSpace(ip)
		}
	}

	ips = c.GetHeader("X-FORWARDED-FOR")
	splitIps = util.StringSplit(ips, ",")
	for _, ip := range splitIps {
		netIP = net.ParseIP(strings.TrimSpace(ip))
		if netIP != nil {
			return strings.TrimSpace(ip)
		}
	}

	// 获取 RemoteAddr 中的 IP
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return ""
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	return ""
}

func (h *BaseHandler) CurrentUser(c *gin.Context) (*model.User, error) {
	user := &model.User{}
	ssValue := h.GetCookie(c, "SessionID")
	if len(ssValue) == 0 {
		return user, errors.New("SessionID cookie not found ")
	}
	index := strings.Index(ssValue, ":")
	if index == -1 {
		return user, errors.New("SessionID cookie not found ")
	}
	uId, _ := strconv.ParseUint(ssValue[:index], 10, 64)
	if uId == 0 {
		return user, errors.New("UserID is 0 ")
	}
	var ok bool
	model.UserMapMux.RLock()
	user, ok = model.UserMap[uId]
	model.UserMapMux.RUnlock()
	if !ok {
		// 用户有缓存，很少触发
		_ = h.App.Db.View(func(tx *bbolt.Tx) error {
			_ = h.App.Db.HGetFunc(tx, model.UserTbName, mdb.I2b(uId), func(val []byte) error {
				_ = json.Unmarshal(val, &user)
				if user.ID > 0 {
					model.UserMapMux.Lock()
					model.UserMap[uId] = user
					model.UserMapMux.Unlock()
				}
				return nil
			})
			return nil
		})
	}
	if user != nil {
		_ = h.SetCookie(c, "SessionID", ssValue, 365)
		return user, nil
	}

	return &model.User{}, errors.New("user not found")
}

// 清理 Cookie 名称，去掉冒号等非法字符
func (h *BaseHandler) formatCookieName(name string) string {
	// 例如 ":8082" -> "8082"
	// 最终得到的 Cookie 名称形如 "SessionID8082"
	return name + h.App.Cf.Main.Addr[1:]
}

func (h *BaseHandler) SetCookie(c *gin.Context, name, value string, days int) error {
	cookieName := h.formatCookieName(name)
	encoded, err := h.App.Sc.Encode(cookieName, value)
	if err != nil {
		return err
	}

	maxAge := days * 86400

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})

	return err
}

func (h *BaseHandler) GetCookie(c *gin.Context, name string) string {
	cookieName := h.formatCookieName(name)
	if cookieStr, err := c.Cookie(cookieName); err == nil && len(cookieStr) > 0 {
		var value string
		if err := h.App.Sc.Decode(cookieName, cookieStr, &value); err == nil {
			return value
		}
	}
	return ""
}

func (h *BaseHandler) DelCookie(c *gin.Context, name string) {
	if len(name) > 0 {
		cookieName := h.formatCookieName(name)
		c.SetCookie(cookieName, "", -1, "/", "", false, true)
	}
}

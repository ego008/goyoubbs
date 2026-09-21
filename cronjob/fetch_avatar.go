package cronjob

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"goyoubbs/util"
	"image/jpeg"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/mdb"
	"go.etcd.io/bbolt"
	"golang.org/x/net/proxy"
)

func FetchAvatar(db *mdb.DB, uid uint64, targetUrl, saveFilePath, ua, sock5Str string) (err error) {
	//if _, err := os.Stat(saveFilePath); err == nil {
	//log.Println("saveFilePath exist", saveFilePath)
	// return nil // !important 否则读取不了
	//}

	// use socks5 proxy
	if len(sock5Str) > 0 {
		if strings.Contains(targetUrl, "github") {
			sock5Str = strings.ReplaceAll(sock5Str, "socks5://", "")
		} else {
			sock5Str = ""
		}
	}

	defaultUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36"

	if len(ua) == 0 {
		ua = defaultUA
	}

	bsUrl, host := util.GetDomainFromURL(targetUrl)

	// 1. 创建 HTTP Request
	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		log.Println("NewRequest error:", err)
		return err
	}

	// 2. 设置 Request Headers
	req.Header.Set("Host", host)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", targetUrl)
	req.Header.Set("Origin", bsUrl)
	// 注意：标准库 net/http 默认会自动发送并解压 Accept-Encoding: gzip，不需要手动设置和手动 Gunzip/Inflate

	// 3. 配置 Transport（TLS 跳过校验、超时、代理）
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        12000,
		MaxIdleConnsPerHost: 1000,
		IdleConnTimeout:     time.Minute,
	}

	// 4. 处理 SOCKS5 代理（如果有）
	if len(sock5Str) > 0 {
		dialer, err := proxy.SOCKS5("tcp", sock5Str, nil, proxy.Direct)
		if err != nil {
			log.Println("SOCKS5 proxy error:", err)
			return err
		}

		// 使用闭包包装，兼容标准库 http.Transport.DialContext
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}

	// 5. 创建 Client 并配置重定向（上限 5 次）
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Minute, // 整体超时
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			return nil
		},
	}

	// 6. 执行请求
	res, err := client.Do(req)
	if err != nil {
		log.Println("client.Do error:", err)
		return err
	}
	defer res.Body.Close()

	// 7. 检查 HTTP 状态码
	if res.StatusCode != http.StatusOK {
		log.Println("res.StatusCode:", res.StatusCode)
		return errors.New("StatusCode: " + strconv.Itoa(res.StatusCode))
	}

	// 8. 读取响应 Body（标准库已自动解压 Gzip/Deflate，不需要手动 switch 判断）
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("read body error:", err)
		return err
	}

	// load original image
	img, err := util.GetImageObj(bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	// https://gist.github.com/sergiotapia/7882944#gistcomment-3394951
	b := img.Bounds()
	imgWidth := b.Dx()  //b.Max.X
	imgHeight := b.Dy() // b.Max.Y
	if imgWidth < 48 && imgHeight < 48 {
		log.Println("img len < 48px", imgWidth, imgHeight)
		return nil
	}

	if len(body) < 800 {
		log.Println("nb is small " + strconv.Itoa(len(body)))
		return nil
	}

	// save to db
	dstImg := util.ImageResize(img, 119, 119)
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, dstImg, &jpeg.Options{Quality: 95})
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		return db.HSet(tx, "user_avatar", mdb.I2b(uid), buf.Bytes())
	})

	if err != nil {
		return err
	}
	log.Println("save img to db ok", uid)
	return nil
}

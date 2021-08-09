package cronjob

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"goyoubbs/util"
	"image/jpeg"
	"log"
	"strconv"
	"strings"
	"time"
)

func FetchAvatar(db *sdb.DB, uid uint64, targetUrl, saveFilePath, ua, sock5Str string) (err error) {
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

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	req.SetRequestURI(targetUrl)
	req.Header.SetMethod("GET")
	req.Header.Set("Host", host)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", targetUrl)
	req.Header.Set("Origin", bsUrl)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	httpCli := &fasthttp.Client{
		TLSConfig:                     &tls.Config{InsecureSkipVerify: true},
		NoDefaultUserAgentHeader:      true, // Don't send: User-Agent: fasthttp
		MaxConnsPerHost:               12000,
		ReadBufferSize:                4096, // Make sure to set this big enough that your whole request can be read at once.
		WriteBufferSize:               4096, // Same but for your response.
		ReadTimeout:                   time.Second,
		WriteTimeout:                  time.Second,
		MaxIdleConnDuration:           time.Minute,
		DisableHeaderNamesNormalizing: true, // If you set the case on your headers correctly you can enable this.
	}

	if len(sock5Str) > 0 {
		httpCli.Dial = fasthttpproxy.FasthttpSocksDialer(sock5Str)
	}
	err = httpCli.DoRedirects(req, res, 5)
	if err != nil {
		log.Println(err)
		return err
	}
	if res.StatusCode() != fasthttp.StatusOK {
		log.Println("res.StatusCode()", res.StatusCode())
		return errors.New("StatusCode: " + strconv.Itoa(res.StatusCode()))
	}

	var body []byte
	switch string(res.Header.Peek("Content-Encoding")) {
	case "gzip":
		body, err = res.BodyGunzip()
		if err != nil {
			log.Println("#gzip", err)
			return err
		}
	case "deflate":
		body, err = res.BodyInflate()
		if err != nil {
			log.Println("#deflate", err)
			return err
		}
	default:
		body = res.Body()
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
	err = db.Hset("user_avatar", sdb.I2b(uid), buf.Bytes())
	if err != nil {
		return err
	}
	log.Println("save img to db ok", uid)
	return nil

	// save to local
	/*
		err = ioutil.WriteFile(saveFilePath, body, 0644)
		if err != nil {
			log.Println("WriteFile err", err)
			return err
		}

		// resize

		dstImg := util.ImageResize(img, 119, 119)

		var f3 *os.File
		f3, err = os.Create(saveFilePath)
		if err != nil {
			log.Println("os.Create err", err)
			return err
		}
		defer func() {
			_ = f3.Close()
		}()

		err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
		if err != nil {
			return err
		}
		log.Println("save img ok", saveFilePath)

		return nil
	*/
}

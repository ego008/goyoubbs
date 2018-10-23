package util

import (
	"errors"
	"github.com/weint/httpclient"
	"io"
	"os"
	"strconv"
	"time"
)

func FetchAvatar(url, save, ua string) error {
	defaultUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36"

	if len(ua) == 0 {
		ua = defaultUA
	}
	if _, err := os.Stat(save); err == nil {
		return nil
	}

	hc := httpclient.NewHttpClientRequest("GET", url)
	hc.SetTimeout(30 * time.Second)
	hc.Header("User-Agent", ua)
	hc.Header("Referer", url)

	resp, err := hc.Response()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("StatusCode " + strconv.Itoa(resp.StatusCode))
	}

	out, err := os.Create(save)
	defer out.Close()

	nb, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	if nb == 800 {
		return errors.New("nb is small " + strconv.FormatInt(nb, 10))
	}

	return nil
}

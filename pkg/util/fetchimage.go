/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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
	defer resp.Body.Close()
	if err != nil {
		return err
	}
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

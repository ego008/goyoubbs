// Copyright 2013-2017 lessgo Author, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var (
	defaultUserAgent = "goHttpClient"
	defaultTimeout   = 60000 * time.Millisecond
)

type HttpClientSignHandler func(*http.Request)

type HttpClientRequest struct {
	Req             *http.Request
	url             string
	timeout         time.Duration
	tlsClientConfig *tls.Config
	rsp             *http.Response
	params          map[string]string
	SignHandler     HttpClientSignHandler
	redirect        func(req *http.Request, via []*http.Request) error
}

func NewHttpClientRequest(method, req_url string) *HttpClientRequest {

	var req http.Request
	req.Method = method
	req.Header = http.Header{}
	req.Header.Set("User-Agent", defaultUserAgent)

	return &HttpClientRequest{
		Req:     &req,
		url:     req_url,
		timeout: defaultTimeout,
		params:  map[string]string{},
	}
}

// SetTimeout sets connect time out.
func (c *HttpClientRequest) SetTimeout(timeout time.Duration) *HttpClientRequest {
	c.timeout = timeout * time.Millisecond
	return c
}

// SetTLSClientConfig sets tls connection configurations if visiting https url.
func (c *HttpClientRequest) SetTLSClientConfig(config *tls.Config) *HttpClientRequest {
	c.tlsClientConfig = config
	return c
}

// Header add header item string in request.
func (c *HttpClientRequest) Header(key, value string) *HttpClientRequest {
	c.Req.Header.Set(key, value)
	return c
}

// SetCookie add cookie into request.
func (c *HttpClientRequest) SetCookie(cookie *http.Cookie) *HttpClientRequest {
	c.Req.Header.Add("Cookie", cookie.String())
	return c
}

// Get returns *HttpClientRequest with GET method.
func Get(url string) *HttpClientRequest {
	return NewHttpClientRequest("GET", url)
}

// Post returns *HttpClientRequest with POST method.
func Post(url string) *HttpClientRequest {
	return NewHttpClientRequest("POST", url)
}

// Put returns *HttpClientRequest with PUT method.
func Put(url string) *HttpClientRequest {
	return NewHttpClientRequest("PUT", url)
}

// Delete returns *HttpClientRequest with DELETE method.
func Delete(url string) *HttpClientRequest {
	return NewHttpClientRequest("DELETE", url)
}

// Head returns *HttpClientRequest with HEAD method.
func Head(url string) *HttpClientRequest {
	return NewHttpClientRequest("HEAD", url)
}

func (c *HttpClientRequest) SetRedirectHandler(fn func(req *http.Request, via []*http.Request) error) {
	c.redirect = fn
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (c *HttpClientRequest) Param(key, value string) *HttpClientRequest {
	c.params[key] = value
	return c
}

// Params adds query params from map in to request.
// params build query string as ?key1=value1&key2=value2...
func (c *HttpClientRequest) Params(params map[string]string) *HttpClientRequest {
	for k, v := range params {
		c.params[k] = v
	}
	return c
}

// Params adds form from map in to request.
func (c *HttpClientRequest) Payload(data map[string]string) *HttpClientRequest {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}
	c.Body(form.Encode())
	return c
}

// Json adds body from interface in to request.
func (c *HttpClientRequest) Json(data interface{}) *HttpClientRequest {
	buf, err := json.Marshal(data)
	if err != nil {
		return c
	}
	c.Body(buf)
	return c
}

// Body adds request raw body.
// it supports string and []byte.
func (c *HttpClientRequest) Body(data interface{}) *HttpClientRequest {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		c.Req.Body = ioutil.NopCloser(bf)
		c.Req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		c.Req.Body = ioutil.NopCloser(bf)
		c.Req.ContentLength = int64(len(t))
	}
	return c
}

// Response executes request client gets response mannually.
func (c *HttpClientRequest) Response() (*http.Response, error) {

	if c.rsp != nil {
		return c.rsp, nil
	}

	var paramBody string
	if len(c.params) > 0 {
		var buf bytes.Buffer
		for k, v := range c.params {
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
			buf.WriteByte('&')
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}
	if c.Req.Method == "GET" && len(paramBody) > 0 {
		if strings.Index(c.url, "?") != -1 {
			c.url += "&" + paramBody
		} else {
			c.url = c.url + "?" + paramBody
		}
	} else if (c.Req.Method == "POST" || c.Req.Method == "PUT") &&
		c.Req.Body == nil && len(paramBody) > 0 {
		c.Header("Content-Type", "application/x-www-form-urlencoded")
		c.Body(paramBody)
	}

	url, err := url.Parse(c.url)
	if err != nil {
		return nil, err
	}
	url.Path = filepath.Clean(url.Path)
	c.Req.URL = url

	if c.SignHandler != nil {
		c.SignHandler(c.Req)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: c.tlsClientConfig,
			Dial:            timeoutDialer(c.timeout, c.timeout),
		},
	}
	if c.redirect != nil {
		client.CheckRedirect = c.redirect
	}
	c.rsp, err = client.Do(c.Req)
	if err != nil {
		return nil, err
	}

	return c.rsp, nil
}

func (c *HttpClientRequest) Status() int {

	if c.rsp != nil {
		return c.rsp.StatusCode
	}

	return 0
}

// Close close the TCP connection, the caller MUST use Close when done reading from it.
func (c *HttpClientRequest) Close() {
	if c.rsp != nil && c.rsp.Body != nil {
		c.rsp.Body.Close()
	}
}

// timeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
func timeoutDialer(cTimeout, rwTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

// ReplyBytes returns the body []byte in response.
// it calls Response inner.
func (c *HttpClientRequest) ReplyBytes() ([]byte, error) {
	resp, err := c.Response()
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ReplyString returns the body string in response.
// it calls Response inner.
func (c *HttpClientRequest) ReplyString() (string, error) {

	data, err := c.ReplyBytes()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ReplyJson returns the map that marshals from the body bytes as json in response .
func (c *HttpClientRequest) ReplyJson(v interface{}) error {

	data, err := c.ReplyBytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &v)
}

func (c *HttpClientRequest) ReplyHeader(key string) string {
	return c.rsp.Header.Get(key)
}

// Copyright (c) 2015 The Httpgzip Authors.
// Use of this source code is governed by an Expat-style
// MIT license that can be found in the LICENSE file.

// Package httpgzip implements an http.Handler wrapper adding gzip
// compression for appropriate requests.
//
// It attempts to properly parse the request's Accept-Encoding header
// according to RFC 2616 and does not do a simple string search for
// "gzip" (which will fail to do the correct thing for values such as
// "*" or "identity,gzip;q=0"). It will serve either gzip or identity
// content codings (identity meaning no encoding), or return 406 Not
// Acceptable status if it can do neither.
//
// It works correctly with handlers which honour Range request headers
// (such as http.FileServer) by removing the Range header for requests
// which prefer gzip encoding. This is necessary since Range requests
// apply to the gzipped content but the wrapped handler is not aware
// of the compression when it writes byte ranges. The Accept-Ranges
// header is also stripped from corresponding responses.
//
// For requests which prefer gzip encoding a Content-Type header is
// set using http.DetectContentType if it is not set by the wrapped
// handler.
//
// Gzip implementation
//
// By default, httpgzip uses the standard library gzip
// implementation. To use the optimized gzip implementation from
// https://github.com/klauspost/compress instead, download and install
// httpgzip with the "kpgzip" build tag:
//
//     go get -tags kpgzip github.com/xi2/httpgzip
//
// or simply alter the import line in httpgzip.go.
//
// Thanks
//
// Thanks are due to Klaus Post for his blog post which inspired the
// creation of this package and is recommended reading:
//
//     https://blog.klauspost.com/gzip-performance-for-go-webservers/
package httpgzip

import (
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/xi2/httpgzip/internal/gzip"
)

// These constants are copied from the gzip package, so that code that
// imports this package does not also have to import the gzip package.
const (
	NoCompression      = gzip.NoCompression
	BestSpeed          = gzip.BestSpeed
	BestCompression    = gzip.BestCompression
	DefaultCompression = gzip.DefaultCompression
)

// DefaultContentTypes is the default list of content types for which
// a Handler considers gzip compression. This list originates from the
// file compression.conf within the Apache configuration found at
// https://html5boilerplate.com/.
var DefaultContentTypes = []string{
	"application/atom+xml",
	"application/font-sfnt",
	"application/javascript",
	"application/json",
	"application/ld+json",
	"application/manifest+json",
	"application/rdf+xml",
	"application/rss+xml",
	"application/schema+json",
	"application/vnd.geo+json",
	"application/vnd.ms-fontobject",
	"application/x-font-ttf",
	"application/x-javascript",
	"application/x-web-app-manifest+json",
	"application/xhtml+xml",
	"application/xml",
	"font/eot",
	"font/opentype",
	"image/bmp",
	"image/svg+xml",
	"image/vnd.microsoft.icon",
	"image/x-icon",
	"text/cache-manifest",
	"text/css",
	"text/html",
	"text/javascript",
	"text/plain",
	"text/vcard",
	"text/vnd.rim.location.xloc",
	"text/vtt",
	"text/x-component",
	"text/x-cross-domain-policy",
	"text/xml",
}

var gzipWriterPools = map[int]*sync.Pool{}

func init() {
	levels := map[int]struct{}{
		DefaultCompression: struct{}{},
		NoCompression:      struct{}{},
	}
	for i := BestSpeed; i <= BestCompression; i++ {
		levels[i] = struct{}{}
	}
	for k := range levels {
		level := k // create new variable for closure
		gzipWriterPools[level] = &sync.Pool{
			New: func() interface{} {
				w, _ := gzip.NewWriterLevel(nil, level)
				return w
			},
		}
	}
}

var gzipBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// A gzipResponseWriter is a modified http.ResponseWriter. It adds
// gzip compression to certain responses, and there are two cases
// where this is done. Case 1 is when encs only allows gzip encoding
// and forbids identity. Case 2 is when encs prefers gzip encoding,
// the response is at least 512 bytes and the response's content type
// is in ctMap.
//
// A gzipResponseWriter sets the Content-Encoding and Content-Type
// headers when appropriate. It is important to call the Close method
// when writing is finished in order to flush and close the
// gzipResponseWriter. The slice encs must contain only encodings from
// {encGzip,encIdentity} and contain at least one encoding.
//
// If a gzip.Writer is used in order to write a response it will use a
// compression level of level.
type gzipResponseWriter struct {
	http.ResponseWriter
	httpStatus int
	ctMap      map[string]struct{}
	encs       []encoding
	level      int
	gw         *gzip.Writer
	buf        *bytes.Buffer
}

func newGzipResponseWriter(w http.ResponseWriter, ctMap map[string]struct{}, encs []encoding, level int) *gzipResponseWriter {
	buf := gzipBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return &gzipResponseWriter{
		ResponseWriter: w,
		httpStatus:     http.StatusOK,
		ctMap:          ctMap,
		encs:           encs,
		level:          level,
		buf:            buf}
}

// init gets called by Write once at least 512 bytes have been written
// to the temporary buffer buf, or by Close if it has not yet been
// called. Firstly it determines the content type, either from the
// Content-Type header, or by calling http.DetectContentType on
// buf. Then, if needed, a gzip.Writer is initialized. Lastly,
// appropriate headers are set and the ResponseWriter's WriteHeader
// method is called.
func (w *gzipResponseWriter) init() {
	cth := w.Header().Get("Content-Type")
	var ct string
	if cth != "" {
		ct = cth
	} else {
		ct = http.DetectContentType(w.buf.Bytes())
	}
	var gzipContentType bool
	if mt, _, err := mime.ParseMediaType(ct); err == nil {
		if _, ok := w.ctMap[mt]; ok {
			gzipContentType = true
		}
	}
	var useGzip bool
	if w.Header().Get("Content-Encoding") == "" && w.encs[0] == encGzip {
		if gzipContentType && w.buf.Len() >= 512 || len(w.encs) == 1 {
			useGzip = true
		}
	}
	if useGzip {
		w.gw = gzipWriterPools[w.level].Get().(*gzip.Writer)
		w.gw.Reset(w.ResponseWriter)
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.Header().Del("Accept-Ranges")
	if cth == "" {
		w.Header().Set("Content-Type", ct)
	}
	w.ResponseWriter.WriteHeader(w.httpStatus)
}

func (w *gzipResponseWriter) Write(p []byte) (int, error) {
	var n, written int
	var err error
	if w.buf != nil {
		written = w.buf.Len()
		_, _ = w.buf.Write(p)
		if w.buf.Len() < 512 {
			return len(p), nil
		}
		w.init()
		p = w.buf.Bytes()
		defer func() {
			gzipBufPool.Put(w.buf)
			w.buf = nil
		}()
	}
	switch {
	case w.gw != nil:
		n, err = w.gw.Write(p)
	default:
		n, err = w.ResponseWriter.Write(p)
	}
	n -= written
	if n < 0 {
		n = 0
	}
	return n, err
}

func (w *gzipResponseWriter) WriteHeader(httpStatus int) {
	// postpone WriteHeader call until end of init method
	w.httpStatus = httpStatus
}

func (w *gzipResponseWriter) Close() (err error) {
	if w.buf != nil {
		w.init()
		p := w.buf.Bytes()
		defer func() {
			gzipBufPool.Put(w.buf)
			w.buf = nil
		}()
		switch {
		case w.gw != nil:
			_, err = w.gw.Write(p)
		default:
			_, err = w.ResponseWriter.Write(p)
		}
	}
	if w.gw != nil {
		e := w.gw.Close()
		if e != nil && err == nil {
			err = e
		}
		gzipWriterPools[w.level].Put(w.gw)
		w.gw = nil
	}
	return err
}

// An encoding is a supported content coding.
type encoding int

const (
	encIdentity encoding = iota
	encGzip
)

// acceptedEncodings returns the supported content codings that are
// accepted by the request r. It returns a slice of encodings in
// client preference order.
//
// If the Sec-WebSocket-Key header is present then compressed content
// encodings are not considered.
//
// ref: http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
func acceptedEncodings(r *http.Request) []encoding {
	h := r.Header.Get("Accept-Encoding")
	swk := r.Header.Get("Sec-WebSocket-Key")
	if h == "" {
		return []encoding{encIdentity}
	}
	gzip := float64(-1)    // -1 means not accepted, 0 -> 1 means value of q
	identity := float64(0) // -1 means not accepted, 0 -> 1 means value of q
	for _, s := range strings.Split(h, ",") {
		f := strings.Split(s, ";")
		f0 := strings.ToLower(strings.Trim(f[0], " "))
		q := float64(1.0)
		if len(f) > 1 {
			f1 := strings.ToLower(strings.Trim(f[1], " "))
			if strings.HasPrefix(f1, "q=") {
				if flt, err := strconv.ParseFloat(f1[2:], 32); err == nil {
					if flt >= 0 && flt <= 1 {
						q = flt
					}
				}
			}
		}
		if (f0 == "gzip" || f0 == "*") && q > gzip && swk == "" {
			gzip = q
		}
		if (f0 == "gzip" || f0 == "*") && q == 0 {
			gzip = -1
		}
		if (f0 == "identity" || f0 == "*") && q > identity {
			identity = q
		}
		if (f0 == "identity" || f0 == "*") && q == 0 {
			identity = -1
		}
	}
	switch {
	case gzip == -1 && identity == -1:
		return []encoding{}
	case gzip == -1:
		return []encoding{encIdentity}
	case identity == -1:
		return []encoding{encGzip}
	case identity > gzip:
		return []encoding{encIdentity, encGzip}
	default:
		return []encoding{encGzip, encIdentity}
	}
}

// NewHandler returns a new http.Handler which wraps a handler h
// adding gzip compression to certain responses. There are two cases
// where gzip compression is done. Case 1 is responses whose requests
// only allow gzip encoding and forbid identity encoding (identity
// encoding meaning no encoding). Case 2 is responses whose requests
// prefer gzip encoding, whose size is at least 512 bytes and whose
// content types are in contentTypes. If contentTypes is nil then
// DefaultContentTypes is considered instead.
//
// The new http.Handler sets the Content-Encoding, Vary and
// Content-Type headers in its responses as appropriate. If a request
// expresses a preference for gzip encoding then any Range headers are
// removed from the request before it is passed through to h and
// Accept-Ranges headers are stripped from corresponding
// responses. This happens regardless of whether gzip encoding is
// eventually used in the response or not.
func NewHandler(h http.Handler, contentTypes []string) http.Handler {
	gzh, _ := NewHandlerLevel(h, contentTypes, DefaultCompression)
	return gzh
}

// NewHandlerLevel is like NewHandler but allows one to specify the
// gzip compression level instead of assuming DefaultCompression.
//
// The compression level can be DefaultCompression, NoCompression, or
// any integer value between BestSpeed and BestCompression
// inclusive. The error returned will be nil if the level is valid.
func NewHandlerLevel(h http.Handler, contentTypes []string, level int) (http.Handler, error) {
	switch {
	case level == DefaultCompression || level == NoCompression:
		// no action needed
	case level < BestSpeed || level > BestCompression:
		return nil, fmt.Errorf(
			"httpgzip: invalid compression level: %d", level)
	}
	if contentTypes == nil {
		contentTypes = DefaultContentTypes
	}
	ctMap := map[string]struct{}{}
	for _, ct := range contentTypes {
		ctMap[ct] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// add Vary header
		w.Header().Add("Vary", "Accept-Encoding")
		// check client's accepted encodings
		encs := acceptedEncodings(r)
		// return if no acceptable encodings
		if len(encs) == 0 {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		if encs[0] == encGzip {
			// cannot accept Range requests for possibly gzipped
			// responses
			r.Header.Del("Range")
			// create new ResponseWriter
			w = newGzipResponseWriter(w, ctMap, encs, level)
			defer w.(*gzipResponseWriter).Close()
		}
		// call original handler's ServeHTTP
		h.ServeHTTP(w, r)
	}), nil
}

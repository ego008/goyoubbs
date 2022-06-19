package util

import (
	"bytes"
	"encoding/xml"
	"github.com/ego008/goutils/json"
	"github.com/valyala/fasthttp"
)

var (
	// Charset is the default content type charset for Request Processors .
	Charset = "utf-8"

	// JSON implements the full `Processor` interface.
	// It is responsible to dispatch JSON results to the client and to read JSON
	// data from the request body.
	//
	// Usage:
	// To read from a request:
	// util.Bind(ctx, util.JSON, &myStructValue)
	// To send a response:
	// muxie.Dispatch(w, muxie.JSON, mySendDataValue)
	JSON = &jsonProcessor{Prefix: nil, Indent: "", UnescapeHTML: false}

	// XML implements the full `Processor` interface.
	// It is responsible to dispatch XML results to the client and to read XML
	// data from the request body.
	//
	// Usage:
	// To read from a request:
	// muxie.Bind(r, muxie.XML, &myStructValue)
	// To send a response:
	// muxie.Dispatch(w, muxie.XML, mySendDataValue)
	XML = &xmlProcessor{Indent: ""}
)

func withCharset(cType string) string {
	return cType + "; charset=" + Charset
}

// Binder is the interface which `muxie.Bind` expects.
// It is used to bind a request to a go struct value (ptr).
type Binder interface {
	Bind(*fasthttp.RequestCtx, interface{}) error
}

// Bind accepts the current request and any `Binder` to bind
// the request data to the "ptrOut".
func Bind(ctx *fasthttp.RequestCtx, b Binder, ptrOut interface{}) error {
	return b.Bind(ctx, ptrOut)
}

// Dispatcher is the interface which `muxie.Dispatch` expects.
// It is used to send a response based on a go struct value.
type Dispatcher interface {
	// no io.Writer because we need to set the headers here,
	// Binder and Processor are only for HTTP.
	Dispatch(*fasthttp.RequestCtx, interface{}) error
	// http.Response
}

// Dispatch accepts the current response writer and any `Dispatcher`
// to send the "v" to the client.
func Dispatch(ctx *fasthttp.RequestCtx, d Dispatcher, v interface{}) error {
	return d.Dispatch(ctx, v)
}

// Processor implements both `Binder` and `Dispatcher` interfaces.
// It is used for implementations that can `Bind` and `Dispatch`
// the same data form.
//
// Look `JSON` and `XML` for more.
type Processor interface {
	Binder
	Dispatcher
}

var (
	newLineB byte = '\n'
	// the html codes for unescaping
	ltHex = []byte("\\u003c")
	lt    = []byte("<")

	gtHex = []byte("\\u003e")
	gt    = []byte(">")

	andHex = []byte("\\u0026")
	and    = []byte("&")
)

type jsonProcessor struct {
	Prefix       []byte
	Indent       string
	UnescapeHTML bool
}

var _ Processor = (*jsonProcessor)(nil)

func (p *jsonProcessor) Bind(ctx *fasthttp.RequestCtx, v interface{}) error {
	return json.Unmarshal(ctx.PostBody(), v)
}

func (p *jsonProcessor) Dispatch(ctx *fasthttp.RequestCtx, v interface{}) error {
	var (
		result []byte
		err    error
	)

	if indent := p.Indent; indent != "" {
		marshalIndent := json.MarshalIndent

		result, err = marshalIndent(v, "", indent)
		result = append(result, newLineB)
	} else {
		marshal := json.Marshal
		result, err = marshal(v)
	}

	if err != nil {
		return err
	}

	if p.UnescapeHTML {
		result = bytes.Replace(result, ltHex, lt, -1)
		result = bytes.Replace(result, gtHex, gt, -1)
		result = bytes.Replace(result, andHex, and, -1)
	}

	if len(p.Prefix) > 0 {
		result = append([]byte(p.Prefix), result...)
	}

	ctx.SetContentType(withCharset("application/json"))
	_, err = ctx.Write(result)
	return err
}

type xmlProcessor struct {
	Indent string
}

var _ Processor = (*xmlProcessor)(nil)

func (p *xmlProcessor) Bind(ctx *fasthttp.RequestCtx, v interface{}) error {
	return xml.Unmarshal(ctx.PostBody(), v)
}

func (p *xmlProcessor) Dispatch(ctx *fasthttp.RequestCtx, v interface{}) error {
	var (
		result []byte
		err    error
	)

	if indent := p.Indent; indent != "" {
		marshalIndent := xml.MarshalIndent

		result, err = marshalIndent(v, "", indent)
		result = append(result, newLineB)
	} else {
		marshal := xml.Marshal
		result, err = marshal(v)
	}

	if err != nil {
		return err
	}

	ctx.SetContentType(withCharset("text/xml"))
	_, err = ctx.Write(result)
	return err
}

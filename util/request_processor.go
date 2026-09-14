package util

import (
	"bytes"
	"encoding/xml"
	"net/http"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
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
	// util.Bind(c, util.JSON, &myStructValue)
	// To send a response:
	// util.Dispatch(c, util.JSON, mySendDataValue)
	JSON = &jsonProcessor{Prefix: nil, Indent: "", UnescapeHTML: false}

	// XML implements the full `Processor` interface.
	// It is responsible to dispatch XML results to the client and to read XML
	// data from the request body.
	//
	// Usage:
	// To read from a request:
	// util.Bind(c, util.XML, &myStructValue)
	// To send a response:
	// util.Dispatch(c, util.XML, mySendDataValue)
	XML = &xmlProcessor{Indent: ""}
)

func withCharset(cType string) string {
	return cType + "; charset=" + Charset
}

// Binder is the interface which `util.Bind` expects.
// It is used to bind a request to a go struct value (ptr).
type Binder interface {
	Bind(*gin.Context, interface{}) error
}

// Bind accepts the current request and any `Binder` to bind
// the request data to the "ptrOut".
func Bind(c *gin.Context, b Binder, ptrOut interface{}) error {
	return b.Bind(c, ptrOut)
}

// Dispatcher is the interface which `util.Dispatch` expects.
// It is used to send a response based on a go struct value.
type Dispatcher interface {
	Dispatch(*gin.Context, interface{}) error
}

// Dispatch accepts the current response writer and any `Dispatcher`
// to send the "v" to the client.
func Dispatch(c *gin.Context, d Dispatcher, v interface{}) error {
	return d.Dispatch(c, v)
}

// Processor implements both `Binder` and `Dispatcher` interfaces.
// It is used for implementations that can `Bind` and `Dispatch`
// the same data form.
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
	Indent       string
	Prefix       []byte
	UnescapeHTML bool
}

var _ Processor = (*jsonProcessor)(nil)

func (p *jsonProcessor) Bind(c *gin.Context, v interface{}) error {
	body, err := c.GetRawData()
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (p *jsonProcessor) Dispatch(c *gin.Context, v interface{}) error {
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

	c.Data(http.StatusOK, withCharset("application/json"), result)
	return nil
}

type xmlProcessor struct {
	Indent string
}

var _ Processor = (*xmlProcessor)(nil)

func (p *xmlProcessor) Bind(c *gin.Context, v interface{}) error {
	body, err := c.GetRawData()
	if err != nil {
		return err
	}
	return xml.Unmarshal(body, v)
}

func (p *xmlProcessor) Dispatch(c *gin.Context, v interface{}) error {
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

	c.Data(http.StatusOK, withCharset("text/xml"), result)
	return nil
}

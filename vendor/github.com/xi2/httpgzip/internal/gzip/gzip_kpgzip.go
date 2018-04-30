// Copyright (c) 2015 The Httpgzip Authors.
// Use of this source code is governed by an Expat-style
// MIT license that can be found in the LICENSE file.

// +build kpgzip

// Package gzip is a partial implementation of the gzip API using the
// github.com/klauspost/compress/gzip package. It contains the part of
// the API used by httpgzip.
package gzip

import (
	"io"

	"github.com/klauspost/compress/gzip"
)

const (
	NoCompression      = gzip.NoCompression
	BestSpeed          = gzip.BestSpeed
	BestCompression    = gzip.BestCompression
	DefaultCompression = gzip.DefaultCompression
)

type Writer gzip.Writer

func NewWriterLevel(w io.Writer, level int) (*Writer, error) {
	z, err := gzip.NewWriterLevel(w, level)
	return (*Writer)(z), err
}

func (z *Writer) Reset(w io.Writer) {
	(*gzip.Writer)(z).Reset(w)
}

func (z *Writer) Write(p []byte) (int, error) {
	return (*gzip.Writer)(z).Write(p)
}

func (z *Writer) Close() error {
	return (*gzip.Writer)(z).Close()
}

package util

import (
	"bytes"
	"errors"
	"github.com/disintegration/imaging"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
)

var imgTable = map[string]string{
	"image/jpeg": "jpg",
	"image/jpg":  "jpg",
	"image/gif":  "gif",
	"image/png":  "png",
}

func CheckImageType(buff []byte) string {
	// why 512 bytes ? see http://golang.org/pkg/net/http/#DetectContentType
	//buff := make([]byte, 512)
	fileType := http.DetectContentType(buff)
	if v, ok := imgTable[fileType]; ok {
		return v
	}
	return ""
}

func GetImageObj(buff *bytes.Buffer) (image.Image, error) {
	var img image.Image
	var err error
	filetype := http.DetectContentType(buff.Bytes()[:512])
	switch filetype {
	case "image/jpeg", "image/jpg":
		img, err = jpeg.Decode(bytes.NewReader(buff.Bytes()))
	case "image/gif":
		img, err = gif.Decode(bytes.NewReader(buff.Bytes()))
	case "image/png":
		img, err = png.Decode(bytes.NewReader(buff.Bytes()))
	default:
		err = errors.New("unknown image format")
	}
	return img, err
}

func ImageResize(srcImg image.Image, w, h int) *image.NRGBA {
	if w > 73 {
		srcW := srcImg.Bounds().Max.X
		srcH := srcImg.Bounds().Max.Y

		if srcW < w {
			w = srcW
		}
		if srcH < h {
			h = srcH
		}
	}
	return imaging.Resize(srcImg, w, h, imaging.Lanczos)
}

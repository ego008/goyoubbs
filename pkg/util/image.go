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
	"bytes"
	"errors"
	"github.com/disintegration/imaging"
	"github.com/o1egl/govatar"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
)

var imgTable = map[string]string{
	"image/jpeg": "jpeg",
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

func GetImageObj2(buff *bytes.Buffer) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(buff.Bytes()))
	return img, err
}

func GetImageObj(buff *bytes.Buffer) (image.Image, error) {
	var img image.Image
	var err error
	fileType := http.DetectContentType(buff.Bytes()[:512])
	switch fileType {
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

func AvatarResize(srcImg image.Image, w, h int, filePath string) error {
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
	dstImg := imaging.Resize(srcImg, w, h, imaging.Lanczos)

	f3, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f3.Close()

	err = jpeg.Encode(f3, dstImg, &jpeg.Options{Quality: 95})
	if err != nil {
		return err
	}
	return nil
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

func GenerateAvatar(sex, userName string, w, h int, filePath string) error {
	var gender govatar.Gender
	if sex == "male" {
		gender = govatar.MALE
	} else {
		gender = govatar.FEMALE
	}
	img, err := govatar.GenerateFromUsername(gender, userName)
	if err != nil {
		return err
	}
	return AvatarResize(img, w, h, filePath)
}

package util

import (
	"bytes"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"

	"github.com/disintegration/imaging"
)

var (
	imgTable = map[string]string{
		"image/jpeg": "jpg",
		"image/jpg":  "jpg",
		"image/gif":  "gif",
		"image/png":  "png",
	}

	mediaTable = map[string]string{
		"audio/mpeg": "mp3",
		"video/mp4":  "mp4",
	}
)

func CheckImageType(buff []byte) string {
	// why 512 bytes ? see http://golang.org/pkg/net/http/#DetectContentType
	//buff := make([]byte, 512)
	fileType := http.DetectContentType(buff)
	if v, ok := imgTable[fileType]; ok {
		return v
	}
	return ""
}

func CheckMediaType(buff []byte) string {
	// why 512 bytes ? see http://golang.org/pkg/net/http/#DetectContentType
	//buff := make([]byte, 512)
	fileType := http.DetectContentType(buff)
	if v, ok := mediaTable[fileType]; ok {
		return v
	}
	if isMP3(buff) {
		return "mp3"
	}
	if isMP4(buff) {
		return "mp4"
	}
	return ""
}

// 检测MP4文件的魔数
func isMP4(buffer []byte) bool {
	if len(buffer) < 8 {
		return false
	}

	// MP4文件通常以ftyp开头
	if string(buffer[4:8]) == "ftyp" {
		return true
	}

	return false
}

// 检测MP3文件的魔数
func isMP3(buffer []byte) bool {
	if len(buffer) < 3 {
		return false
	}

	// MP3文件帧头检测
	// 检查ID3v2标签（以"ID3"开头）
	if len(buffer) >= 3 && string(buffer[0:3]) == "ID3" {
		return true
	}

	// 检查MPEG帧同步位
	if len(buffer) >= 2 {
		// 检查FF FB (MPEG版本1, 层3)
		if buffer[0] == 0xFF && (buffer[1]&0xE0) == 0xE0 {
			// 检查具体的MPEG音频帧
			if (buffer[1]&0x18)>>3 == 0x03 { // MPEG版本1
				if (buffer[1]&0x06)>>1 == 0x01 { // 层3
					return true
				}
			}
		}
	}

	return false
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

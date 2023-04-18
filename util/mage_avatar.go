package util

import (
	"bytes"
	"errors"
	"github.com/nfnt/resize"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
)

type Writer struct {
	bytes.Buffer
}

func MemoryNewWriter(buf []byte) *Writer {
	return &Writer{
		Buffer: *bytes.NewBuffer(buf),
	}
}

func (w Writer) Close() error {
	return nil
}

// thank to https://www.cnblogs.com/cqvoip/p/8078843.html

// The following is an imitation WeChat group avatar

// Merge Jiugongge avatar synthesis fixed new picture size 132*132px gap 3px
// At least one picture
func Merge(src []io.Reader, dst io.Writer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Merge.recover:", r)
		}
	}()
	var err error
	imagePoints := getXy(len(src))
	width := getWidth(len(src))

	//Create a background image
	background := image.NewRGBA(image.Rect(0, 0, 132, 132))

	//Set the background to gray (127, 127, 127)
	for m := 0; m < 132; m++ {
		for n := 0; n < 132; n++ {
			background.SetRGBA(m, n, color.RGBA{R: 214, G: 214, B: 214, A: 0})
		}
	}

	for i, v := range imagePoints {
		x := v.x
		y := v.y

		fOut := MemoryNewWriter(nil)

		_, _, err = Scale(src[i], fOut, width, width, 100)
		if err != nil {
			return err
		}

		//file2draw, err := jpeg.Decode(fOut)
		file2draw, _, err := image.Decode(fOut)
		if err != nil {
			return err
		}
		draw.Draw(background, background.Bounds(), file2draw, file2draw.Bounds().Min.Sub(image.Pt(x, y)), draw.Src)
	}

	return jpeg.Encode(dst, background, nil)
}

type Point struct {
	x, y int
}

func getXy(size int) []*Point {
	s := make([]*Point, size)
	var _x, _y int

	if size == 1 {
		_x, _y = 6, 6
		s[0] = &Point{_x, _y}
	}
	if size == 2 {
		_x, _y = 4, 4
		s[0] = &Point{_x, 132/2 - 60/2}
		s[1] = &Point{60 + 2*_x, 132/2 - 60/2}
	}
	if size == 3 {
		_x, _y = 4, 4
		s[0] = &Point{132/2 - 60/2, _y}
		s[1] = &Point{_x, 60 + 2*_y}
		s[2] = &Point{60 + 2*_y, 60 + 2*_y}
	}
	if size == 4 {
		_x, _y = 4, 4
		s[0] = &Point{_x, _y}
		s[1] = &Point{_x*2 + 60, _y}
		s[2] = &Point{_x, 60 + 2*_y}
		s[3] = &Point{60 + 2*_y, 60 + 2*_y}
	}
	if size == 5 {
		_x, _y = 3, 3
		s[0] = &Point{(132 - 40*2 - _x) / 2, (132 - 40*2 - _y) / 2}
		s[1] = &Point{(132-40*2-_x)/2 + 40 + _x, (132 - 40*2 - _y) / 2}
		s[2] = &Point{_x, (132-40*2-_x)/2 + 40 + _y}
		s[3] = &Point{_x*2 + 40, (132-40*2-_x)/2 + 40 + _y}
		s[4] = &Point{_x*3 + 40*2, (132-40*2-_x)/2 + 40 + _y}
	}
	if size == 6 {
		_x, _y = 3, 3
		s[0] = &Point{_x, (132 - 40*2 - _x) / 2}
		s[1] = &Point{_x*2 + 40, (132 - 40*2 - _x) / 2}
		s[2] = &Point{_x*3 + 40*2, (132 - 40*2 - _x) / 2}
		s[3] = &Point{_x, (132-40*2-_x)/2 + 40 + _y}
		s[4] = &Point{_x*2 + 40, (132-40*2-_x)/2 + 40 + _y}
		s[5] = &Point{_x*3 + 40*2, (132-40*2-_x)/2 + 40 + _y}
	}

	if size == 7 {
		_x, _y = 3, 3
		s[0] = &Point{(132 - 40) / 2, _y}
		s[1] = &Point{_x, _y*2 + 40}
		s[2] = &Point{_x*2 + 40, _y*2 + 40}
		s[3] = &Point{_x*3 + 40*2, _y*2 + 40}
		s[4] = &Point{_x, _y*3 + 40*2}
		s[5] = &Point{_x*2 + 40, _y*3 + 40*2}
		s[6] = &Point{_x*3 + 40*2, _y*3 + 40*2}
	}
	if size == 8 {
		_x, _y = 3, 3
		s[0] = &Point{(132 - 80 - _x) / 2, _y}
		s[1] = &Point{(132-80-_x)/2 + _x + 40, _y}
		s[2] = &Point{_x, _y*2 + 40}
		s[3] = &Point{_x*2 + 40, _y*2 + 40}
		s[4] = &Point{_x*3 + 40*2, _y*2 + 40}
		s[5] = &Point{_x, _y*3 + 40*2}
		s[6] = &Point{_x*2 + 40, _y*3 + 40*2}
		s[7] = &Point{_x*3 + 40*2, _y*3 + 40*2}
	}
	if size == 9 {
		_x, _y = 3, 3
		s[0] = &Point{_x, _y}
		s[1] = &Point{_x*2 + 40, _y}
		s[2] = &Point{_x*3 + 40*2, _y}
		s[3] = &Point{_x, _y*2 + 40}
		s[4] = &Point{_x*2 + 40, _y*2 + 40}
		s[5] = &Point{_x*3 + 40*2, _y*2 + 40}
		s[6] = &Point{_x, _y*3 + 40*2}
		s[7] = &Point{_x*2 + 40, _y*3 + 40*2}
		s[8] = &Point{_x*3 + 40*2, _y*3 + 40*2}
	}
	return s
}

func getWidth(size int) int {
	var width int
	if size == 1 {
		width = 120
	}
	if size > 1 && size <= 4 {
		width = 60
	}
	if size >= 5 {
		width = 40
	}
	return width
}

// Scale
/*
 * Scale thumbnail generation
 * Input parameters: picture input and output, thumbnail width, height, precision
 * Rule: If one of width or height is 0, the size remains unchanged. If the precision is 0, the accuracy remains unchanged.
 * Back: Thumbnail true width, height, error
 */
func Scale(in io.Reader, out io.Writer, width, height, quality int) (int, int, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	var (
		w, h int
	)
	origin, fm, err := image.Decode(in)
	if err != nil {
		log.Println(err)
		return 0, 0, err
	}
	if width == 0 || height == 0 {
		width = origin.Bounds().Max.X
		height = origin.Bounds().Max.Y
	}
	if quality == 0 {
		quality = 100
	}
	canvas := resize.Thumbnail(uint(width), uint(height), origin, resize.Lanczos3)

	//return jpeg.Encode(out, canvas, &jpeg.Options{quality})
	w = canvas.Bounds().Dx()
	h = canvas.Bounds().Dy()
	switch fm {
	case "jpeg":
		return w, h, jpeg.Encode(out, canvas, &jpeg.Options{Quality: quality})
	case "png":
		return w, h, png.Encode(out, canvas)
	case "gif":
		return w, h, gif.Encode(out, canvas, &gif.Options{})
		//case "bmp": //x/image/bmp was commented out by me
	//	return w, h, bmp.Encode(out, canvas)
	default:
		return w, h, errors.New("ERROR FORMAT")
	}
}

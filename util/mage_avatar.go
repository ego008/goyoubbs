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

// Merge6Grid 6 palace grid
// rule NO1: at least 3 pictures, up to 6 pictures
// NO2: the first size 60*60 other size 28*28 at 4px interval composite image size 102*102
// NO3: Arrange order from left to right, top to small, then right to left
// ++|
// ++|
// --|
func Merge6Grid(src []io.Reader, dst io.Writer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Merge.recover:", r)
		}
	}()
	if len(src) < 3 || len(src) > 6 {
		panic("the pic num is between 3 and 6")
	}
	var err error
	imagePoints := getXy6Grid(len(src))

	//Create a large background image
	background := image.NewRGBA(image.Rect(0, 0, 100, 100))
	//Set the background to gray
	for m := 0; m < 100; m++ {
		for n := 0; n < 100; n++ {
			//rgba := GetRGBA(0xC8CED4)
			background.SetRGBA(m, n, color.RGBA{R: 127, G: 127, B: 127, A: 0})
		}
	}

	//Background rectangle with rounded corners
	newBg, err := CreateRoundRectWithRGBA(background, 10)
	if err != nil {
		return err
	}

	//Start synthesis
	var width int
	for i, v := range imagePoints {
		x := v.x
		y := v.y

		if i == 0 {
			width = 60
		} else {
			width = 28
		}
		fOut := MemoryNewWriter(nil)

		//Abbreviate first
		_, _, err = Scale(src[i], fOut, width, width, 100)
		if err != nil {
			return err
		}

		//Round rectangle
		rgba, err := CreateRoundRectWithoutColor(fOut, 6)
		if err != nil {
			return err
		}

		draw.Draw(newBg, newBg.Bounds(), rgba, rgba.Bounds().Min.Sub(image.Pt(x, y)), draw.Src)
	}

	return png.Encode(dst, newBg)
	//return jpeg.Encode(dst, newBg, nil)
}

//CreateRoundRect creates a rounded rectangle r is the input image r is the radius of the corner and color is the color of the corner
func CreateRoundRect(rd io.Reader, r int, c *color.RGBA) (*image.RGBA, error) {
	src, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	x := b.Dx()
	y := b.Dy()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, src.Bounds().Min, draw.Src)

	p1 := image.Point{X: r, Y: r}
	p2 := image.Point{X: x - r, Y: r}
	p3 := image.Point{X: r, Y: y - r}
	p4 := image.Point{X: x - r, Y: y - r}

	for m := 0; m < x; m++ {
		for n := 0; n < y; n++ {
			if (p1.X-m)*(p1.X-m)+(p1.Y-n)*(p1.Y-n) > r*r && m <= p1.X && n <= p1.Y {
				dst.Set(m, n, c)
			} else if (p2.X-m)*(p2.X-m)+(p2.Y-n)*(p2.Y-n) > r*r && m > p2.X && n <= p2.Y {
				dst.Set(m, n, c)
			} else if (p3.X-m)*(p3.X-m)+(p3.Y-n)*(p3.Y-n) > r*r && m <= p3.X && n > p3.Y {
				dst.Set(m, n, c)
			} else if (p4.X-m)*(p4.X-m)+(p4.Y-n)*(p4.Y-n) > r*r && m > p4.X && n > p4.Y {
				dst.Set(m, n, c)
			}
		}
	}
	return dst, nil
}

func CreateRoundRectWithoutColor(rd io.Reader, r int) (*image.RGBA, error) {
	src, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}

	b := src.Bounds()
	x := b.Dx()
	y := b.Dy()
	dst := image.NewRGBA(b)

	p1 := image.Point{X: r, Y: r}
	p2 := image.Point{X: x - r, Y: r}
	p3 := image.Point{X: r, Y: y - r}
	p4 := image.Point{X: x - r, Y: y - r}

	for m := 0; m < x; m++ {
		for n := 0; n < y; n++ {
			if (p1.X-m)*(p1.X-m)+(p1.Y-n)*(p1.Y-n) > r*r && m <= p1.X && n <= p1.Y {
			} else if (p2.X-m)*(p2.X-m)+(p2.Y-n)*(p2.Y-n) > r*r && m > p2.X && n <= p2.Y {
			} else if (p3.X-m)*(p3.X-m)+(p3.Y-n)*(p3.Y-n) > r*r && m <= p3.X && n > p3.Y {
			} else if (p4.X-m)*(p4.X-m)+(p4.Y-n)*(p4.Y-n) > r*r && m > p4.X && n > p4.Y {
			} else {
				dst.Set(m, n, src.At(m, n))
			}
		}
	}
	return dst, nil
}

func CreateRoundRectWithRGBA(src *image.RGBA, r int) (*image.RGBA, error) {
	b := src.Bounds()
	x := b.Dx()
	y := b.Dy()
	dst := image.NewRGBA(b)

	p1 := image.Point{X: r, Y: r}
	p2 := image.Point{X: x - r, Y: r}
	p3 := image.Point{X: r, Y: y - r}
	p4 := image.Point{X: x - r, Y: y - r}

	for m := 0; m < x; m++ {
		for n := 0; n < y; n++ {
			if (p1.X-m)*(p1.X-m)+(p1.Y-n)*(p1.Y-n) > r*r && m <= p1.X && n <= p1.Y {
			} else if (p2.X-m)*(p2.X-m)+(p2.Y-n)*(p2.Y-n) > r*r && m > p2.X && n <= p2.Y {
			} else if (p3.X-m)*(p3.X-m)+(p3.Y-n)*(p3.Y-n) > r*r && m <= p3.X && n > p3.Y {
			} else if (p4.X-m)*(p4.X-m)+(p4.Y-n)*(p4.Y-n) > r*r && m > p4.X && n > p4.Y {
			} else {
				dst.Set(m, n, src.At(m, n))
			}
		}
	}
	return dst, nil
}

// GetRGBA c
func GetRGBA(c int) *color.RGBA {
	var cl color.RGBA
	cl.R = uint8((c >> 16) & 0xFF)
	cl.G = uint8((c >> 8) & 0xFF)
	cl.B = uint8(c & 0xFF)
	return &cl
}

func getXy6Grid(size int) []*Point {
	s := make([]*Point, size)
	_x, _y := 4, 4
	s[0] = &Point{_x, _y}
	s[1] = &Point{60 + 2*_x, _y}
	s[2] = &Point{60 + 2*_x, 28 + 2*_y}
	if size >= 4 {
		s[3] = &Point{60 + 2*_x, 28*2 + 3*_y}
	}
	if size >= 5 {
		s[4] = &Point{28 + 2*_x, 60 + 2*_y}
	}
	if size >= 6 {
		s[5] = &Point{_x, 60 + 2*_y}
	}

	return s
}

// The following is an imitation WeChat group avatar

// Merge Jiugongge avatar synthesis fixed new picture size 132*132px gap 3px
//At least one picture
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

// Clip
//Make up the missing code
//* Clip Picture crop
//* Input parameters: image input, output, thumbnail width, thumbnail height, Rectangle{Pt(x0, y0), Pt(x1, y1)}, precision
//* Rule: If the precision is 0, the precision remains unchanged
//*
//* returns: error
// */
func Clip(in io.Reader, out io.Writer, wi, hi, x0, y0, x1, y1, quality int) (err error) {
	err = errors.New("unknown error")
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	var origin image.Image
	var fm string
	origin, fm, err = image.Decode(in)
	if err != nil {
		log.Println(err)
		return err
	}

	if wi == 0 || hi == 0 {
		wi = origin.Bounds().Max.X
		hi = origin.Bounds().Max.Y
	}
	var canvas image.Image
	if wi != origin.Bounds().Max.X {
		//Abbreviate first
		canvas = resize.Thumbnail(uint(wi), uint(hi), origin, resize.Lanczos3)
	} else {
		canvas = origin
	}

	switch fm {
	case "jpeg":
		img := canvas.(*image.YCbCr)
		subImg := img.SubImage(image.Rect(x0, y0, x1, y1)).(*image.YCbCr)
		return jpeg.Encode(out, subImg, &jpeg.Options{Quality: quality})
	case "png":
		switch canvas.(type) {
		case *image.NRGBA:
			img := canvas.(*image.NRGBA)
			subImg := img.SubImage(image.Rect(x0, y0, x1, y1)).(*image.NRGBA)
			return png.Encode(out, subImg)
		case *image.RGBA:
			img := canvas.(*image.RGBA)
			subImg := img.SubImage(image.Rect(x0, y0, x1, y1)).(*image.RGBA)
			return png.Encode(out, subImg)
		}
	case "gif":
		img := canvas.(*image.Paletted)
		subImg := img.SubImage(image.Rect(x0, y0, x1, y1)).(*image.Paletted)
		return gif.Encode(out, subImg, &gif.Options{})
	//case "bmp":
	//	img := canvas.(*image.RGBA)
	//	subImg := img.SubImage(image.Rect(x0, y0, x1, y1)).(*image.RGBA)
	//	return bmp.Encode(out, subImg)
	default:
		return errors.New("ERROR FORMAT")
	}
	return nil
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

// Format2JPEG converts the picture to JPEG format
func Format2JPEG(in io.Reader, out io.Writer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	origin, _, err := image.Decode(in)
	if err != nil {
		return err
	}
	return jpeg.Encode(out, origin, nil)
}

// Format2PNG converts the picture to PNG format
func Format2PNG(in io.Reader, out io.Writer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	origin, _, err := image.Decode(in)
	if err != nil {
		return err
	}
	return png.Encode(out, origin)
}

// Format2GIF converts the picture to GIF format
func Format2GIF(in io.Reader, out io.Writer) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()
	origin, _, err := image.Decode(in)
	if err != nil {
		return err
	}
	return gif.Encode(out, origin, nil)
}

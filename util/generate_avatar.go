package util

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"github.com/ego008/sdb"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/big"
	"os"
	"regexp"
	"strings"
)

const (
	fontFile = "static/fonts/default.ttf"
)

var (
	imgWidth  = 112
	imgHeight = 112
	fontSize  = math.Min(float64(imgWidth), float64(imgHeight)) * 0.6
)

func GenAvatar(db *sdb.DB, uid uint64, text string) error {

	upLeft := image.Point{X: 0, Y: 0}
	lowRight := image.Point{X: imgWidth, Y: imgHeight}

	dst := image.NewRGBA(image.Rectangle{Min: upLeft, Max: lowRight})

	// Colors are defined by Red, Green, Blue, Alpha uint8 values.
	bgColor := str2rgb(text) //color.RGBA{R: 243, G: 180, B: 121, A: 0xff}

	// 画背景
	// Set color for each pixel.
	for x := 0; x < imgWidth; x++ {
		for y := 0; y < imgHeight; y++ {
			dst.Set(x, y, bgColor)
		}
	}

	c := freetype.NewContext()

	// 设置屏幕每英寸的分辨率
	c.SetDPI(100)

	// 背景
	c.SetClip(dst.Bounds())
	// 设置目标图像
	c.SetDst(dst)
	c.SetHinting(font.HintingFull)

	// 设置文字颜色、字体、字大小
	fontColor := generateFontColorByBgColor(bgColor)
	//log.Println("fontColor", fontColor)
	c.SetSrc(image.NewUniform(fontColor))
	// 以磅为单位设置字体大小
	c.SetFontSize(fontSize)
	fontFam, err := getFontFamily()
	if err != nil {
		log.Println("get font family error", err)
		return err
	}
	// 设置用于绘制文本的字体
	c.SetFont(fontFam)

	var drawStr string
	if len(text) > 1 {
		drawStr = string([]rune(text)[:1])
	} else {
		drawStr = text
	}

	var getFontWidth = func() int {
		return int(c.PointToFixed(fontSize) >> 6)
	}
	var calculateCenterLocation = func(s string) (x int, y int) {
		cr := regexp.MustCompile("[\u4e00-\u9FA5]{1}")
		er := regexp.MustCompile("[a-zA-Z]{1}")
		nr := regexp.MustCompile("[0-9]{1}")

		cCount := len(cr.FindAllString(s, -1))
		eCount := len(er.FindAllString(s, -1))
		nCount := len(nr.FindAllString(s, -1))

		fw := getFontWidth()
		x = imgWidth/2 - (cCount*fw+eCount*fw*3/5+nCount*fw*3/5)/2
		y = imgHeight/2 + fw*4/11
		return
	}

	x, y := calculateCenterLocation(drawStr)
	pt := freetype.Pt(x, y)

	_, err = c.DrawString(drawStr, pt)
	if err != nil {
		log.Println("draw error:", err)
		return err
	}

	// Encode as PNG.

	// save to db
	buf := new(bytes.Buffer)
	err = png.Encode(buf, dst)
	if err != nil {
		return err
	}
	return db.Hset("user_avatar", sdb.I2b(uid), buf.Bytes())

	// save to local
	/*
		f, _ := os.Create(avatarDir + "/" + strconv.FormatUint(uid, 10) + ".jpg")
		defer func() {
			_ = f.Close()
		}()
		return png.Encode(f, dst)
	*/

}

// 获取字符集，仅调用一次
func getFontFamily() (*truetype.Font, error) {
	// 这里需要读取中文字体，否则中文文字会变成方格
	fontBytes, err := os.ReadFile(fontFile)
	if err != nil {
		log.Println("read file error:", err)
		return &truetype.Font{}, err
	}

	var ft *truetype.Font
	ft, err = freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println("parse font error:", err)
		return &truetype.Font{}, err
	}

	return ft, err
}

func generateFontColorByBgColor(bg color.RGBA) color.RGBA {
	l := (int(bg.R)*229 + int(bg.G)*587 + int(bg.B)*114) / 1000
	if l >= 128 {
		return color.RGBA{
			R: 0,
			G: 0,
			B: 0,
			A: bg.A,
		}
	}

	return color.RGBA{
		R: 255,
		G: 255,
		B: 255,
		A: bg.A,
	}
}

func str2rgb(text string) color.RGBA {
	s384 := sha512.New384()
	s384.Write([]byte(text))
	digest := hex.EncodeToString(s384.Sum(nil))

	subSize := len(digest) / 3

	mv := big.NewInt(math.MaxInt64)
	mv.SetString(strings.Repeat("f", subSize), 16)

	maxValue := big.NewFloat(math.MaxFloat64)
	maxValue.SetInt(mv)

	digests := make([]string, 3)
	for i := 0; i < 3; i++ {
		digests[i] = digest[i*subSize : (i+1)*subSize]
	}

	goldPoint := big.NewFloat(0.618033988749895)

	rgbInt := make([]int64, 3)
	for i, v := range digests {
		in := big.NewInt(math.MaxInt64)
		in.SetString(v, 16)

		inv := big.NewFloat(math.MaxFloat64)
		inv.SetInt(in)

		inf := big.NewFloat(math.MaxFloat64)
		inf.Quo(inv, maxValue).Add(inf, goldPoint)

		oneFloat := big.NewFloat(1)
		cmp := inf.Cmp(oneFloat)
		if cmp > -1 {
			inf.Sub(inf, oneFloat)
		}
		inf.Mul(inf, big.NewFloat(255)).Add(inf, big.NewFloat(0.5)).Sub(inf, big.NewFloat(0.0000005))

		i64, _ := inf.Int64()

		rgbInt[i] = i64
	}

	return color.RGBA{R: uint8(rgbInt[0]), G: uint8(rgbInt[1]), B: uint8(rgbInt[2]), A: 0xff}
}

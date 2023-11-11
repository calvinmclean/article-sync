package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"

	"github.com/fogleman/gg"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
)

func createCoverImage(gopherURL, title string) (image.Image, error) {
	gopher, err := downloadAndResizeImage(gopherURL)
	if err != nil {
		return nil, fmt.Errorf("error getting gopher image: %w", err)
	}

	text, err := createText(title)
	if err != nil {
		return nil, fmt.Errorf("error creating text image: %w", err)
	}

	combined, err := combine(gopher, text)
	if err != nil {
		return nil, fmt.Errorf("error combining images: %w", err)
	}

	return combined, nil
}

func downloadAndResizeImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request to get image: %w", err)
	}
	defer resp.Body.Close()

	img, err := resizeImage(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error resizing image: %w", err)
	}

	return img, nil
}

func resizeImage(r io.Reader) (*image.RGBA, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %w", err)
	}

	ratio := 420.0 / float64(img.Bounds().Max.Y)
	x := float64(img.Bounds().Max.X) * ratio
	scaledImg := image.NewRGBA(image.Rect(0, 0, int(x), 420))

	draw.BiLinear.Scale(scaledImg, scaledImg.Rect, img, img.Bounds(), draw.Over, nil)

	return scaledImg, nil
}

func createText(text string) (image.Image, error) {
	imgWidth := 1000 - 392
	imgHeight := 420

	dc := gg.NewContext(imgWidth, imgHeight)

	ff, err := freetype.ParseFont(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("error creating font: %w", err)
	}

	dc.SetFontFace(truetype.NewFace(ff, &truetype.Options{
		Size:    72,
		Hinting: font.HintingFull,
	}))

	textMargin := 60.0
	maxWidth := float64(imgWidth) - textMargin

	lineWidth := 1.5
	w, _ := dc.MeasureMultilineString(text, lineWidth)
	numLines := float64(int(w/maxWidth + 0.5))

	if numLines > 3 {
		dc.SetFontFace(truetype.NewFace(ff, &truetype.Options{
			Size:    50,
			Hinting: font.HintingFull,
		}))
	}

	x := float64(imgWidth / 2)
	y := float64(imgHeight / 2)
	dc.SetColor(color.Black)
	dc.DrawStringWrapped(text, x, y, 0.5, 0.5, maxWidth, lineWidth, gg.AlignCenter)

	return dc.Image(), nil
}

func combine(gopher, text image.Image) (image.Image, error) {
	imgWidth := 1000
	imgHeight := 420

	dc := gg.NewContext(imgWidth, imgHeight)

	dc.DrawImage(gopher, 0, 0)
	dc.DrawImage(text, 392, 0)

	return dc.Image(), nil
}

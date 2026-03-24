package imagecache

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

var errSkipResize = errors.New("imagecache: image already within max side")

// resizeToMaxJPEG decodifica, reduz mantendo proporção (maior lado = maxSide) e codifica JPEG Q85.
// Se já cabe em maxSide, retorna errSkipResize (caller mantém bytes originais).
func resizeToMaxJPEG(data []byte, maxSide int) ([]byte, string, error) {
	if maxSide <= 0 {
		return nil, "", errSkipResize
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= maxSide && sh <= maxSide {
		return nil, "", errSkipResize
	}
	var nw, nh int
	if sw >= sh {
		nw = maxSide
		nh = maxSide * sh / sw
	} else {
		nh = maxSide
		nw = maxSide * sw / sh
	}
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}

package qr

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	// Register additional image decoders (webp, bmp) with image.Decode.
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// Decode reads an image (PNG/JPEG/WebP/BMP/GIF) and returns the decoded QR
// payload string.
func Decode(imgBytes []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", errors.New("qr: unsupported image format — upload PNG, JPG, or WebP")
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", errors.New("qr: no QR code detected")
	}

	reader := qrcode.NewQRCodeReader()
	result, err := reader.Decode(bmp, nil)
	if err != nil {
		return "", errors.New("qr: no QR code found in image")
	}
	return result.GetText(), nil
}
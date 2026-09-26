package invoice

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Data is the transaction info rendered on the invoice.
type Data struct {
	TransactionID string
	Reference     string
	StoreName     string
	Amount        int64 // unique amount (what the buyer actually paid)
	Provider      string
	PaidAt        time.Time
}

// palette — matches the dashboard design tokens.
var (
	cBG      = color.RGBA{0x0B, 0x0D, 0x10, 0xFF}
	cCard    = color.RGBA{0x12, 0x15, 0x1B, 0xFF}
	cLine    = color.RGBA{0x22, 0x28, 0x32, 0xFF}
	cFG      = color.RGBA{0xE8, 0xEB, 0xF0, 0xFF}
	cMuted   = color.RGBA{0x8B, 0x94, 0xA3, 0xFF}
	cMint    = color.RGBA{0x10, 0xB9, 0x81, 0xFF}
	cMintDim = color.RGBA{0x0C, 0x2B, 0x22, 0xFF}
)

const (
	width  = 600
	height = 720
	padX   = 44
)

type fontSet struct {
	bold   *sfnt.Font
	medium *sfnt.Font
	mono   *sfnt.Font
}

func loadFonts() (*fontSet, error) {
	b, err := sfnt.Parse(gobold.TTF)
	if err != nil {
		return nil, err
	}
	m, err := sfnt.Parse(gomedium.TTF)
	if err != nil {
		return nil, err
	}
	mo, err := sfnt.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}
	return &fontSet{bold: b, medium: m, mono: mo}, nil
}

func (f *fontSet) face(which *sfnt.Font, sizePx float64) font.Face {
	face, _ := opentype.NewFace(which, &opentype.FaceOptions{
		Size:    sizePx,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	return face
}

// Generate renders a modern payment-receipt PNG and returns the encoded bytes.
func Generate(d Data) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{cBG}, image.Point{}, draw.Src)

	fs, err := loadFonts()
	if err != nil {
		return nil, err
	}

	// Card background + border.
	fillRounded(img, 20, 20, width-20, height-20, 24, cCard)
	strokeRounded(img, 20, 20, width-20, height-20, 24, cLine)

	// ---- Brand ----
	fillHexagon(img, width/2-13, 58, 13, cMint)
	drawCentered(img, fs.face(fs.bold, 26), "PayGateMe", width/2+16, 82, cFG)

	// ---- Header ----
	drawCentered(img, fs.face(fs.medium, 17), "PAYMENT RECEIVED", width/2, 152, cMint)

	// ---- Amount ----
	drawCentered(img, fs.face(fs.bold, 50), formatRupiah(d.Amount), width/2, 214, cFG)

	// Status pill.
	pill := "  SETTLED  "
	pillFace := fs.face(fs.medium, 16)
	pillW := font.MeasureString(pillFace, pill).Round()
	pillX := width/2 - pillW/2
	fillRounded(img, pillX, 232, pillX+pillW, 260, 14, cMintDim)
	drawCentered(img, pillFace, pill, width/2, 253, cMint)

	// ---- Divider ----
	hLine(img, padX, 300, width-padX, cLine)

	// ---- Details ----
	rows := [][2]string{
		{"Transaction", d.TransactionID},
		{"Reference", d.Reference},
		{"Store", d.StoreName},
		{"Provider", d.Provider},
		{"Date", d.PaidAt.Format("02 Jan 2006")},
		{"Time", d.PaidAt.Format("15:04") + " WIB"},
	}
	y := 350
	for _, r := range rows {
		drawLeft(img, fs.face(fs.medium, 17), r[0], padX, y, cMuted)
		drawRight(img, fs.face(fs.mono, 17), r[1], width-padX, y, cFG)
		y += 54
	}

	hLine(img, padX, y+14, width-padX, cLine)

	// ---- Footer ----
	drawCentered(img, fs.face(fs.medium, 14), "Powered by PayGateMe · paygateme.com", width/2, height-36, cMuted)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ---- drawing helpers ----

func drawLeft(img *image.RGBA, face font.Face, s string, x, y int, c color.Color) {
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

func drawCentered(img *image.RGBA, face font.Face, s string, cx, y int, c color.Color) {
	w := font.MeasureString(face, s).Round()
	drawLeft(img, face, s, cx-w/2, y, c)
}

func drawRight(img *image.RGBA, face font.Face, s string, rightX, y int, c color.Color) {
	w := font.MeasureString(face, s).Round()
	drawLeft(img, face, s, rightX-w, y, c)
}

func hLine(img *image.RGBA, x0, x1, y int, c color.Color) {
	for x := x0; x <= x1; x++ {
		img.Set(x, y, c)
	}
}

func fillRounded(img *image.RGBA, x0, y0, x1, y1, r int, c color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if inRounded(x, y, x0, y0, x1, y1, r) {
				img.Set(x, y, c)
			}
		}
	}
}

func strokeRounded(img *image.RGBA, x0, y0, x1, y1, r int, c color.Color) {
	// Draw a 2px border ring.
	for t := 0; t < 2; t++ {
		xx0, yy0, xx1, yy1 := x0+t, y0+t, x1-t, y1-t
		for y := yy0; y < yy1; y++ {
			for x := xx0; x < xx1; x++ {
				if inRounded(x, y, xx0, yy0, xx1, yy1, r) &&
					!inRounded(x, y, xx0+1, yy0+1, xx1-1, yy1-1, r-1) {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func inRounded(x, y, x0, y0, x1, y1, r int) bool {
	if x < x0 || x >= x1 || y < y0 || y >= y1 {
		return false
	}
	cx0, cy0 := x0+r, y0+r
	cx1, cy1 := x1-r-1, y1-r-1
	if x < cx0 && y < cy0 {
		return dist2(x, y, cx0, cy0) <= r*r
	}
	if x > cx1 && y < cy0 {
		return dist2(x, y, cx1, cy0) <= r*r
	}
	if x < cx0 && y > cy1 {
		return dist2(x, y, cx0, cy1) <= r*r
	}
	if x > cx1 && y > cy1 {
		return dist2(x, y, cx1, cy1) <= r*r
	}
	return true
}

func dist2(x, y, cx, cy int) int {
	dx, dy := x-cx, y-cy
	return dx*dx + dy*dy
}

func fillHexagon(img *image.RGBA, cx, cy, r int, c color.Color) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if inHex(x, y, cx, cy, r) {
				img.Set(x, y, c)
			}
		}
	}
}

func inHex(x, y, cx, cy, r int) bool {
	h := int(float64(r) * 0.866)
	if x < cx-h || x > cx+h {
		return false
	}
	fx := float64(x-cx) / float64(h)
	span := float64(r) * (1 - 0.5*absf(fx))
	dy := float64(y - cy)
	return dy >= -span && dy <= span
}

func absf(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func formatRupiah(n int64) string {
	s := fmt.Sprintf("%d", n)
	out := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(c)
	}
	return "Rp " + out
}
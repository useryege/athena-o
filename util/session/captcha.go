package session

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"math/big"
	"strings"
	"time"
)

const (
	captchaAnswerLength = 5
	captchaTTL          = 2 * time.Minute
	captchaKeyPrefix    = "login-captcha|"
	captchaCharset      = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
)

var captchaFont = map[rune][]string{
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10011", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
}

type CaptchaChallenge struct {
	ID           string
	ImageDataURL string
	ExpiresIn    int32
}

func (mgr *SessionManager) NewCaptcha(ctx context.Context) (*CaptchaChallenge, error) {
	id, err := randomCaptchaID()
	if err != nil {
		return nil, err
	}
	answer, err := randomCaptchaAnswer()
	if err != nil {
		return nil, err
	}
	imageDataURL, err := renderCaptchaDataURL(answer)
	if err != nil {
		return nil, err
	}
	if err := mgr.storage.SaveCaptcha(ctx, id, answer, captchaTTL); err != nil {
		return nil, err
	}
	return &CaptchaChallenge{
		ID:           id,
		ImageDataURL: imageDataURL,
		ExpiresIn:    int32(captchaTTL.Seconds()),
	}, nil
}

func (mgr *SessionManager) VerifyCaptcha(ctx context.Context, id, answer string) error {
	if id == "" || answer == "" {
		return InvalidLoginErr
	}
	expected, err := mgr.storage.ConsumeCaptcha(ctx, id)
	if err != nil || expected == "" {
		return InvalidLoginErr
	}
	if !strings.EqualFold(strings.TrimSpace(answer), expected) {
		return InvalidLoginErr
	}
	return nil
}

func randomCaptchaID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func randomCaptchaAnswer() (string, error) {
	var b strings.Builder
	b.Grow(captchaAnswerLength)
	for i := 0; i < captchaAnswerLength; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(captchaCharset))))
		if err != nil {
			return "", err
		}
		b.WriteByte(captchaCharset[n.Int64()])
	}
	return b.String(), nil
}

func renderCaptchaDataURL(answer string) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, 164, 54))
	fillRect(img, img.Bounds(), color.RGBA{R: 248, G: 250, B: 252, A: 255})

	for i := 0; i < 42; i++ {
		x, err := randomInt(img.Bounds().Dx())
		if err != nil {
			return "", err
		}
		y, err := randomInt(img.Bounds().Dy())
		if err != nil {
			return "", err
		}
		img.Set(x, y, color.RGBA{R: 94, G: 114, B: 138, A: 255})
	}
	for i := 0; i < 5; i++ {
		x0, err := randomInt(img.Bounds().Dx())
		if err != nil {
			return "", err
		}
		y0, err := randomInt(img.Bounds().Dy())
		if err != nil {
			return "", err
		}
		x1, err := randomInt(img.Bounds().Dx())
		if err != nil {
			return "", err
		}
		y1, err := randomInt(img.Bounds().Dy())
		if err != nil {
			return "", err
		}
		drawLine(img, x0, y0, x1, y1, color.RGBA{R: 171, G: 184, B: 199, A: 255})
	}

	for i, r := range answer {
		x := 12 + i*30
		y := 9
		drawGlyph(img, r, x, y, 4, color.RGBA{R: 31, G: 41, B: 55, A: 255})
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func randomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func drawGlyph(img *image.RGBA, r rune, x, y, scale int, ink color.Color) {
	rows, ok := captchaFont[r]
	if !ok {
		return
	}
	for rowIndex, row := range rows {
		for colIndex, v := range row {
			if v != '1' {
				continue
			}
			fillRect(img, image.Rect(x+colIndex*scale, y+rowIndex*scale, x+(colIndex+1)*scale-1, y+(rowIndex+1)*scale-1), ink)
		}
	}
}

func fillRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if image.Pt(x, y).In(img.Bounds()) {
				img.Set(x, y, c)
			}
		}
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

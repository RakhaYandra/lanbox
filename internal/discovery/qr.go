package discovery

import (
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QR renders text as terminal block art (two rows per module line).
// It returns an error for content too large to encode.
func QR(text string) (string, error) {
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return "", err
	}
	m := q.Bitmap()
	var b strings.Builder
	white := "\033[47m  \033[0m"
	black := "\033[40m  \033[0m"
	for _, row := range m {
		for _, v := range row {
			if v {
				b.WriteString(black)
			} else {
				b.WriteString(white)
			}
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

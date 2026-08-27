// Package avatarimage validates private account and wallet avatar uploads.
package avatarimage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"golang.org/x/image/vp8"
	"golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

const (
	DefaultMaxBytes = int64(2 * 1024 * 1024)
	MaxDimension    = 4096
	MaxPixels       = 16_000_000
)

var (
	ErrTooLarge       = errors.New("avatar exceeds the configured size limit")
	ErrUnsupported    = errors.New("avatar must be a JPEG, PNG, or non-animated WebP image")
	ErrDimensions     = errors.New("avatar dimensions exceed the supported limit")
	ErrInvalidContent = errors.New("avatar image content is invalid")
)

type Image struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}

func Validate(reader io.Reader, maxBytes int64) (Image, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return Image{}, fmt.Errorf("read avatar: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return Image{}, ErrTooLarge
	}
	if len(data) == 0 {
		return Image{}, ErrInvalidContent
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Image{}, ErrInvalidContent
	}
	var contentType string
	switch format {
	case "jpeg":
		contentType = "image/jpeg"
	case "png":
		contentType = "image/png"
	case "webp":
		inspection, inspectErr := inspectWebP(data)
		if inspectErr != nil {
			return Image{}, ErrInvalidContent
		}
		if inspection.animated {
			return Image{}, ErrUnsupported
		}
		if inspection.frameWidth != config.Width || inspection.frameHeight != config.Height {
			return Image{}, ErrInvalidContent
		}
		contentType = "image/webp"
	default:
		return Image{}, ErrUnsupported
	}
	if config.Width <= 0 || config.Height <= 0 {
		return Image{}, ErrInvalidContent
	}
	if config.Width > MaxDimension || config.Height > MaxDimension || config.Width > MaxPixels/config.Height {
		return Image{}, ErrDimensions
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decodedFormat != format {
		return Image{}, ErrInvalidContent
	}
	decodedBounds := decoded.Bounds()
	if decodedBounds.Dx() != config.Width || decodedBounds.Dy() != config.Height {
		return Image{}, ErrInvalidContent
	}
	return Image{Data: data, ContentType: contentType, Width: config.Width, Height: config.Height}, nil
}

type webPInspection struct {
	animated    bool
	frameWidth  int
	frameHeight int
}

func inspectWebP(data []byte) (webPInspection, error) {
	if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return webPInspection{}, ErrInvalidContent
	}
	declaredLength := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if declaredLength < 16 || declaredLength > uint64(len(data)) {
		return webPInspection{}, ErrInvalidContent
	}
	inspection := webPInspection{}
	seenExtendedHeader := false
	seenFrame := false
	containerEnd := int(declaredLength)
	for offset := 12; offset < containerEnd; {
		if offset+8 > containerEnd {
			return webPInspection{}, ErrInvalidContent
		}
		chunk := string(data[offset : offset+4])
		rawSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		payload := offset + 8
		if payload > containerEnd || rawSize > uint64(containerEnd-payload) {
			return webPInspection{}, ErrInvalidContent
		}
		size := int(rawSize)
		chunkData := data[payload : payload+size]
		switch chunk {
		case "VP8X":
			if seenExtendedHeader || size != 10 {
				return webPInspection{}, ErrInvalidContent
			}
			seenExtendedHeader = true
			inspection.animated = chunkData[0]&0x02 != 0
		case "ANIM", "ANMF":
			inspection.animated = true
		case "VP8 ":
			if seenFrame {
				return webPInspection{}, ErrInvalidContent
			}
			decoder := vp8.NewDecoder()
			decoder.Init(bytes.NewReader(chunkData), size)
			frame, err := decoder.DecodeFrameHeader()
			if err != nil || frame.Width <= 0 || frame.Height <= 0 {
				return webPInspection{}, ErrInvalidContent
			}
			inspection.frameWidth, inspection.frameHeight = frame.Width, frame.Height
			seenFrame = true
		case "VP8L":
			if seenFrame {
				return webPInspection{}, ErrInvalidContent
			}
			frame, err := vp8l.DecodeConfig(bytes.NewReader(chunkData))
			if err != nil || frame.Width <= 0 || frame.Height <= 0 {
				return webPInspection{}, ErrInvalidContent
			}
			inspection.frameWidth, inspection.frameHeight = frame.Width, frame.Height
			seenFrame = true
		}
		next := payload + size + size%2
		if next > containerEnd {
			return webPInspection{}, ErrInvalidContent
		}
		offset = next
	}
	if !inspection.animated && !seenFrame {
		return webPInspection{}, ErrInvalidContent
	}
	return inspection, nil
}

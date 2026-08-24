package accountavatarhttp

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
	defaultMaxAvatarBytes = int64(2 * 1024 * 1024)
	maxAvatarDimension    = 4096
	maxAvatarPixels       = 4096 * 4096
)

var (
	errAvatarTooLarge       = errors.New("avatar exceeds the configured size limit")
	errAvatarUnsupported    = errors.New("avatar must be a JPEG, PNG, or non-animated WebP image")
	errAvatarDimensions     = errors.New("avatar dimensions exceed the supported limit")
	errAvatarInvalidContent = errors.New("avatar image content is invalid")
)

type validatedImage struct {
	data        []byte
	contentType string
	width       int
	height      int
}

func validateImage(reader io.Reader, maxBytes int64) (validatedImage, error) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxAvatarBytes
	}

	data, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return validatedImage{}, fmt.Errorf("read avatar: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return validatedImage{}, errAvatarTooLarge
	}
	if len(data) == 0 {
		return validatedImage{}, errAvatarInvalidContent
	}

	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return validatedImage{}, errAvatarInvalidContent
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
			return validatedImage{}, errAvatarInvalidContent
		}
		if inspection.animated {
			return validatedImage{}, errAvatarUnsupported
		}
		if inspection.frameWidth != config.Width || inspection.frameHeight != config.Height {
			return validatedImage{}, errAvatarInvalidContent
		}
		contentType = "image/webp"
	default:
		return validatedImage{}, errAvatarUnsupported
	}

	if config.Width <= 0 || config.Height <= 0 {
		return validatedImage{}, errAvatarInvalidContent
	}
	if config.Width > maxAvatarDimension || config.Height > maxAvatarDimension || config.Width > maxAvatarPixels/config.Height {
		return validatedImage{}, errAvatarDimensions
	}

	decoded, decodedFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || decodedFormat != format {
		return validatedImage{}, errAvatarInvalidContent
	}
	decodedBounds := decoded.Bounds()
	if decodedBounds.Dx() != config.Width || decodedBounds.Dy() != config.Height {
		return validatedImage{}, errAvatarInvalidContent
	}

	return validatedImage{
		data:        data,
		contentType: contentType,
		width:       config.Width,
		height:      config.Height,
	}, nil
}

type webPInspection struct {
	animated    bool
	frameWidth  int
	frameHeight int
}

func inspectWebP(data []byte) (webPInspection, error) {
	if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return webPInspection{}, errAvatarInvalidContent
	}

	declaredLength := uint64(binary.LittleEndian.Uint32(data[4:8])) + 8
	if declaredLength < 16 || declaredLength > uint64(len(data)) {
		return webPInspection{}, errAvatarInvalidContent
	}
	inspection := webPInspection{}
	seenExtendedHeader := false
	seenFrame := false
	containerEnd := int(declaredLength)
	for offset := 12; offset < containerEnd; {
		if offset+8 > containerEnd {
			return webPInspection{}, errAvatarInvalidContent
		}
		chunk := string(data[offset : offset+4])
		rawSize := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		payload := offset + 8
		if payload > containerEnd || rawSize > uint64(containerEnd-payload) {
			return webPInspection{}, errAvatarInvalidContent
		}
		size := int(rawSize)
		chunkData := data[payload : payload+size]
		switch chunk {
		case "VP8X":
			if seenExtendedHeader || size != 10 {
				return webPInspection{}, errAvatarInvalidContent
			}
			seenExtendedHeader = true
			inspection.animated = chunkData[0]&0x02 != 0
		case "ANIM", "ANMF":
			inspection.animated = true
		case "VP8 ":
			if seenFrame {
				return webPInspection{}, errAvatarInvalidContent
			}
			decoder := vp8.NewDecoder()
			decoder.Init(bytes.NewReader(chunkData), size)
			frame, err := decoder.DecodeFrameHeader()
			if err != nil || frame.Width <= 0 || frame.Height <= 0 {
				return webPInspection{}, errAvatarInvalidContent
			}
			inspection.frameWidth, inspection.frameHeight = frame.Width, frame.Height
			seenFrame = true
		case "VP8L":
			if seenFrame {
				return webPInspection{}, errAvatarInvalidContent
			}
			frame, err := vp8l.DecodeConfig(bytes.NewReader(chunkData))
			if err != nil || frame.Width <= 0 || frame.Height <= 0 {
				return webPInspection{}, errAvatarInvalidContent
			}
			inspection.frameWidth, inspection.frameHeight = frame.Width, frame.Height
			seenFrame = true
		}
		next := payload + size + size%2
		if next > containerEnd {
			return webPInspection{}, errAvatarInvalidContent
		}
		offset = next
	}
	if !inspection.animated && !seenFrame {
		return webPInspection{}, errAvatarInvalidContent
	}
	return inspection, nil
}

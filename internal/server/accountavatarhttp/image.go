package accountavatarhttp

import (
	"io"

	"github.com/useryege/athena/internal/avatarimage"
)

const defaultMaxAvatarBytes = avatarimage.DefaultMaxBytes

var (
	errAvatarTooLarge       = avatarimage.ErrTooLarge
	errAvatarUnsupported    = avatarimage.ErrUnsupported
	errAvatarDimensions     = avatarimage.ErrDimensions
	errAvatarInvalidContent = avatarimage.ErrInvalidContent
)

type validatedImage struct {
	data        []byte
	contentType string
	width       int
	height      int
}

func validateImage(reader io.Reader, maxBytes int64) (validatedImage, error) {
	image, err := avatarimage.Validate(reader, maxBytes)
	if err != nil {
		return validatedImage{}, err
	}
	return validatedImage{
		data:        image.Data,
		contentType: image.ContentType,
		width:       image.Width,
		height:      image.Height,
	}, nil
}

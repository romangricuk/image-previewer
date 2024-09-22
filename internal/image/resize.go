package image

import (
	"bytes"
	"context"
	"github.com/romangricuk/image-previewer/internal/logger"

	"github.com/disintegration/imaging"
)

func ResizeImage(ctx context.Context, data []byte, width, height int, log logger.Logger) ([]byte, error) {
	select {
	case <-ctx.Done():
		log.Warn("ResizeImage operation cancelled")
		return nil, ctx.Err()
	default:
		// Продолжаем обработку
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		log.Errorf("Failed to decode image: %v", err)
		return nil, err
	}

	// Изменение размера с обрезкой
	img = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	var buf bytes.Buffer
	err = imaging.Encode(&buf, img, imaging.JPEG)
	if err != nil {
		log.Errorf("Failed to encode image: %v", err)
		return nil, err
	}

	return buf.Bytes(), nil
}

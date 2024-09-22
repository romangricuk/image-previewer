package utils

import (
	"context"
	"github.com/romangricuk/image-previewer/internal/logger"
	"net/http"
)

func FetchImage(ctx context.Context, r *http.Request, imageURL string, log logger.Logger) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+imageURL, nil)
	if err != nil {
		log.Errorf("Failed to create request to fetch image: %v", err)
		return nil, err
	}

	// Проксирование заголовков
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error fetching image from URL %s: %v", imageURL, err)
		return nil, err
	}

	return resp, nil
}

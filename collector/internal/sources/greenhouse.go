package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"jobpulse/collector/internal/model"
)

type greenhouseResponse struct {
	Jobs []struct {
		ID          int64  `json:"id"`
		Title       string `json:"title"`
		AbsoluteURL string `json:"absolute_url"`
		Location    struct {
			Name string `json:"name"`
		} `json:"location"`
		Content string `json:"content"`
	} `json:"jobs"`
}

func FetchGreenhouse(ctx context.Context, client *http.Client, board string) ([]model.JobPosting, error) {
	url := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs?content=true", board)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("greenhouse board %q returned %s", board, resp.Status)
	}
	var payload greenhouseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode greenhouse board %q: %w", board, err)
	}
	jobs := make([]model.JobPosting, 0, len(payload.Jobs))
	for _, job := range payload.Jobs {
		jobs = append(jobs, model.JobPosting{
			ID:          fmt.Sprintf("greenhouse:%s:%d", board, job.ID),
			Company:     board,
			Title:       strings.TrimSpace(job.Title),
			Location:    strings.TrimSpace(job.Location.Name),
			URL:         job.AbsoluteURL,
			Description: strings.TrimSpace(job.Content),
			Source:      "greenhouse",
		})
	}
	return jobs, nil
}

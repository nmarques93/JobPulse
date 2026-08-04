package model

import "time"

type JobPosting struct {
	ID           string    `json:"id"`
	Company      string    `json:"company"`
	Title        string    `json:"title"`
	Location     string    `json:"location"`
	URL          string    `json:"url"`
	Description  string    `json:"description"`
	Source       string    `json:"source"`
	DiscoveredAt time.Time `json:"discovered_at"`
}

package storage

import (
	"testing"
	"time"

	"jobpulse/collector/internal/model"
)

func TestSaveDetectsNewAndChangedPostings(t *testing.T) {
	store, err := Open(t.TempDir() + "/jobpulse.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	posting := model.JobPosting{
		ID: "greenhouse:acme:1", Company: "acme", Title: "Backend Engineer",
		Location: "Remote", URL: "https://example.test/jobs/1", Description: "Build APIs",
		Source: "greenhouse", DiscoveredAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}
	newPosting, err := store.Save(posting)
	if err != nil || !newPosting {
		t.Fatalf("first save: new=%v err=%v", newPosting, err)
	}
	samePosting, err := store.Save(posting)
	if err != nil || samePosting {
		t.Fatalf("duplicate save: new=%v err=%v", samePosting, err)
	}
	posting.Description = "Build reliable APIs and workers"
	changedPosting, err := store.Save(posting)
	if err != nil || !changedPosting {
		t.Fatalf("changed save: new=%v err=%v", changedPosting, err)
	}
}

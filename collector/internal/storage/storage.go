package storage

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jobpulse/collector/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }

type ScoredPosting struct {
	ID              string `json:"id"`
	Company         string `json:"company"`
	Title           string `json:"title"`
	Location        string `json:"location"`
	URL             string `json:"url"`
	Score           int    `json:"score"`
	Recommendation  string `json:"recommendation"`
	MatchedKeywords string `json:"matched_keywords"`
	Gaps            string `json:"gaps"`
	Summary         string `json:"summary"`
	ScoredAt        string `json:"scored_at"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ListScored(limit int, recommendation string) ([]ScoredPosting, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	query := `SELECT p.id, p.company, p.title, p.location, p.url, s.score,
		s.recommendation, s.matched_keywords, s.gaps, s.summary, s.scored_at
		FROM scored_postings s JOIN postings p ON p.id = s.posting_id`
	args := []any{}
	if recommendation != "" {
		query += " WHERE s.recommendation = ?"
		args = append(args, recommendation)
	}
	query += " ORDER BY s.scored_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var postings []ScoredPosting
	for rows.Next() {
		var posting ScoredPosting
		if err := rows.Scan(&posting.ID, &posting.Company, &posting.Title, &posting.Location,
			&posting.URL, &posting.Score, &posting.Recommendation, &posting.MatchedKeywords,
			&posting.Gaps, &posting.Summary, &posting.ScoredAt); err != nil {
			return nil, err
		}
		postings = append(postings, posting)
	}
	return postings, rows.Err()
}

// Save returns true only when the posting is new or its meaningful content changed.
func (s *Store) Save(posting model.JobPosting) (bool, error) {
	now := posting.DiscoveredAt.UTC().Format(time.RFC3339)
	hash := sha256.Sum256([]byte(strings.Join([]string{posting.Title, posting.Location, posting.URL, posting.Description}, "\x00")))
	contentHash := hex.EncodeToString(hash[:])
	var previous string
	err := s.db.QueryRow("SELECT content_hash FROM postings WHERE id = ?", posting.ID).Scan(&previous)
	if err == nil && previous == contentHash {
		_, err = s.db.Exec("UPDATE postings SET last_seen_at = ?, active = 1 WHERE id = ?", now, posting.ID)
		return false, err
	}
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	_, err = s.db.Exec(`INSERT INTO postings
		(id, company, title, location, url, description, source, content_hash, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET company=excluded.company, title=excluded.title,
		location=excluded.location, url=excluded.url, description=excluded.description,
		source=excluded.source, content_hash=excluded.content_hash, last_seen_at=excluded.last_seen_at,
		active=1`, posting.ID, posting.Company, posting.Title, posting.Location, posting.URL,
		posting.Description, posting.Source, contentHash, now, now)
	return true, err
}

const schema = `
CREATE TABLE IF NOT EXISTS postings (
 id TEXT PRIMARY KEY, company TEXT NOT NULL, title TEXT NOT NULL, location TEXT NOT NULL,
 url TEXT NOT NULL, description TEXT NOT NULL, source TEXT NOT NULL, content_hash TEXT NOT NULL,
 first_seen_at TEXT NOT NULL, last_seen_at TEXT NOT NULL, active INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS scored_postings (
 posting_id TEXT PRIMARY KEY, score INTEGER NOT NULL, recommendation TEXT NOT NULL,
 matched_keywords TEXT NOT NULL, gaps TEXT NOT NULL, summary TEXT NOT NULL,
 scoring_status TEXT NOT NULL, attempts INTEGER NOT NULL DEFAULT 1, last_error TEXT,
 scored_at TEXT NOT NULL
);`

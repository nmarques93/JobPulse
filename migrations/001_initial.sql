CREATE TABLE IF NOT EXISTS postings (
    id TEXT PRIMARY KEY,
    company TEXT NOT NULL,
    title TEXT NOT NULL,
    location TEXT NOT NULL,
    url TEXT NOT NULL,
    description TEXT NOT NULL,
    source TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS scored_postings (
    posting_id TEXT PRIMARY KEY REFERENCES postings(id),
    score INTEGER NOT NULL,
    recommendation TEXT NOT NULL,
    matched_keywords TEXT NOT NULL,
    gaps TEXT NOT NULL,
    summary TEXT NOT NULL,
    scoring_status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 1,
    last_error TEXT,
    scored_at TEXT NOT NULL
);

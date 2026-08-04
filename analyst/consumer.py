import json
import logging
import os
import signal
import sqlite3
import time
from datetime import datetime, timezone

import redis

from .filters import score_posting

logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
LOGGER = logging.getLogger("jobpulse.analyst")


SCHEMA = """
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
);"""


class Analyst:
    def __init__(self) -> None:
        self.stream = os.getenv("REDIS_STREAM", "job.posting.discovered")
        self.group = os.getenv("REDIS_GROUP", "analyst")
        self.consumer = os.getenv("REDIS_CONSUMER", f"analyst-{os.getpid()}")
        self.redis = redis.Redis.from_url(os.getenv("REDIS_URL", "redis://localhost:6379/0"), decode_responses=True)
        self.db = sqlite3.connect(os.getenv("DATABASE_PATH", "./data/jobpulse.db"))
        self.db.executescript(SCHEMA)
        self.profile = self._load_profile()
        self.running = True

    @staticmethod
    def _load_profile() -> dict:
        path = os.getenv("PROFILE_PATH", "./profile/profile.json")
        try:
            with open(path, encoding="utf-8") as profile:
                return json.load(profile)
        except FileNotFoundError:
            LOGGER.warning("profile not found at %s; all jobs will be skipped", path)
            return {}
        except json.JSONDecodeError as error:
            raise ValueError(f"invalid profile JSON at {path}: {error}") from error

    def stop(self, *_args) -> None:
        self.running = False

    def ensure_group(self) -> None:
        try:
            self.redis.xgroup_create(self.stream, self.group, id="0", mkstream=True)
        except redis.exceptions.ResponseError as error:
            if "BUSYGROUP" not in str(error):
                raise

    def process(self, message: dict[str, str]) -> None:
        posting = json.loads(message["payload"])
        result = score_posting(posting["title"], posting["location"], posting["description"], self.profile)
        summary = json.dumps({
            "role_type": result.role_type,
            "evidence": result.evidence,
            "concerns": result.concerns,
            "compensation": result.compensation.as_dict() if result.compensation else None,
        }, separators=(",", ":"))
        self.db.execute(
            """INSERT INTO scored_postings
            (posting_id, score, recommendation, matched_keywords, gaps, summary, scoring_status, scored_at)
            VALUES (?, ?, ?, ?, ?, ?, 'complete', ?)
            ON CONFLICT(posting_id) DO UPDATE SET score=excluded.score,
            recommendation=excluded.recommendation, matched_keywords=excluded.matched_keywords,
            gaps=excluded.gaps, summary=excluded.summary, scoring_status='complete',
            attempts=scored_postings.attempts + 1, scored_at=excluded.scored_at""",
            (posting["id"], result.score, result.recommendation, json.dumps(result.matched),
             json.dumps(result.gaps), summary, datetime.now(timezone.utc).isoformat()),
        )
        self.db.commit()
        LOGGER.info("scored %s: %s (%d/10)", posting["id"], result.recommendation, result.score)

    def run(self) -> None:
        self.ensure_group()
        while self.running:
            try:
                batches = self.redis.xreadgroup(self.group, self.consumer, {self.stream: ">"}, count=10, block=5000)
                for _stream, messages in batches:
                    for message_id, message in messages:
                        try:
                            self.process(message)
                            self.redis.xack(self.stream, self.group, message_id)
                        except Exception:
                            LOGGER.exception("processing failed for message %s; leaving it pending", message_id)
            except redis.exceptions.RedisError:
                LOGGER.exception("Redis read failed; retrying")
                time.sleep(2)

    def close(self) -> None:
        self.db.close()


def main() -> None:
    analyst = Analyst()
    signal.signal(signal.SIGINT, analyst.stop)
    signal.signal(signal.SIGTERM, analyst.stop)
    try:
        analyst.run()
    finally:
        analyst.close()


if __name__ == "__main__":
    main()

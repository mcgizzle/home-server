# Reddit Sentiment Analysis for NFL Game Ratings

## Overview

Add a new signal to game ratings by scraping Reddit post-game threads and analyzing fan sentiment via LLM. This complements the existing play-by-play excitement rating with real fan reactions.

## Decisions

- **Reddit Access:** Old Reddit JSON endpoints (no auth, simple)
- **Scoring:** Keep sentiment and play-by-play as separate scores
- **Comments:** Top-level only to start
- **Timing:** Internal job queue, runs 30 mins post-game

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│ Old Reddit  │────▶│ Thread       │────▶│ LLM         │────▶│ Sentiment    │
│ JSON API    │     │ Scraper      │     │ Analysis    │     │ Score        │
└─────────────┘     └──────────────┘     └─────────────┘     └──────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │ Job Queue    │
                    │ (30min delay)│
                    └──────────────┘
```

## Components

### 1. Reddit Client (`internal/external/reddit_client.go`)

Using Old Reddit JSON endpoints (no auth required):
```
https://old.reddit.com/r/nfl/search.json?q=Post+Game+Thread+Rams+Seahawks&restrict_sr=on&sort=new
https://old.reddit.com/r/nfl/comments/{thread_id}.json?sort=top&limit=100
```

### 2. Thread Finder

Find the correct post-game thread by:
- Subreddit: `r/nfl`
- Title pattern: `[Post Game Thread]` or `Post Game Thread`
- Team names in title
- Date proximity to game

### 3. Comment Extractor

Extract top-level comments only:
- Limit to top N comments (e.g., 100-200)
- Sort by "top"
- Filter out removed/deleted comments
- Include comment score for weighting

### 4. Sentiment Analyzer (`internal/external/sentiment_adapter.go`)

Pass aggregated comments to LLM with prompt:
```
Analyze these Reddit comments from an NFL post-game thread.
Rate the overall fan excitement/sentiment from 0-100.

Consider:
- Enthusiasm level (caps, exclamations, superlatives)
- References to "instant classic", "game of the year", etc.
- Complaints about refs (negative but engaged)
- "I can't believe..." moments
- Heart attack/stress references (indicates close game)

Return JSON: { "score": 0, "sentiment": "excited|neutral|disappointed", "highlights": ["key themes"] }
```

### 5. Domain Integration

New domain entity:
```go
type SentimentRating struct {
    Source      string    // "reddit"
    ThreadURL   string
    CommentCount int
    Score       int       // 0-100
    Sentiment   string    // excited, neutral, disappointed
    Highlights  []string
    GeneratedAt time.Time
}
```

Add to Competition:
```go
type Competition struct {
    // ... existing fields
    SentimentRating *SentimentRating
}
```

### 5. Job Queue (`internal/infrastructure/queue/`)

Simple interface designed for future migration to OSS tooling (Redis/BullMQ, pg_boss, Temporal, etc.):

```go
type Job struct {
    ID            string
    Type          string        // e.g., "sentiment_analysis"
    Payload       []byte        // JSON-encoded job data
    ScheduledFor  time.Time
    CreatedAt     time.Time
}

type JobQueue interface {
    Schedule(job Job) error
    // Handler is called when job is ready
    Process(ctx context.Context, jobType string, handler func(Job) error)
}
```

**MVP Implementation:** Simple goroutine + timer
```go
type SimpleQueue struct {
    jobs chan Job
}

func (q *SimpleQueue) Schedule(job Job) error {
    delay := time.Until(job.ScheduledFor)
    time.AfterFunc(delay, func() {
        q.jobs <- job
    })
    return nil
}
```

**Future options:**
- [Asynq](https://github.com/hibiken/asynq) - Redis-based, Go native
- [River](https://github.com/riverqueue/river) - Postgres-based, Go native
- [Temporal](https://temporal.io/) - Full workflow orchestration

## Implementation Plan

### Phase 1: Reddit Client (MVP)
- [ ] Create `reddit_client.go` with search and fetch methods
- [ ] Add `cmd/reddit-client` CLI for testing
- [ ] Test finding post-game threads by team names
- [ ] Handle rate limiting / User-Agent requirements

### Phase 2: Sentiment Analysis
- [ ] Create sentiment prompt in `eval/prompts/sentiment.txt`
- [ ] Test with promptfoo against real Reddit threads
- [ ] Create `sentiment_adapter.go` using OpenAI

### Phase 3: Job Queue Infrastructure
- [ ] Define `JobQueue` interface in `internal/infrastructure/queue/`
- [ ] Implement simple goroutine + timer queue (MVP)
- [ ] Wire into main.go alongside existing background jobs

### Phase 4: Integration
- [ ] Add `SentimentRating` to domain
- [ ] Create use case `GenerateSentimentRating`
- [ ] Add database migration for sentiment storage
- [ ] Create `SentimentConsumer` that processes jobs
- [ ] Schedule jobs when games complete

## CLI Tool Design

```bash
# Search for post-game thread
./reddit-client -search "rams seahawks" -after="2025-12-19"

# Fetch thread and analyze sentiment
./reddit-client -thread "https://reddit.com/r/nfl/..." -analyze

# Export for promptfoo testing
./reddit-client -thread "..." -export eval/reddit/rams_seahawks.json
```

## Data Flow Example

```
1. Game ends: Rams 37 - Seahawks 38

2. Find thread:
   GET /r/nfl/search.json?q=Post+Game+Thread+Rams+Seahawks

3. Fetch comments:
   GET /r/nfl/comments/{thread_id}.json?sort=top&limit=100

4. Extract text:
   [
     "GAME OF THE YEAR",
     "My heart can't take this",
     "Geno is CLUTCH",
     ...
   ]

5. LLM Analysis:
   {
     "score": 92,
     "sentiment": "excited",
     "highlights": ["instant classic", "cardiac game", "MVP performance"]
   }
```

## Open Questions

1. **Rate limiting:** How often can we hit Reddit? May need caching/backoff.
2. **Spoilers:** Sentiment analysis inherently contains spoilers - how to handle in UI?
3. **Retry policy:** How many times to retry failed jobs?

## Dependencies

- Old Reddit JSON endpoints (no auth)
- OpenAI API (already have)
- New database table: `sentiment_ratings`
- Queue: in-memory for MVP (swap to Asynq/River later)

## Success Metrics

- Sentiment score correlates with play-by-play score (r > 0.7)
- Can identify "sleeper" games that fans loved but play-by-play missed
- Processing time < 30s per game
- Jobs execute within 5min of scheduled time

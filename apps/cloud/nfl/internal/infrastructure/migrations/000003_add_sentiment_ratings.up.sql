-- Migration 003: Add sentiment ratings table
-- This migration creates the sentiment_ratings table for storing fan sentiment analysis

CREATE TABLE IF NOT EXISTS sentiment_ratings (
    id TEXT PRIMARY KEY,
    competition_id TEXT NOT NULL,
    source TEXT NOT NULL,
    thread_url TEXT,
    comment_count INTEGER,
    score INTEGER NOT NULL,
    sentiment TEXT NOT NULL,
    highlights TEXT,  -- JSON array
    generated_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (competition_id) REFERENCES competitions(id)
);

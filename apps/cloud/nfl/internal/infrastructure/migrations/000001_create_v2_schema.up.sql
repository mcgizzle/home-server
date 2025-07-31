-- Migration 001: Create V2 Clean Architecture Schema
-- This migration creates the normalized schema for the clean architecture redesign

-- Sports catalog
CREATE TABLE sports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

-- Universal competitions (games/matches)
CREATE TABLE competitions (
    id TEXT PRIMARY KEY,
    event_id TEXT UNIQUE,
    sport_id TEXT REFERENCES sports(id),
    season TEXT NOT NULL,
    period TEXT NOT NULL,           -- Week/Game/Round
    period_type TEXT NOT NULL,      -- Regular/Playoff/etc
    start_time DATETIME,
    status TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Teams across all sports
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sport_id TEXT REFERENCES sports(id),
    logo_url TEXT,
    UNIQUE(name, sport_id)
);

-- Competition participants
CREATE TABLE competition_teams (
    competition_id TEXT REFERENCES competitions(id),
    team_id TEXT REFERENCES teams(id),
    home_away TEXT NOT NULL,
    score REAL DEFAULT 0,
    stats JSON, -- JSON for sport-specific stats
    PRIMARY KEY(competition_id, team_id)
);

-- Ratings (extensible for future rating types)
CREATE TABLE ratings (
    id INTEGER PRIMARY KEY,
    competition_id TEXT REFERENCES competitions(id),
    type TEXT NOT NULL,            -- excitement (extensible for future types)
    score INTEGER NOT NULL,
    explanation TEXT,
    spoiler_free TEXT,
    source TEXT,
    confidence REAL,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(competition_id, type, source)
);

-- Rich competition data
CREATE TABLE competition_details (
    competition_id TEXT PRIMARY KEY REFERENCES competitions(id),
    play_by_play JSON,             -- JSON array
    metadata JSON                  -- JSON for sport-specific data
); 
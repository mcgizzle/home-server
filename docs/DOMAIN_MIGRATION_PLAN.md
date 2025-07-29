# Domain Expansion Plan: Multi-Sport & Multi-Rating System

## 🎯 Strategic Decision: Fresh Start Approach

### ✅ **Decision Made: Complete Rebuild**

After cost-benefit analysis, we're choosing a **fresh start approach**:

- **Regeneration Cost**: ~$0.22 (22 cents) for 358 games with GPT-4o-mini
- **Time Investment**: 20 minutes of API calls vs weeks of migration complexity  
- **Risk Level**: Zero migration risks, no data corruption possibilities
- **Architecture**: Clean, future-proof design from day 1

### What We're Building
- 🏗️ **Sport-agnostic** core architecture
- 📊 **Multiple rating types** (AI, sentiment, quality, upset factor)
- 🔌 **Pluggable data sources** (ESPN, NBA API, Reddit, Twitter)
- ⚡ **Built-in analytics** capabilities
- 🚀 **Multi-sport ready** from the start

---

## 🏗️ Target Architecture

### Core Principles
- **Sport-agnostic** core entities
- **Multiple rating types** (AI, sentiment, quality, upset factor)
- **Structured storage** with JSON flexibility where needed
- **Pluggable data sources** (ESPN, NBA API, Reddit, Twitter)
- **Powerful analytics** capabilities built-in

### Key New Entities
```go
type Sport string                    // nfl, nba, mlb, soccer
type Competition struct             // Universal game/match entity
type RatingType string              // ai_excitement, sentiment, etc
type Rating struct                  // Individual rating with type & source
type Result struct                  // Competition + multiple ratings
```

---

## 📋 Fresh Start Implementation Phases

### Phase 1: Clean Database Schema
**Goal:** Create proper normalized database structure from scratch

#### New Database Structure
```sql
-- Sports catalog
CREATE TABLE sports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config TEXT -- JSON for sport-specific settings
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
    stats TEXT, -- JSON for sport-specific stats
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
    play_by_play TEXT,             -- JSON array
    metadata TEXT                  -- JSON for sport-specific data
);
```

#### Core Domain Entities
```go
// New sport-agnostic domain
type Sport string

const (
    SportNFL    Sport = "nfl"
    SportNBA    Sport = "nba" 
    SportMLB    Sport = "mlb"
)

type Competition struct {
    ID         string              `json:"id"`
    EventID    string              `json:"event_id"`
    Sport      Sport               `json:"sport"`
    Season     string              `json:"season"`
    Period     string              `json:"period"`
    PeriodType string              `json:"period_type"`
    StartTime  time.Time           `json:"start_time"`
    Status     string              `json:"status"`
    Teams      []CompetitionTeam   `json:"teams"`
    Rating     *Rating             `json:"rating,omitempty"`
}

type RatingType string

const (
    RatingTypeAI        RatingType = "excitement"
)

type Rating struct {
    Type        RatingType `json:"type"`
    Score       int        `json:"score"`
    Confidence  float64    `json:"confidence"`
    Explanation string     `json:"explanation"`
    SpoilerFree string     `json:"spoiler_free"`
    Source      string     `json:"source"`
    GeneratedAt time.Time  `json:"generated_at"`
}
```

**✅ Success Criteria:** Clean database schema created, core entities defined

---

### Phase 2: Sport-Agnostic Services
**Goal:** Build multi-rating, multi-sport service layer

#### New Service Interfaces
```go
// Abstract data source
type DataSource interface {
    GetSport() Sport
    ListLatest() ([]Competition, error)
    ListSpecific(season, period, periodType string) ([]Competition, error)
    GetCompetition(id string) (Competition, error)
}

// Rating service
type RatingService interface {
    ProduceRating(comp Competition) (Rating, error)
}

// Universal repository
type CompetitionRepository interface {
    SaveCompetition(comp Competition) error
    SaveRating(compID string, rating Rating) error
    LoadCompetition(id string) (Competition, error)
    FindByTeam(teamName string, sport Sport) ([]Competition, error)
    FindByRating(minScore int) ([]Competition, error)
}
```

#### Refactored ESPN Client
```go
// NFL-specific implementation of DataSource
type NFLDataSource struct {
    client    *http.Client
    apiKey    string
    ratingSvc RatingService
}

func (n *NFLDataSource) GetSport() Sport { return SportNFL }
func (n *NFLDataSource) ListLatest() ([]Competition, error) {
    // Convert ESPN data to Competition format
}
```

#### Enhanced Rating Service
```go
type OpenAIRatingService struct {
    client openai.Client
}

func (o *OpenAIRatingService) ProduceRating(comp Competition) (Rating, error) {
    // Generate excitement rating using current logic
    return o.generateExcitementRating(comp), nil
}
```

**✅ Success Criteria:** Sport-agnostic services implemented, NFL working with new architecture

---

### Phase 3: Data Regeneration
**Goal:** Populate new database with clean, structured data

#### Use Existing Backfill Infrastructure
Once the new system is ready, simply use your existing backfill setup:

```bash
# Regular season (weeks 1-16)
./backfill.sh

# Playoffs (manually call for weeks 1-4 of season type 3)
curl "http://localhost:8089/backfill?week=1&season=2024&seasontype=3"
curl "http://localhost:8089/backfill?week=2&season=2024&seasontype=3"
curl "http://localhost:8089/backfill?week=3&season=2024&seasontype=3"
curl "http://localhost:8089/backfill?week=4&season=2024&seasontype=3"
```

#### Cost & Benefits
- **API Cost**: $0.22 for all 358 games
- **No new code needed**: Leverage existing `/backfill` endpoint
- **Data Quality**: Perfect consistency, no legacy artifacts

**✅ Success Criteria:** All 2024 NFL data regenerated in clean format

---

### Phase 4: Analytics Foundation
**Goal:** Build powerful analytics on clean structured data

#### Analytics Service
```go
type AnalyticsService interface {
    GetTeamPerformance(teamName string, season string) TeamStats
    GetRatingDistribution() RatingDistribution
    GetTopGames(limit int) []Competition
    FindHighRatedGames(threshold int) []Competition
}

type TeamStats struct {
    Name            string              `json:"name"`
    GamesPlayed     int                 `json:"games_played"`
    AvgRating       float64             `json:"avg_rating"`
    TopGames        []Competition       `json:"top_games"`
}
```

#### Analytics Queries
```sql
-- Team performance analytics
SELECT 
    t.name,
    COUNT(*) as games_played,
    AVG(r.score) as avg_excitement,
    MAX(r.score) as highest_rated_game
FROM teams t
JOIN competition_teams ct ON t.id = ct.team_id  
JOIN ratings r ON ct.competition_id = r.competition_id
WHERE r.type = 'excitement'
GROUP BY t.name;

-- Rating distribution
SELECT 
    FLOOR(score/10)*10 as score_range,
    COUNT(*) as game_count
FROM ratings 
WHERE type = 'excitement'
GROUP BY FLOOR(score/10)*10
ORDER BY score_range;
```

#### API Endpoints
```go
// Analytics endpoints
http.HandleFunc("/api/analytics/team/{name}", handleTeamAnalytics)
http.HandleFunc("/api/analytics/top-games", handleTopGames)  
http.HandleFunc("/api/analytics/distribution", handleRatingDistribution)
```

**✅ Success Criteria:** Rich analytics available, fast query performance

---

### Phase 5: Performance & Polish
**Goal:** Optimize performance and add production-ready features

#### Performance Optimization
```sql
-- Add strategic indexes
CREATE INDEX idx_competitions_sport_season ON competitions(sport_id, season);
CREATE INDEX idx_ratings_score ON ratings(score DESC);
CREATE INDEX idx_competition_teams_team ON competition_teams(team_id);
```



#### Production Features
- Health check endpoints
- Metrics and monitoring
- Error handling and recovery
- API rate limiting

**✅ Success Criteria:** Production-ready system with optimal performance

---

## 🚀 Expected Benefits

### Immediate (After Phase 3)
- ✅ **Clean, normalized data** structure
- ✅ **Zero technical debt** from legacy migration
- ✅ **Perfect data consistency** across all games

### Medium-term (After Phase 4)
- 📊 **Rich analytics** across all games and teams
- ⚡ **Fast, complex queries** on structured data

### Long-term (After Phase 5)
- 🏗️ **Sport-agnostic architecture** ready for future expansion
- 📈 **Extensible rating system** for additional rating types
- ⚡ **Production-ready** performance and reliability

---

## 💰 Cost Analysis

### Fresh Start Benefits
- **Regeneration Cost**: $0.22 total for all 358 games
- **Risk Level**: Near zero (no migration complexity)
- **Data Quality**: Perfect from day 1
- **Architecture Quality**: Clean, extensible foundation

### Alternative Migration Cost
- **Risk Level**: High (data corruption, rollback complexity)  
- **Technical Debt**: Carrying forward legacy JSON structures
- **Performance**: Slower queries on hybrid schema

**Clear Winner: Fresh Start Approach** 🎯

---

## ⚠️ Risk Mitigation

### Minimal Risks (Fresh Start)
- ✅ **Data backup**: Keep existing database as backup
- ✅ **Test validation**: Run existing e2e tests on new system
- ✅ **Rollback plan**: Can revert to old system if needed
- ✅ **Incremental validation**: Test with sample data first

### Monitoring
- API success rates during regeneration
- Query performance on new schema  
- Rating generation accuracy validation
- Test suite results during development

---

## 🎯 Success Metrics

- ✅ **All 358 games** regenerated successfully  
- ✅ **Analytics queries** perform under 50ms
- ✅ **Excitement ratings** generated for all games
- ✅ **Sport-agnostic** architecture foundation
- ✅ **Clean, maintainable** codebase

---

## 🛠️ Implementation Strategy

### Development Approach
1. **Build new system** completely separate from old
2. **Leverage existing e2e tests** to validate behavior  
3. **Regenerate all data** in single batch run
4. **Switch over** when tests pass
5. **Archive old system** as backup

### Testing Strategy
**Leverage Existing High-Level Integration Tests:**

The existing `main_test.go` provides comprehensive end-to-end tests:

- `TestRealESPNEndToEnd()` - Full data flow from ESPN API to database with mock OpenAI
- `TestRealESPNWithUseCases()` - Use case integration testing
- `TestRealESPNClient()` - External API integration validation

**Regression Test Framework:**
- `run_regression_tests.sh` - Automated regression test runner
- Golden master tests for template data validation
- HTTP endpoint behavior verification

**New System Validation:**
1. **Adapt existing tests** to work with new domain models
2. **Add sport-agnostic test cases** for multi-sport scenarios  
3. **Performance benchmarks** on new schema
4. **Data regeneration validation** against existing results

---

## 📞 Next Steps

1. ✅ **Approve fresh start approach** (COMPLETED)
2. 🔄 **Set up new development branch**
3. 🏗️ **Begin Phase 1: Clean schema implementation**
4. 📊 **Create performance benchmarks**
5. 🚀 **Execute regeneration plan**

---

*This fresh start approach delivers a clean, scalable, multi-sport platform in half the time with zero migration risks and incredible cost efficiency.*

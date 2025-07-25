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

-- Multi-type ratings
CREATE TABLE ratings (
    id INTEGER PRIMARY KEY,
    competition_id TEXT REFERENCES competitions(id),
    type TEXT NOT NULL,            -- ai_excitement, sentiment, quality
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
    Ratings    map[RatingType]Rating `json:"ratings"`
}

type RatingType string

const (
    RatingTypeAI        RatingType = "ai_excitement"
    RatingTypeSentiment RatingType = "sentiment"
    RatingTypeQuality   RatingType = "game_quality"
    RatingTypeUpset     RatingType = "upset_factor"
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

// Multi-rating service
type RatingService interface {
    GetSupportedTypes() []RatingType
    ProduceRating(comp Competition, ratingType RatingType) (Rating, error)
    ProduceAllRatings(comp Competition) map[RatingType]Rating
}

// Universal repository
type CompetitionRepository interface {
    SaveCompetition(comp Competition) error
    SaveRatings(compID string, ratings map[RatingType]Rating) error
    LoadCompetition(id string) (Competition, error)
    FindByTeam(teamName string, sport Sport) ([]Competition, error)
    FindByRating(ratingType RatingType, minScore int) ([]Competition, error)
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

func (o *OpenAIRatingService) ProduceRating(comp Competition, ratingType RatingType) (Rating, error) {
    switch ratingType {
    case RatingTypeAI:
        // Current AI excitement logic
    case RatingTypeQuality:
        // New: rate game quality independent of outcome
    case RatingTypeUpset:
        // New: rate how unexpected the result was
    }
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
    GetRatingDistribution(ratingType RatingType) RatingDistribution
    GetTopGames(ratingType RatingType, limit int) []Competition
    GetRatingCorrelations() map[RatingType]map[RatingType]float64
    FindUpsets(sport Sport, threshold int) []Competition
}

type TeamStats struct {
    Name            string              `json:"name"`
    GamesPlayed     int                 `json:"games_played"`
    AvgRatings      map[RatingType]float64 `json:"avg_ratings"`
    TopGames        []Competition       `json:"top_games"`
    UpsetVictories  []Competition       `json:"upset_victories"`
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
WHERE r.type = 'ai_excitement'
GROUP BY t.name;

-- Rating correlations
SELECT 
    r1.type as rating_1,
    r2.type as rating_2,
    CORR(r1.score, r2.score) as correlation
FROM ratings r1
JOIN ratings r2 ON r1.competition_id = r2.competition_id
WHERE r1.type != r2.type
GROUP BY r1.type, r2.type;
```

#### API Endpoints
```go
// New analytics endpoints
http.HandleFunc("/api/analytics/team/{name}", handleTeamAnalytics)
http.HandleFunc("/api/analytics/top-games", handleTopGames)  
http.HandleFunc("/api/analytics/correlations", handleRatingCorrelations)
http.HandleFunc("/api/analytics/upsets", handleUpsets)
```

**✅ Success Criteria:** Rich analytics available, fast query performance

---

### Phase 5: Multi-Rating Implementation  
**Goal:** Generate multiple rating types for comprehensive game evaluation

#### New Rating Types
```go
// Quality rating (independent of excitement)
func (o *OpenAIRatingService) generateQualityRating(comp Competition) Rating {
    prompt := `Rate this game's overall quality (0-100) based on:
    - Player performance levels
    - Strategic execution  
    - Competitive balance
    - Technical skill displayed
    
    Focus on quality independent of excitement or upset factor.`
}

// Upset factor rating
func (o *OpenAIRatingService) generateUpsetRating(comp Competition) Rating {
    prompt := `Rate how surprising this result was (0-100) based on:
    - Pre-game expectations
    - Team records and standings
    - Historical matchup data
    - Score differential vs expected`
}
```

#### Batch Rating Generation
```go
func (o *OpenAIRatingService) ProduceAllRatings(comp Competition) map[RatingType]Rating {
    ratings := make(map[RatingType]Rating)
    
    // Generate all rating types in parallel
    var wg sync.WaitGroup
    wg.Add(3)
    
    go func() {
        defer wg.Done()
        ratings[RatingTypeAI] = o.generateAIRating(comp)
    }()
    
    go func() {
        defer wg.Done()
        ratings[RatingTypeQuality] = o.generateQualityRating(comp)
    }()
    
    go func() {
        defer wg.Done() 
        ratings[RatingTypeUpset] = o.generateUpsetRating(comp)
    }()
    
    wg.Wait()
    return ratings
}
```

#### UI Updates
- Multi-rating display on game cards
- Rating type filters  
- Correlation visualizations
- Top games by rating type

**✅ Success Criteria:** All games have multiple rating perspectives

---

### Phase 6: Second Sport Integration
**Goal:** Prove architecture extensibility with NBA/MLB

#### NBA Data Source
```go
type NBADataSource struct {
    client *http.Client
    apiKey string
}

func (n *NBADataSource) GetSport() Sport { return SportNBA }

func (n *NBADataSource) ListLatest() ([]Competition, error) {
    // Implement NBA API integration
    // Map NBA games to Competition format
    // Handle NBA-specific periods (quarters, OT)
}
```

#### Sport Configuration
```go
type SportConfig struct {
    ID           Sport             `json:"id"`
    Name         string            `json:"name"`
    PeriodNames  map[string]string `json:"period_names"`
    SeasonTypes  map[string]string `json:"season_types"`
    RatingTypes  []RatingType      `json:"rating_types"`
}

// NFL config
nflConfig := SportConfig{
    ID:   SportNFL,
    Name: "National Football League",
    PeriodNames: map[string]string{
        "1": "Week", "2": "Week", ..., "22": "Super Bowl",
    },
    SeasonTypes: map[string]string{
        "1": "Preseason", "2": "Regular Season", "3": "Playoffs",
    },
}

// NBA config  
nbaConfig := SportConfig{
    ID:   SportNBA,
    Name: "National Basketball Association",
    PeriodNames: map[string]string{
        "1": "Game", "2": "Game", ...,
    },
    SeasonTypes: map[string]string{
        "1": "Preseason", "2": "Regular Season", "3": "Playoffs",
    },
}
```

#### Multi-Sport UI
- Sport selector in navigation
- Sport-specific terminology
- Unified analytics across sports

**✅ Success Criteria:** Two sports working independently with shared infrastructure

---

### Phase 7: Advanced Features
**Goal:** Add sentiment analysis and social media integration

#### Sentiment Rating Service
```go
type SentimentRatingService struct {
    redditClient  RedditClient
    twitterClient TwitterClient
    analyzer      SentimentAnalyzer
}

func (s *SentimentRatingService) ProduceRating(comp Competition) Rating {
    // 1. Find social media discussions
    posts := s.findGameDiscussions(comp)
    
    // 2. Analyze sentiment and engagement
    sentiment := s.analyzer.AnalyzeSentiment(posts)
    
    // 3. Convert to 0-100 excitement score
    score := s.sentimentToExcitement(sentiment)
    
    return Rating{
        Type:        RatingTypeSentiment,
        Score:       score,
        Source:      "social_media",
        Confidence:  sentiment.Confidence,
        Explanation: sentiment.Summary,
    }
}
```

#### Social Media Integration
```go
type RedditClient interface {
    FindGameThreads(teamNames []string, date time.Time) []RedditPost
    GetPostMetrics(postID string) RedditMetrics
}

type SentimentAnalyzer interface {
    AnalyzeSentiment(texts []string) SentimentScore
    GetEngagementScore(metrics []SocialMetrics) float64
}
```

**✅ Success Criteria:** Games have social sentiment scores alongside AI ratings

---

### Phase 8: Performance & Polish
**Goal:** Optimize performance and add production-ready features

#### Performance Optimization
```sql
-- Add strategic indexes
CREATE INDEX idx_competitions_sport_season ON competitions(sport_id, season);
CREATE INDEX idx_ratings_type_score ON ratings(type, score DESC);
CREATE INDEX idx_competition_teams_team ON competition_teams(team_id);
```

#### Caching Layer
```go
type CachedAnalyticsService struct {
    base  AnalyticsService
    cache map[string]interface{}
    ttl   time.Duration
}

// Cache frequently accessed analytics
func (c *CachedAnalyticsService) GetTeamPerformance(team string) TeamStats {
    key := fmt.Sprintf("team_perf_%s", team)
    if cached, ok := c.cache[key]; ok {
        return cached.(TeamStats)
    }
    
    stats := c.base.GetTeamPerformance(team)
    c.cache[key] = stats
    return stats
}
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

### Medium-term (After Phase 5)
- 📊 **Rich analytics** across all games and teams
- 🎯 **Multiple rating perspectives** per game
- ⚡ **Fast, complex queries** on structured data

### Long-term (After Phase 7)
- 🏀 **Multi-sport platform** ready for NBA, MLB, etc.
- 🌐 **Social sentiment** integration
- 📈 **Comprehensive game evaluation** system

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
- ✅ **Multiple rating types** generated for all games
- ✅ **Multi-sport ready** architecture
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

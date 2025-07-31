# Domain Migration Plan: Clean Architecture for NFL Excitement Ratings

## 🎯 Project Scope

### **Current Focus: NFL + AI Excitement Only**
- **Single Sport**: NFL games only (no multi-sport expansion initially)
- **Single Rating**: AI excitement rating only (no additional rating types initially) 
- **Existing Functionality**: Maintain all current features while improving architecture
- **Future Ready**: Build foundation that can support expansion later

### ✅ **Strategic Decision: Complete Rebuild**

After cost-benefit analysis, we're choosing a **fresh start approach**:

- **Regeneration Cost**: ~$0.22 (22 cents) for 358 games with GPT-4o-mini
- **Time Investment**: 20 minutes of API calls vs weeks of migration complexity  
- **Risk Level**: Zero migration risks, no data corruption possibilities
- **Architecture**: Clean, future-proof design from day 1

### What We're Building
- 🏗️ **Clean, normalized** database architecture  
- 📊 **AI excitement rating** system (existing functionality)
- 🏈 **NFL-focused** implementation (no new sports initially)
- ⚡ **Built-in analytics** capabilities
- 🚀 **Foundation ready** for future expansion

---

## 🏗️ Target Architecture

### Core Principles
- **Clean normalized** database schema
- **AI excitement rating** (single rating type initially)
- **Structured storage** with JSON flexibility where needed
- **ESPN data source** for NFL games
- **Powerful analytics** capabilities built-in

### Key New Entities
```go
type Sport string                    // nfl (only sport initially)
type Competition struct             // NFL game entity
type RatingType string              // ai_excitement (only rating type initially)
type Rating struct                  // AI excitement rating with metadata
type Result struct                  // Competition + excitement rating
```

---

## 📋 Implementation Phases

### Phase 1: Database Schema Migration
**Goal:** Replace existing schema with new normalized structure

#### Step 1.1: Create New Schema
```sql
-- Sports catalog
CREATE TABLE sports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config JSON -- JSON for sport-specific settings
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
```

**🛑 VALIDATION STOP:** Verify schema creation, run basic queries

#### Step 1.2: Create V2 Domain Entities
Create new directory structure:
```
internal/
├── v2/
│   ├── domain/
│   │   ├── entities.go      # New sport-agnostic entities
│   │   └── constants.go     # Sport and rating type constants
│   ├── repository/
│   ├── application/
│   └── external/
└── (existing v1 code remains)
```

**File: `internal/v2/domain/entities.go`**
```go
// New sport-agnostic domain
type Sport string

const (
    SportNFL    Sport = "nfl"  // Only NFL initially
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
    RatingTypeAI        RatingType = "excitement"  // Only excitement rating initially
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

**🛑 VALIDATION STOP:** Verify v2 domain entities compile, adapt existing e2e tests

**✅ Phase 1 Success Criteria:** 
- ✅ New database schema created and validated
- ✅ V2 domain entities created in `internal/v2/domain/`
- ✅ V2 entities compile successfully
- ✅ Existing e2e tests adapted to work with v2 entities

---

### Phase 2: Repository Layer Migration
**Goal:** Update repository to work with new schema

#### Step 2.1: Create V2 Repository Interface
**File: `internal/v2/repository/interfaces.go`**
```go
package repository

import "your-app/internal/v2/domain"

// Universal repository interface
type CompetitionRepository interface {
    SaveCompetition(comp domain.Competition) error
    SaveRating(compID string, rating domain.Rating) error
    LoadCompetition(id string) (domain.Competition, error)
    FindByTeam(teamName string, sport domain.Sport) ([]domain.Competition, error)
    FindByRating(minScore int) ([]domain.Competition, error)
}

// Abstract data source interface  
type DataSource interface {
    GetSport() domain.Sport
    ListLatest() ([]domain.Competition, error)
    ListSpecific(season, period, periodType string) ([]domain.Competition, error)
    GetCompetition(id string) (domain.Competition, error)
}

// Rating service interface
type RatingService interface {
    ProduceRating(comp domain.Competition) (domain.Rating, error)
}
```

**🛑 VALIDATION STOP:** Verify v2 interfaces compile and make sense

#### Step 2.2: Create V2 Repository Implementation
**File: `internal/v2/repository/sqlite_repository.go`**
```go
package repository

import (
    "database/sql"
    "your-app/internal/v2/domain"
)

type SQLiteCompetitionRepository struct {
    db *sql.DB
}

func NewSQLiteCompetitionRepository(db *sql.DB) *SQLiteCompetitionRepository {
    return &SQLiteCompetitionRepository{db: db}
}

func (r *SQLiteCompetitionRepository) SaveCompetition(comp domain.Competition) error {
    // Implementation using new schema tables
    // Can reference old implementation in internal/repository/result_repository.go
}

func (r *SQLiteCompetitionRepository) LoadCompetition(id string) (domain.Competition, error) {
    // Implementation using new schema
}
```

**🛑 VALIDATION STOP:** Run e2e tests to verify repository works with new schema

**✅ Phase 2 Success Criteria:**
- ✅ V2 repository interfaces created in `internal/v2/repository/`
- ✅ V2 repository implementation created using new schema
- ✅ Existing e2e tests pass with v2 repository
- ✅ Data persists correctly in new schema format

---

### Phase 3: Data Source Layer Migration  
**Goal:** Update ESPN client to work with new domain entities

#### Step 3.1: Create V2 Data Source Implementation
**File: `internal/v2/external/espn_client.go`**
```go
package external

import (
    "your-app/internal/v2/domain"
    "your-app/internal/v2/repository"
)

// NFL-specific implementation of DataSource
type NFLDataSource struct {
    client    *http.Client
    ratingSvc repository.RatingService
}

func NewNFLDataSource(client *http.Client, ratingSvc repository.RatingService) *NFLDataSource {
    return &NFLDataSource{client: client, ratingSvc: ratingSvc}
}

func (n *NFLDataSource) GetSport() domain.Sport { 
    return domain.SportNFL 
}

func (n *NFLDataSource) ListLatest() ([]domain.Competition, error) {
    // Convert ESPN data to domain.Competition format
    // Can reference existing implementation in internal/external/espn_client.go
}
```

**🛑 VALIDATION STOP:** Run e2e tests to verify ESPN integration works with v2

#### Step 3.2: Create V2 Rating Service
**File: `internal/v2/application/rating_service.go`**
```go
package application

import "your-app/internal/v2/domain"

type OpenAIRatingService struct {
    client openai.Client
}

func NewOpenAIRatingService(client openai.Client) *OpenAIRatingService {
    return &OpenAIRatingService{client: client}
}

func (o *OpenAIRatingService) ProduceRating(comp domain.Competition) (domain.Rating, error) {
    // Generate excitement rating using current logic
    // Can reference existing implementation in internal/application/rating_service.go
}
```

**🛑 VALIDATION STOP:** Run full e2e test suite with v2 architecture

**✅ Phase 3 Success Criteria:**
- ✅ V2 data source created in `internal/v2/external/`
- ✅ V2 rating service created in `internal/v2/application/`
- ✅ End-to-end tests pass: ESPN → Rating → Database flow
- ✅ Template rendering works with v2 domain entities

---

### Phase 2.5: Backfill Service 
**Goal:** Make backfill a first-class citizen with gap detection and efficient processing

#### Backfill Service Architecture

**Clean Architecture Integration:**
- **Application Layer**: New use cases for gap analysis and incremental processing
- **Existing Interfaces**: Reuse current repository and external service contracts
- **Domain Layer**: No changes needed to existing entities
- **Infrastructure**: Minor enhancements to repository queries

**Core Use Cases:**
- `use_cases/backfill/analyze_missing_data.go` - Gap detection between database and data sources
- `use_cases/backfill/backfill_missing_data.go` - Incremental processing of missing competitions  
- `use_cases/backfill/get_backfill_status.go` - Progress tracking and completion status

**Key Concepts:**
- **Gap Detection**: Compare database state vs data source availability
- **Incremental Processing**: Only fetch and process missing competitions
- **Progress Tracking**: Monitor completion status across seasons/periods
- **Failure Recovery**: Resume from interruption points

#### Service Responsibilities

**BackfillService Interface:**
- Analyze gaps between database and data sources
- Orchestrate incremental backfill operations
- Track completion status and progress
- Handle partial failures gracefully

**Enhanced Repository Interface:**
- Query existing periods/competitions by sport/season
- Count competitions per period for validation
- Identify missing ratings by type
- Support efficient gap analysis queries

**Extended DataSource Interface:**
- List available periods from external APIs
- Report expected game counts per period
- Validate data availability before processing

#### API Design Philosophy

**Status & Discovery:**
- `/api/backfill/status` - Overall completion status
- `/api/backfill/missing` - Gap analysis results

**Processing Operations:**
- `/api/backfill/incremental` - Process only missing data
- `/api/backfill/period` - Target specific period

**Administrative:**
- `/api/backfill/reset` - Clear specific periods for re-processing

#### Processing Strategy

**Gap Analysis Workflow:**
1. Query database for existing competitions by sport/season
2. Query data source for available periods/competitions  
3. Compare sets to identify missing elements
4. Validate game counts for completeness detection
5. Prioritize gaps by importance (recent vs historical)

**Incremental Backfill Workflow:**
1. Process gaps in priority order
2. Generate ratings only for new competitions
3. Continue processing on individual failures
4. Update progress tracking throughout
5. Provide resumable checkpoints

#### Benefits Over Current Approach
- ⚡ **Efficiency**: Only processes missing data
- 💰 **Cost Optimization**: Minimal redundant API calls
- 🛡️ **Resilience**: Partial failure recovery
- 📊 **Observability**: Clear gap visibility  
- 🎯 **Precision**: Surgical data operations
- 🔄 **Resumability**: Interrupt and restart capability

#### Migration Strategy
- **Direct replacement** of existing backfill code as we implement
- **Update tests in parallel** with interface changes
- **Delete legacy backfill code** immediately after new implementation
- **Measure efficiency improvements** against git history baselines

**✅ Success Criteria:** Efficient backfill service operational with measurable performance improvements

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

**✅ Success Criteria:** All 2024 NFL data regenerated in clean format using v2 architecture

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

**✅ Success Criteria:** Rich analytics available using v2 architecture, fast query performance

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

**✅ Success Criteria:** Production-ready v2 system with optimal performance

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
- 🏗️ **Clean architecture** ready for future sport expansion
- 📈 **Extensible rating system** ready for additional rating types
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

### Minimal Risks (Direct Migration)
- ✅ **Git safety**: All changes tracked in version control
- ✅ **Incremental commits**: Small, testable changes per commit
- ✅ **Rollback capability**: Git revert to any previous state
- ✅ **Test-driven changes**: Update tests before breaking changes

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
- ✅ **Clean architecture** foundation for future expansion
- ✅ **Maintainable** codebase

---

## 🛠️ Implementation Strategy

### Development Approach
1. **V2 directory structure** - Create `internal/v2/` for new architecture
2. **Reference old code** - Keep existing code available during development
3. **Incremental migration** - Move endpoints to v2 implementations gradually
4. **Adapt e2e tests** to work with v2 as we migrate each component

### Testing Strategy
**Leverage Existing High-Level E2E Tests:**

The existing `main_test.go` provides comprehensive end-to-end tests that we'll adapt:

- `TestRealESPNEndToEnd()` - Full data flow from ESPN API to database with mock OpenAI
- `TestRealESPNWithUseCases()` - Use case integration testing
- `TestRealESPNClient()` - External API integration validation

**High-Level Testing Approach:**
- **Input/Output focused** - Test complete workflows, not individual functions
- **Real integrations** - Use actual ESPN API and database operations
- **Template validation** - Verify web page rendering with golden master tests
- **Regression protection** - Existing `run_regression_tests.sh` framework

**V2 Migration Testing:**
1. **Adapt existing e2e tests** to work with v2 domain entities as we build
2. **Focus on NFL e2e scenarios** (no multi-sport expansion initially)
3. **Performance benchmarks** on new schema using real data flows
4. **Keep high-level focus** - Test complete request → response cycles

---

## 📞 Next Steps

1. ✅ **Approve fresh start approach** (COMPLETED)
2. 🔄 **Set up new development branch**
3. 🏗️ **Begin Phase 1: Clean schema implementation**
4. 📊 **Create performance benchmarks**
5. 🚀 **Execute regeneration plan**

---

*This direct migration approach delivers a clean, scalable NFL platform with git-based safety and efficient iterative development, ready for future expansion.*

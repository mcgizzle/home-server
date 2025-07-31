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

## 🔑 **CORE PRINCIPLE: COMPLETE V1/V2 SEPARATION**

### **V2 Independence Requirements**
- ✅ **V2 has its own schema** - No shared tables with V1
- ✅ **V2 has its own entities** - No V1 domain imports in V2 code
- ✅ **V2 has its own use cases** - Direct ESPN → V2 entity conversion
- ✅ **V2 has its own repositories** - Pure V2 CRUD operations
- ✅ **V1 continues working** - Existing endpoints remain functional during migration
- ✅ **Gradual replacement** - Replace V1 use cases one by one in existing endpoints

### **What V2 Can Use**
- ✅ External services (ESPN client, OpenAI)
- ✅ Infrastructure (database connection, HTTP client)
- ✅ Utilities (logging, configuration)

### **What V2 Cannot Use**
- ❌ V1 domain entities (`domain.Result`, `domain.Game`, etc.)
- ❌ V1 repository interfaces
- ❌ V1 use cases
- ❌ V1 converters or adapters

---

## 🏗️ Target Architecture

### Core Principles
- **Clean normalized** database schema
- **AI excitement rating** (single rating type initially)
- **Structured storage** with JSON flexibility where needed
- **ESPN data source** for NFL games
- **Powerful analytics** capabilities built-in
- **COMPLETE V1/V2 SEPARATION** at all layers

### Key New Entities
```go
// V2 Domain - COMPLETELY SEPARATE from V1
type Sport string                    // nfl (only sport initially)
type Competition struct             // NFL game entity - NO V1 dependencies
type RatingType string              // ai_excitement (only rating type initially)
type Rating struct                  // AI excitement rating with metadata - NO V1 dependencies
type Result struct                  // Competition + excitement rating - NO V1 dependencies
```

---

## 📋 Implementation Phases

### Phase 1: V2 Schema & Domain - PURE V2 FOUNDATION
**Goal:** Create completely separate V2 foundation with zero V1 dependencies

#### Step 1.1: Create V2-Only Schema
```sql
-- V2 SCHEMA - COMPLETELY SEPARATE FROM V1
-- V1 table "results" remains untouched and functional

-- Sports catalog (V2 only)
CREATE TABLE sports (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

-- Universal competitions (V2 only)
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

-- Teams across all sports (V2 only)
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    sport_id TEXT REFERENCES sports(id),
    logo_url TEXT,
    UNIQUE(name, sport_id)
);

-- Competition participants (V2 only)
CREATE TABLE competition_teams (
    competition_id TEXT REFERENCES competitions(id),
    team_id TEXT REFERENCES teams(id),
    home_away TEXT NOT NULL,
    score REAL DEFAULT 0,
    stats JSON, -- JSON for sport-specific stats
    PRIMARY KEY(competition_id, team_id)
);

-- Ratings (V2 only)
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

-- Rich competition data (V2 only)
CREATE TABLE competition_details (
    competition_id TEXT PRIMARY KEY REFERENCES competitions(id),
    play_by_play JSON,             -- JSON array
    metadata JSON                  -- JSON for sport-specific data
);
```

**🛑 VALIDATION STOP:** V2 schema completely separate from V1, both schemas coexist

#### Step 1.2: Create V2-Only Domain Entities
Create new directory structure with ZERO V1 imports:
```
internal/
├── v2/                          # V2 ONLY - NO V1 DEPENDENCIES
│   ├── domain/
│   │   ├── entities.go          # PURE V2 entities - NO V1 imports
│   │   └── constants.go         # V2 constants only
│   ├── repository/
│   ├── application/
│   └── external/
└── (existing v1 code remains untouched)
```

**File: `internal/v2/domain/entities.go`**
```go
// PURE V2 DOMAIN - NO V1 IMPORTS ALLOWED
package domain

import "time" // Only standard library imports allowed

// V2 Sport-agnostic domain - COMPLETELY SEPARATE from V1
type Sport string

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
    RatingTypeExcitement RatingType = "excitement"  // Only excitement rating initially
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

// NO V1 DOMAIN IMPORTS - PURE V2 ENTITIES ONLY
```

**🛑 VALIDATION STOP:** V2 entities compile with zero V1 dependencies

**✅ Phase 1 Success Criteria:** 
- ✅ V2 database schema created (separate from V1)
- ✅ V2 domain entities created with ZERO V1 imports
- ✅ V1 system continues working unchanged
- ✅ Both V1 and V2 schemas coexist in same database

---

### Phase 2: V2 Repository Layer - PURE V2 DATA ACCESS
**Goal:** Create V2 repository layer with zero V1 dependencies

#### Step 2.1: Create V2-Only Repository Interfaces
**File: `internal/v2/repository/competition_repository.go`**
```go
// PURE V2 REPOSITORY - NO V1 IMPORTS
package repository

import "github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain" // V2 ONLY

// V2 Competition repository - WORKS ONLY WITH V2 ENTITIES
type CompetitionRepository interface {
    SaveCompetition(comp domain.Competition) error
    FindByPeriod(season, period, periodType string, sport domain.Sport) ([]domain.Competition, error)
    GetAvailablePeriods(sport domain.Sport) ([]domain.Date, error)
}

// NO V1 DOMAIN IMPORTS ALLOWED
```

**🛑 VALIDATION STOP:** V2 interfaces compile with zero V1 dependencies

#### Step 2.2: Create V2-Only Repository Implementation
**File: `internal/v2/repository/sqlite_v2_repository.go`**
```go
// PURE V2 REPOSITORY IMPLEMENTATION - NO V1 DEPENDENCIES
package repository

import (
    "database/sql"
    "github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain" // V2 ONLY
)

type SQLiteV2Repository struct {
    db *sql.DB
}

func NewSQLiteV2Repository(db *sql.DB) *SQLiteV2Repository {
    return &SQLiteV2Repository{db: db}
}

// WORKS ONLY WITH V2 TABLES: competitions, teams, ratings, etc.
// NEVER TOUCHES V1 "results" TABLE
func (r *SQLiteV2Repository) SaveCompetition(comp domain.Competition) error {
    // Implementation using V2 tables ONLY
    // INSERT INTO competitions, teams, ratings, competition_teams
    // NO V1 table access
}

// NO V1 IMPORTS OR V1 TABLE ACCESS
```

**🛑 VALIDATION STOP:** V2 repository works with V2 schema only, V1 untouched

**✅ Phase 2 Success Criteria:**
- ✅ V2 repository interfaces created with ZERO V1 dependencies
- ✅ V2 repository implementation uses V2 tables ONLY
- ✅ V1 "results" table completely untouched
- ✅ V2 data persists in V2 schema correctly

---

### Phase 3: V2 Use Cases - PURE V2 BUSINESS LOGIC
**Goal:** Create V2 use cases that work directly with ESPN and V2 entities

#### Step 3.1: Create V2-Only Fetch Use Case
**File: `internal/v2/application/use_cases/fetch_latest_competitions.go`**
```go
// PURE V2 USE CASE - NO V1 IMPORTS
package use_cases

import (
    "github.com/mcgizzle/home-server/apps/cloud/internal/external"      // ESPN client OK
    "github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"     // V2 ONLY
    "github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository" // V2 ONLY
)

type FetchLatestCompetitionsUseCase interface {
    Execute(sportID string) ([]domain.Competition, error)
}

// CONVERTS DIRECTLY: ESPN API → V2 Entities
// NO V1 DOMAIN CONVERSIONS
func (uc *fetchLatestCompetitionsUseCase) Execute(sportID string) ([]domain.Competition, error) {
    // ESPN API → external.EventResponse
    // external.Competitions → domain.Competition (V2)
    // external.TeamResponse → domain.Team (V2)
    // NO V1 domain.Game OR domain.Result INVOLVED
}
```

#### Step 3.2: Create V2-Only Save Use Case
**File: `internal/v2/application/use_cases/save_competitions.go`**
```go
// PURE V2 SAVE USE CASE - NO V1 IMPORTS
package use_cases

import (
    "github.com/mcgizzle/home-server/apps/cloud/internal/v2/domain"     // V2 ONLY
    "github.com/mcgizzle/home-server/apps/cloud/internal/v2/repository" // V2 ONLY
)

type SaveCompetitionsUseCase interface {
    Execute(competitions []domain.Competition) error // V2 ENTITIES ONLY
}

// SAVES V2 ENTITIES TO V2 TABLES ONLY
func (uc *saveCompetitionsUseCase) Execute(competitions []domain.Competition) error {
    // domain.Competition (V2) → V2 tables
    // NO V1 INVOLVEMENT
}
```

**🛑 VALIDATION STOP:** V2 use cases work end-to-end with zero V1 involvement

**✅ Phase 3 Success Criteria:**
- ✅ V2 fetch use case: ESPN API → V2 entities directly
- ✅ V2 save use case: V2 entities → V2 tables directly  
- ✅ V2 template use case: V2 entities → UI directly
- ✅ ZERO V1 domain imports in any V2 use case
- ✅ Complete V2 pipeline works independently

---

### Phase 4: Gradual V1→V2 Use Case Replacement
**Goal:** Replace V1 use cases in existing endpoints while maintaining functionality

#### Phase 4.1: Template Data Migration
Replace V1 template logic with V2 in existing `/` endpoint:
```go
// IN main.go - EXISTING ENDPOINT
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    // BEFORE: V1 use case
    // templateData := v1GetTemplateDataUseCase.Execute(season, week, seasonType)
    
    // AFTER: V2 use case (same endpoint, different implementation)
    templateData := v2GetTemplateDataUseCase.Execute("nfl", season, week, periodType)
    
    // Same template rendering, same user experience
})
```

#### Phase 4.2: Background Events Migration
Replace V1 background fetching with V2:
```go
// IN main.go - EXISTING BACKGROUND FUNCTION
func backgroundLatestEvents() {
    // BEFORE: V1 fetch → V1 save
    // results := v1FetchUseCase.Execute()
    // v1SaveUseCase.Execute(results)
    
    // AFTER: V2 fetch → V2 save (same function, different implementation)
    competitions := v2FetchUseCase.Execute("nfl")
    v2SaveUseCase.Execute(competitions)
}
```

#### Phase 4.3: Backfill Migration
Replace V1 backfill logic with V2 in existing `/backfill` endpoint:
```go
// IN main.go - EXISTING ENDPOINT
http.HandleFunc("/backfill", func(w http.ResponseWriter, r *http.Request) {
    // BEFORE: V1 specific fetch → V1 save
    // results := v1FetchSpecificUseCase.Execute(season, week, seasonType)
    // v1SaveUseCase.Execute(results)
    
    // AFTER: V2 specific fetch → V2 save (same endpoint, different implementation)
    competitions := v2FetchSpecificUseCase.Execute("nfl", season, week, periodType)
    v2SaveUseCase.Execute(competitions)
})
```

**🛑 VALIDATION STOP:** All existing endpoints work with V2 implementation

**✅ Phase 4 Success Criteria:**
- ✅ All existing HTTP endpoints functional with V2 implementation
- ✅ V1 use cases completely removed from main.go
- ✅ V2 use cases handle all business logic
- ✅ Same user experience, V2 architecture underneath

---

### Phase 5: V1 Cleanup & V2 Production
**Goal:** Remove V1 code and optimize V2 for production

#### Step 5.1: Remove V1 Code
```bash
# Delete V1 implementation files
rm internal/domain/entities.go                    # V1 entities
rm internal/repository/result_repository.go       # V1 repository  
rm internal/application/use_cases/*.go            # V1 use cases
rm internal/application/rating_service.go         # V1 rating service

# Drop V1 table
DROP TABLE results;
```

#### Step 5.2: V2 Production Optimization
```sql
-- Add strategic indexes for V2 tables
CREATE INDEX idx_competitions_sport_season ON competitions(sport_id, season);
CREATE INDEX idx_ratings_score ON ratings(score DESC);
CREATE INDEX idx_competition_teams_team ON competition_teams(team_id);
```

**✅ Phase 5 Success Criteria:**
- ✅ V1 code completely removed
- ✅ V1 database table dropped
- ✅ Pure V2 architecture in production
- ✅ Optimal performance with V2 schema

---

## 🛠️ Implementation Strategy

### Development Approach - SEPARATION ENFORCED
1. **V2 directory isolation** - Create `internal/v2/` with ZERO V1 imports
2. **Independent development** - V2 components never reference V1 code
3. **Parallel operation** - V1 and V2 coexist without interaction
4. **Gradual endpoint migration** - Replace V1 use cases in existing endpoints
5. **V1 cleanup** - Remove V1 code only after V2 handles all functionality

### Testing Strategy - BOTH SYSTEMS WORK
**Preserve V1 Tests + Add V2 Tests:**

**V1 Tests Continue Working:**
- `TestRealESPNEndToEnd()` - Continues using V1 implementation
- Validates V1 system remains functional during V2 development

**V2 Tests Added:**
- `TestV2ESPNEndToEnd()` - Pure V2 test: ESPN → V2 entities → V2 database  
- `TestV2TemplateGeneration()` - V2 template data generation
- `TestV2Migration()` - V2 schema and data operations

**Migration Tests:**
- `TestEndpointMigration()` - Verify endpoints work with V2 implementation
- Tests run against same endpoints, comparing V1 vs V2 behavior

### Benefits of Complete Separation
- ✅ **Zero migration risk** - V1 continues working during V2 development
- ✅ **Independent testing** - Each system can be validated separately  
- ✅ **Clean rollback** - Can revert to V1 at any point
- ✅ **Parallel development** - V2 features can be built without affecting V1
- ✅ **Clear architecture** - No hybrid V1/V2 complexity

---

## 📞 Next Steps

1. ✅ **Approve complete V1/V2 separation approach** 
2. 🔄 **V2 schema implementation** (Phase 1)
3. 🏗️ **V2 repository layer** (Phase 2)  
4. 📊 **V2 use case implementation** (Phase 3)
5. 🚀 **Gradual endpoint migration** (Phase 4)

---

*This complete separation approach ensures zero risk to existing functionality while building a clean, modern V2 architecture that can eventually replace V1 entirely.*

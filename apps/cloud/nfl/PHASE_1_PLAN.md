# Phase 1: Extract Entities & Core Business Logic

## Overview
Phase 1 focuses on extracting the core business entities and domain logic from the monolithic `main.go` file. This establishes the foundation for clean architecture by separating concerns and creating clear boundaries.

## Goals
- Extract core business entities (Game, Team, Rating, etc.)
- Separate domain logic from infrastructure concerns
- Maintain backward compatibility
- Keep all existing functionality working

## Files to Create

### 1. `internal/domain/entities.go`
```go
// Core business entities
type Game struct { ... }
type Team struct { ... }
type Rating struct { ... }
type Result struct { ... }
type Date struct { ... }
```

### 2. `internal/domain/rating_service.go`
```go
// Business logic for generating ratings
type RatingService struct { ... }
func (r *RatingService) GenerateRating(game Game) (Rating, error) { ... }
```

### 3. `internal/domain/game_service.go`
```go
// Business logic for processing games
type GameService struct { ... }
func (g *GameService) ProcessGame(event EventResponse) (*Game, error) { ... }
```

## Refactoring Steps

### Step 1: Extract Entities
1. Create `internal/domain/entities.go`
2. Move all struct definitions from `main.go`
3. Update imports in `main.go`
4. Run regression tests ✅

### Step 2: Extract Rating Logic
1. Create `internal/domain/rating_service.go`
2. Move `produceRating()` function to `RatingService.GenerateRating()`
3. Update `main.go` to use the service
4. Run regression tests ✅

### Step 3: Extract Game Processing Logic
1. Create `internal/domain/game_service.go`
2. Move `getTeamAndScore()` function to `GameService.ProcessGame()`
3. Update `main.go` to use the service
4. Run regression tests ✅

### Step 4: Extract Constants
1. Create `internal/domain/constants.go`
2. Move season type constants
3. Update imports in `main.go`
4. Run regression tests ✅

## Directory Structure After Phase 1
```
apps/cloud/nfl/
├── main.go (simplified)
├── internal/
│   └── domain/
│       ├── entities.go
│       ├── rating_service.go
│       ├── game_service.go
│       └── constants.go
├── testdata/
├── static/
└── data/
```

## Success Criteria
- [ ] All existing functionality works identically
- [ ] Regression tests pass
- [ ] Core business logic is separated from infrastructure
- [ ] Entities are clearly defined and reusable
- [ ] No breaking changes to external interfaces

## Testing Strategy
After each step:
1. Run `./run_regression_tests.sh`
2. Verify the application still works
3. Check that no functionality is lost

## Next Phase Preview
Phase 2 will focus on extracting repositories and external service clients, further separating infrastructure concerns from business logic.

---

**Status**: 🚧 IN PROGRESS  
**Dependencies**: Phase 0 Complete ✅  
**Risk Level**: Low - Pure extraction, no interface changes 
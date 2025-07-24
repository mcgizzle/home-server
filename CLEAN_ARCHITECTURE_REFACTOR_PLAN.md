# NFL Ratings App - Clean Architecture Refactor Plan

## Overview
Refactor the monolithic `main.go` (915 lines) into a clean architecture following Uncle Bob's principles while maintaining 100% backward compatibility and preserving the existing database schema.

## Architecture Layers (Uncle Bob's Clean Architecture)

```
┌─────────────────────────────────────────┐
│           Frameworks & Drivers          │  
│  (HTTP Server, SQLite, External APIs)   │
├─────────────────────────────────────────┤
│         Interface Adapters              │
│   (Controllers, Repositories, Clients)  │
├─────────────────────────────────────────┤
│            Use Cases                    │
│     (Application Business Rules)        │
├─────────────────────────────────────────┤
│             Entities                    │
│      (Enterprise Business Rules)       │
└─────────────────────────────────────────┘
```

## Step-by-Step Refactor Plan

### Phase 0: Lean Edge Testing (CRITICAL FIRST STEP)
**Goal**: Create minimal, focused tests that validate the HTTP boundaries and data contracts only

#### Step 0.1: Create Lean Test Setup
- [ ] Create `main_test.go` with minimal test infrastructure:
  ```go
  // Simple OpenAI mock that returns deterministic ratings
  type mockOpenAI struct{}
  func (m *mockOpenAI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
      // Return fixed rating response
  }
  
  // Test database in memory
  func setupTestDB() *sql.DB {
      // In-memory SQLite for isolation
  }
  ```

#### Step 0.2: Golden Master Test - TemplateData Contract
- [ ] Create the core regression test:
  ```go
  func TestTemplateData_GoldenMaster(t *testing.T) {
      // Use real ESPN API + mock OpenAI
      // Capture exact TemplateData structure passed to template
      // Save as JSON golden file
      // Future runs compare against golden file
  }
  ```

#### Step 0.3: HTTP Edge Tests
- [ ] Test only the HTTP boundaries:
  ```go
  func TestHTTP_MainPage_WithData(t *testing.T) {
      // GET / → HTML response with game data
  }
  
  func TestHTTP_MainPage_EmptyDB(t *testing.T) {
      // GET / → "No data available yet" response
  }
  
  func TestHTTP_SpecificWeek(t *testing.T) {
      // GET /?season=2024&week=10&seasontype=2 → correct data
  }
  
  func TestHTTP_StaticFiles(t *testing.T) {
      // GET /static/main.css → CSS content
  }
  ```

#### Step 0.4: Test Execution
- [ ] Keep it simple:
  - Run tests before/after each refactor phase
  - Use specific historical week (e.g., "2024 Week 10") for deterministic ESPN data
  - Mock only OpenAI to avoid costs
  - No complex infrastructure or helpers

### Phase 1: Extract Entities (Domain Models)
**Goal**: Move all business entities to separate files without changing logic

#### Step 1.1: Create `internal/domain/entities/` directory
- [ ] Create folder structure: `internal/domain/entities/`
- [ ] Move all struct definitions to separate files:
  - `game.go` - Game, TeamResult, DetailsItem
  - `rating.go` - Rating 
  - `result.go` - Result
  - `event.go` - EventResponse, Competitions, Competitors
  - `team.go` - TeamResponse
  - `api_responses.go` - All ESPN API response structs

#### Step 1.2: Update imports in main.go
- [ ] Add imports for new entity packages
- [ ] Verify all tests pass and app runs identically

### Phase 2: Extract Repository Layer
**Goal**: Separate data access from business logic

#### Step 2.1: Create Repository Interfaces
- [ ] Create `internal/domain/repositories/` directory
- [ ] Create `result_repository.go` with interface:
  ```go
  type ResultRepository interface {
      Save(results []entities.Result) error
      FindByWeek(season, week, seasonType string) ([]entities.Result, error)
      FindAllDates() ([]entities.Date, error)
  }
  ```

#### Step 2.2: Create SQLite Repository Implementation
- [ ] Create `internal/infrastructure/persistence/` directory
- [ ] Create `sqlite_result_repository.go` implementing the interface
- [ ] Move all database-related functions: `initDb()`, `saveResults()`, `loadResults()`, `loadDates()`
- [ ] Update main.go to use repository interface

#### Step 2.3: Add Database Configuration
- [ ] Create `internal/infrastructure/config/database.go`
- [ ] Move database connection logic and path configuration

### Phase 3: Extract External Service Clients
**Goal**: Isolate external API dependencies

#### Step 3.1: Create ESPN Client Interface
- [ ] Create `internal/domain/services/` directory
- [ ] Create `espn_service.go` with interface:
  ```go
  type ESPNService interface {
      GetLatestEvents() (*entities.LatestEvents, error)
      GetSpecificEvents(season, week, seasonType string) (*entities.SpecificEvents, error)
      GetEvent(ref string) (*entities.EventResponse, error)
      GetEventById(id string) (*entities.EventResponse, error)
  }
  ```

#### Step 3.2: Create OpenAI Client Interface
- [ ] Create `rating_service.go` with interface:
  ```go
  type RatingService interface {
      GenerateRating(game entities.Game) (*entities.Rating, error)
  }
  ```

#### Step 3.3: Implement External Service Clients
- [ ] Create `internal/infrastructure/external/` directory
- [ ] Create `espn_client.go` implementing ESPNService
- [ ] Create `openai_client.go` implementing RatingService
- [ ] Move all HTTP client logic and API calls

### Phase 4: Create Use Cases (Application Logic)
**Goal**: Extract business logic into clean use cases

#### Step 4.1: Create Use Case Interfaces
- [ ] Create `internal/application/usecases/` directory
- [ ] Create interfaces for main operations:
  ```go
  type FetchResultsUseCase interface {
      FetchCurrentWeek() ([]entities.Result, error)
      FetchSpecificWeek(season, week, seasonType string) ([]entities.Result, error)
  }
  
  type LoadResultsUseCase interface {
      LoadByWeek(season, week, seasonType string) ([]entities.Result, error)
      LoadAvailableDates() ([]entities.Date, error)
      LoadMostRecentWeek() (*entities.Result, error)
  }
  ```

#### Step 4.2: Implement Use Cases
- [ ] Create `fetch_results_usecase.go` 
- [ ] Create `load_results_usecase.go`
- [ ] Move business logic from handlers: `fetchResultsForThisWeek()`, `fetchResults()`
- [ ] Add proper error handling and logging

#### Step 4.3: Create Background Service Use Case
- [ ] Create `background_sync_usecase.go`
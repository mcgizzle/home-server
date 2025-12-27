package use_cases

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/external"
	"github.com/mcgizzle/home-server/apps/cloud/internal/repository"
)

// GenerateSentimentRatingUseCase defines the business operation for generating sentiment ratings
type GenerateSentimentRatingUseCase interface {
	Execute(competitionID string) error
}

// generateSentimentRatingUseCase implements GenerateSentimentRatingUseCase
type generateSentimentRatingUseCase struct {
	competitionRepo repository.CompetitionRepository
	sentimentRepo   repository.SentimentRatingRepository
	redditClient    external.RedditClient
	sentimentSvc    external.SentimentService
}

// NewGenerateSentimentRatingUseCase creates a new instance of GenerateSentimentRatingUseCase
func NewGenerateSentimentRatingUseCase(
	competitionRepo repository.CompetitionRepository,
	sentimentRepo repository.SentimentRatingRepository,
	redditClient external.RedditClient,
	sentimentSvc external.SentimentService,
) GenerateSentimentRatingUseCase {
	return &generateSentimentRatingUseCase{
		competitionRepo: competitionRepo,
		sentimentRepo:   sentimentRepo,
		redditClient:    redditClient,
		sentimentSvc:    sentimentSvc,
	}
}

// Execute generates a sentiment rating for a competition by analyzing Reddit comments
func (uc *generateSentimentRatingUseCase) Execute(competitionID string) error {
	// Get the competition details
	competition, err := uc.competitionRepo.GetCompetitionByID(competitionID)
	if err != nil {
		return fmt.Errorf("failed to load competition %s: %w", competitionID, err)
	}

	// Check if competition is final
	if competition.Status != "final" && competition.Status != "completed" {
		log.Printf("Skipping sentiment analysis for competition %s - status is %s (not final)", competitionID, competition.Status)
		return fmt.Errorf("competition is not final yet (status: %s)", competition.Status)
	}

	// Extract team names from the competition
	if len(competition.Teams) < 2 {
		return fmt.Errorf("competition must have at least 2 teams, got %d", len(competition.Teams))
	}

	team1Name := competition.Teams[0].Team.Name
	team2Name := competition.Teams[1].Team.Name

	// Determine game date for Reddit search time window
	gameDate := time.Now()
	if competition.StartTime != nil {
		gameDate = *competition.StartTime
	}

	log.Printf("Searching for post-game thread: %s vs %s (game date: %s)", team1Name, team2Name, gameDate.Format("2006-01-02"))

	// Search for post-game thread on Reddit
	posts, err := uc.redditClient.SearchPostGameThread(team1Name, team2Name, gameDate)
	if err != nil {
		return fmt.Errorf("failed to search Reddit for post-game thread: %w", err)
	}

	if len(posts) == 0 {
		return fmt.Errorf("no post-game thread found on Reddit for %s vs %s", team1Name, team2Name)
	}

	// Use the first (most recent) post
	post := posts[0]
	log.Printf("Found post-game thread: %s (URL: https://reddit.com%s)", post.Title, post.Permalink)

	// Fetch comments from the thread
	threadURL := fmt.Sprintf("https://reddit.com%s", post.Permalink)
	redditComments, err := uc.redditClient.GetThreadComments(threadURL)
	if err != nil {
		return fmt.Errorf("failed to fetch comments from Reddit thread: %w", err)
	}

	if len(redditComments) == 0 {
		return fmt.Errorf("no comments found in post-game thread")
	}

	log.Printf("Fetched %d comments from Reddit thread", len(redditComments))

	// Extract comment bodies for sentiment analysis
	commentBodies := make([]string, 0, len(redditComments))
	for _, comment := range redditComments {
		// Filter out deleted/removed comments and very short comments
		if comment.Body != "[deleted]" && comment.Body != "[removed]" && len(strings.TrimSpace(comment.Body)) > 10 {
			commentBodies = append(commentBodies, comment.Body)
		}
	}

	if len(commentBodies) == 0 {
		return fmt.Errorf("no valid comments found after filtering")
	}

	log.Printf("Analyzing sentiment from %d valid comments", len(commentBodies))

	// Generate sentiment analysis
	sentimentRating, err := uc.sentimentSvc.AnalyzeSentiment("reddit", threadURL, commentBodies)
	if err != nil {
		return fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	log.Printf("Generated sentiment rating: score=%d, sentiment=%s, highlights=%v",
		sentimentRating.Score, sentimentRating.Sentiment, sentimentRating.Highlights)

	// Save sentiment rating to database
	err = uc.sentimentRepo.SaveSentimentRating(sentimentRating, competitionID)
	if err != nil {
		return fmt.Errorf("failed to save sentiment rating: %w", err)
	}

	log.Printf("Successfully saved sentiment rating for competition %s", competitionID)
	return nil
}

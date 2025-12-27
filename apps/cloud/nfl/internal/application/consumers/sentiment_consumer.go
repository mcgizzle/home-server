package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	usecases "github.com/mcgizzle/home-server/apps/cloud/internal/application/use_cases"
	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/queue"
)

// SentimentConsumer handles sentiment analysis jobs from the job queue
type SentimentConsumer struct {
	generateUseCase usecases.GenerateSentimentRatingUseCase
	jobQueue        queue.JobQueue
}

// NewSentimentConsumer creates a new sentiment consumer
func NewSentimentConsumer(
	generateUseCase usecases.GenerateSentimentRatingUseCase,
	jobQueue queue.JobQueue,
) *SentimentConsumer {
	return &SentimentConsumer{
		generateUseCase: generateUseCase,
		jobQueue:        jobQueue,
	}
}

// SentimentJobPayload represents the payload for sentiment analysis jobs
type SentimentJobPayload struct {
	CompetitionID string `json:"competition_id"`
}

// Start begins processing sentiment analysis jobs
// This method blocks until the context is cancelled
func (c *SentimentConsumer) Start(ctx context.Context) {
	log.Println("Starting sentiment consumer...")

	// Process sentiment_analysis jobs
	c.jobQueue.Process(ctx, "sentiment_analysis", func(job queue.Job) error {
		log.Printf("Processing sentiment job: %s for competition from payload", job.ID)

		// Unmarshal job payload
		var payload SentimentJobPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			log.Printf("Error unmarshaling sentiment job payload: %v", err)
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		if payload.CompetitionID == "" {
			log.Printf("Error: sentiment job missing competition_id")
			return fmt.Errorf("competition_id is required in job payload")
		}

		log.Printf("Generating sentiment rating for competition: %s", payload.CompetitionID)

		// Execute the use case
		if err := c.generateUseCase.Execute(payload.CompetitionID); err != nil {
			log.Printf("Error generating sentiment rating for competition %s: %v", payload.CompetitionID, err)
			return fmt.Errorf("failed to generate sentiment rating: %w", err)
		}

		log.Printf("Successfully processed sentiment job for competition: %s", payload.CompetitionID)
		return nil
	})
}

package queue_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mcgizzle/home-server/apps/cloud/internal/infrastructure/queue"
)

// This example demonstrates how to use the SimpleQueue for sentiment analysis jobs.
func ExampleSimpleQueue() {
	// Create a new queue with buffer size of 100
	q := queue.NewSimpleQueue(100)
	defer q.Shutdown()

	// Define a handler for sentiment analysis jobs
	sentimentHandler := func(job queue.Job) error {
		// Decode the payload
		var data map[string]string
		if err := json.Unmarshal(job.Payload, &data); err != nil {
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		// Process the sentiment analysis
		fmt.Printf("Processing sentiment for text: %s\n", data["text"])
		return nil
	}

	// Start processing sentiment analysis jobs in the background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go q.Process(ctx, "sentiment_analysis", sentimentHandler)

	// Schedule an immediate job
	payload, _ := json.Marshal(map[string]string{
		"text": "The game was amazing!",
	})

	immediateJob := queue.Job{
		ID:           "job-1",
		Type:         "sentiment_analysis",
		Payload:      payload,
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	}

	if err := q.Schedule(immediateJob); err != nil {
		fmt.Printf("Error scheduling job: %v\n", err)
		return
	}

	// Schedule a delayed job (e.g., to analyze a post-game thread after the game)
	delayedJob := queue.Job{
		ID:           "job-2",
		Type:         "sentiment_analysis",
		Payload:      payload,
		ScheduledFor: time.Now().Add(1 * time.Second),
		CreatedAt:    time.Now(),
	}

	if err := q.Schedule(delayedJob); err != nil {
		fmt.Printf("Error scheduling delayed job: %v\n", err)
		return
	}

	// Wait a bit for jobs to process
	time.Sleep(2 * time.Second)

	// Output:
	// Processing sentiment for text: The game was amazing!
	// Processing sentiment for text: The game was amazing!
}

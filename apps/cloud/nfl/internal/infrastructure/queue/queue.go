package queue

import (
	"context"
	"time"
)

// Job represents a task to be executed by the queue system.
type Job struct {
	ID           string
	Type         string    // e.g., "sentiment_analysis"
	Payload      []byte    // JSON-encoded job data
	ScheduledFor time.Time
	CreatedAt    time.Time
}

// JobQueue defines the interface for scheduling and processing jobs.
// This interface is designed to be implementation-agnostic, allowing
// for easy swapping between simple in-memory queues and production-ready
// solutions like Asynq or River.
type JobQueue interface {
	// Schedule adds a job to the queue for execution at the specified time.
	Schedule(job Job) error

	// Process registers a handler for a specific job type and begins processing.
	// The handler will be called for each job of the given type.
	// This method blocks until the context is cancelled.
	Process(ctx context.Context, jobType string, handler func(Job) error)
}

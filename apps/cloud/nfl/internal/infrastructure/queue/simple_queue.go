package queue

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SimpleQueue is an in-memory implementation of JobQueue.
// This is an MVP implementation designed for development and testing.
// It should be replaced with a production-ready solution like Asynq or River
// for production deployments.
type SimpleQueue struct {
	jobChan  chan Job
	mu       sync.RWMutex
	timers   map[string]*time.Timer // Track scheduled jobs for cleanup
	handlers map[string]func(Job) error
	wg       sync.WaitGroup
}

// NewSimpleQueue creates a new in-memory job queue.
// bufferSize determines how many jobs can be queued before Schedule blocks.
func NewSimpleQueue(bufferSize int) *SimpleQueue {
	return &SimpleQueue{
		jobChan:  make(chan Job, bufferSize),
		timers:   make(map[string]*time.Timer),
		handlers: make(map[string]func(Job) error),
	}
}

// Schedule adds a job to the queue. If the job is scheduled for a future time,
// it uses time.AfterFunc to delay execution. Otherwise, it's added to the queue immediately.
func (q *SimpleQueue) Schedule(job Job) error {
	if job.ID == "" {
		return fmt.Errorf("job ID is required")
	}
	if job.Type == "" {
		return fmt.Errorf("job type is required")
	}

	now := time.Now()
	delay := job.ScheduledFor.Sub(now)

	// If the job is scheduled for the past or immediate execution
	if delay <= 0 {
		select {
		case q.jobChan <- job:
			return nil
		default:
			return fmt.Errorf("queue is full")
		}
	}

	// Schedule for future execution
	q.mu.Lock()
	timer := time.AfterFunc(delay, func() {
		q.jobChan <- job
		q.mu.Lock()
		delete(q.timers, job.ID)
		q.mu.Unlock()
	})
	q.timers[job.ID] = timer
	q.mu.Unlock()

	return nil
}

// Process registers a handler for a specific job type and begins processing jobs.
// This method blocks until the context is cancelled. Multiple Process calls can be
// made for different job types, each running in its own goroutine.
func (q *SimpleQueue) Process(ctx context.Context, jobType string, handler func(Job) error) {
	// Register the handler
	q.mu.Lock()
	q.handlers[jobType] = handler
	q.mu.Unlock()

	q.wg.Add(1)
	defer q.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobChan:
			// Only process jobs of the registered type
			if job.Type != jobType {
				// Put the job back if it's not our type
				// In a real implementation, this would be handled better
				// (e.g., with separate channels per type)
				select {
				case q.jobChan <- job:
				case <-ctx.Done():
					return
				}
				continue
			}

			// Execute the handler
			if err := handler(job); err != nil {
				// In a production system, this would log the error
				// and potentially retry the job
				fmt.Printf("error processing job %s: %v\n", job.ID, err)
			}
		}
	}
}

// Shutdown cancels all pending scheduled jobs and waits for active jobs to complete.
// This should be called when shutting down the application.
func (q *SimpleQueue) Shutdown() {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Stop all pending timers
	for id, timer := range q.timers {
		timer.Stop()
		delete(q.timers, id)
	}

	// Close the job channel to signal no more jobs will be added
	close(q.jobChan)

	// Wait for all processors to finish
	q.wg.Wait()
}

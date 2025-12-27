package queue

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSimpleQueue_ImmediateExecution(t *testing.T) {
	q := NewSimpleQueue(10)
	defer q.Shutdown()

	var executed bool
	var mu sync.Mutex

	handler := func(job Job) error {
		mu.Lock()
		defer mu.Unlock()
		executed = true
		if job.Type != "test_job" {
			t.Errorf("expected job type 'test_job', got '%s'", job.Type)
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start processing in background
	go q.Process(ctx, "test_job", handler)

	// Schedule a job for immediate execution
	job := Job{
		ID:           "test-1",
		Type:         "test_job",
		Payload:      []byte(`{"test": "data"}`),
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	}

	err := q.Schedule(job)
	if err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	// Wait a bit for the job to be processed
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !executed {
		t.Error("job was not executed")
	}
	mu.Unlock()
}

func TestSimpleQueue_DelayedExecution(t *testing.T) {
	q := NewSimpleQueue(10)
	defer q.Shutdown()

	var executedAt time.Time
	var mu sync.Mutex

	handler := func(job Job) error {
		mu.Lock()
		defer mu.Unlock()
		executedAt = time.Now()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go q.Process(ctx, "delayed_job", handler)

	// Schedule a job for 500ms in the future
	scheduledFor := time.Now().Add(500 * time.Millisecond)
	job := Job{
		ID:           "delayed-1",
		Type:         "delayed_job",
		Payload:      []byte(`{"test": "delayed"}`),
		ScheduledFor: scheduledFor,
		CreatedAt:    time.Now(),
	}

	scheduleTime := time.Now()
	err := q.Schedule(job)
	if err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	// Wait for the job to be executed
	time.Sleep(800 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if executedAt.IsZero() {
		t.Fatal("job was not executed")
	}

	// Verify the job was executed after the scheduled time
	actualDelay := executedAt.Sub(scheduleTime)
	if actualDelay < 450*time.Millisecond {
		t.Errorf("job executed too early: delay was %v, expected at least 450ms", actualDelay)
	}
}

func TestSimpleQueue_MultipleJobTypes(t *testing.T) {
	q := NewSimpleQueue(10)
	defer q.Shutdown()

	var job1Executed, job2Executed bool
	var mu sync.Mutex

	handler1 := func(job Job) error {
		mu.Lock()
		defer mu.Unlock()
		job1Executed = true
		return nil
	}

	handler2 := func(job Job) error {
		mu.Lock()
		defer mu.Unlock()
		job2Executed = true
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start processing both job types
	go q.Process(ctx, "type1", handler1)
	go q.Process(ctx, "type2", handler2)

	// Schedule jobs of both types
	err := q.Schedule(Job{
		ID:           "job1",
		Type:         "type1",
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to schedule job1: %v", err)
	}

	err = q.Schedule(Job{
		ID:           "job2",
		Type:         "type2",
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to schedule job2: %v", err)
	}

	// Wait for jobs to be processed
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !job1Executed {
		t.Error("job1 was not executed")
	}
	if !job2Executed {
		t.Error("job2 was not executed")
	}
}

func TestSimpleQueue_ContextCancellation(t *testing.T) {
	q := NewSimpleQueue(10)
	defer q.Shutdown()

	handler := func(job Job) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)
	go func() {
		q.Process(ctx, "test_job", handler)
		done <- true
	}()

	// Schedule a job
	err := q.Schedule(Job{
		ID:           "test-1",
		Type:         "test_job",
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	// Cancel the context
	cancel()

	// Wait for Process to return
	select {
	case <-done:
		// Success - Process returned after context cancellation
	case <-time.After(2 * time.Second):
		t.Error("Process did not return after context cancellation")
	}
}

func TestSimpleQueue_ValidationErrors(t *testing.T) {
	q := NewSimpleQueue(10)
	defer q.Shutdown()

	tests := []struct {
		name    string
		job     Job
		wantErr bool
	}{
		{
			name: "missing ID",
			job: Job{
				Type:         "test",
				ScheduledFor: time.Now(),
				CreatedAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing Type",
			job: Job{
				ID:           "test-1",
				ScheduledFor: time.Now(),
				CreatedAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "valid job",
			job: Job{
				ID:           "test-1",
				Type:         "test",
				ScheduledFor: time.Now(),
				CreatedAt:    time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := q.Schedule(tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Schedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSimpleQueue_FullQueue(t *testing.T) {
	// Create a queue with buffer size of 1
	q := NewSimpleQueue(1)
	defer q.Shutdown()

	// Fill the queue
	err := q.Schedule(Job{
		ID:           "job-1",
		Type:         "test",
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("first job should succeed: %v", err)
	}

	// Try to add another job immediately (should fail as queue is full)
	err = q.Schedule(Job{
		ID:           "job-2",
		Type:         "test",
		ScheduledFor: time.Now(),
		CreatedAt:    time.Now(),
	})
	if err == nil {
		t.Error("expected error when queue is full, got nil")
	}
}

func TestSimpleQueue_Shutdown(t *testing.T) {
	q := NewSimpleQueue(10)

	// Schedule a future job
	err := q.Schedule(Job{
		ID:           "future-job",
		Type:         "test",
		ScheduledFor: time.Now().Add(10 * time.Second),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	// Verify timer was created
	q.mu.RLock()
	timerCount := len(q.timers)
	q.mu.RUnlock()

	if timerCount != 1 {
		t.Errorf("expected 1 timer, got %d", timerCount)
	}

	// Shutdown should cancel the timer
	q.Shutdown()

	// Verify timers were cleaned up
	q.mu.RLock()
	timerCount = len(q.timers)
	q.mu.RUnlock()

	if timerCount != 0 {
		t.Errorf("expected 0 timers after shutdown, got %d", timerCount)
	}
}

func TestSimpleQueue_ConcurrentScheduling(t *testing.T) {
	q := NewSimpleQueue(100)
	defer q.Shutdown()

	var executedCount int
	var mu sync.Mutex

	handler := func(job Job) error {
		mu.Lock()
		defer mu.Unlock()
		executedCount++
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go q.Process(ctx, "concurrent_test", handler)

	// Schedule multiple jobs concurrently
	const numJobs = 50
	var wg sync.WaitGroup
	wg.Add(numJobs)

	for i := 0; i < numJobs; i++ {
		go func(id int) {
			defer wg.Done()
			err := q.Schedule(Job{
				ID:           string(rune(id)),
				Type:         "concurrent_test",
				ScheduledFor: time.Now(),
				CreatedAt:    time.Now(),
			})
			if err != nil {
				t.Logf("failed to schedule job %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if executedCount == 0 {
		t.Error("no jobs were executed")
	}
	t.Logf("executed %d out of %d jobs", executedCount, numJobs)
}

package worker

import (
	"context"
	"sync"

	"github.com/quickchat/notification-service/internal/models"
	"github.com/quickchat/notification-service/internal/service"
	"github.com/sirupsen/logrus"
)

// Job represents a notification job to be processed
type Job struct {
	Request *models.NotificationRequest
	Result  chan *JobResult
}

// JobResult represents the result of a job
type JobResult struct {
	Notification *models.Notification
	Error        error
}

// Pool manages a pool of notification workers
type Pool struct {
	size    int
	jobs    chan *Job
	service *service.NotificationService
	log     *logrus.Logger
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewPool creates a new worker pool
func NewPool(size int, svc *service.NotificationService, log *logrus.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		size:    size,
		jobs:    make(chan *Job, size*10),
		service: svc,
		log:     log,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	p.log.Infof("Starting worker pool with %d workers", p.size)

	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop stops the worker pool
func (p *Pool) Stop() {
	p.log.Info("Stopping worker pool...")
	p.cancel()
	close(p.jobs)
	p.wg.Wait()
	p.log.Info("Worker pool stopped")
}

// Submit submits a job to the pool
func (p *Pool) Submit(req *models.NotificationRequest) (*models.Notification, error) {
	resultChan := make(chan *JobResult, 1)

	job := &Job{
		Request: req,
		Result:  resultChan,
	}

	select {
	case p.jobs <- job:
		// Job submitted
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}

	// Wait for result
	select {
	case result := <-resultChan:
		return result.Notification, result.Error
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	}
}

// SubmitAsync submits a job without waiting for result
func (p *Pool) SubmitAsync(req *models.NotificationRequest) error {
	job := &Job{
		Request: req,
		Result:  nil, // No result channel = fire and forget
	}

	select {
	case p.jobs <- job:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	default:
		return ErrQueueFull
	}
}

// worker processes jobs from the queue
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	p.log.Debugf("Worker %d started", id)

	for {
		select {
		case job, ok := <-p.jobs:
			if !ok {
				p.log.Debugf("Worker %d shutting down", id)
				return
			}

			p.processJob(job)

		case <-p.ctx.Done():
			p.log.Debugf("Worker %d context cancelled", id)
			return
		}
	}
}

// processJob processes a single job
func (p *Pool) processJob(job *Job) {
	notification, err := p.service.Send(context.Background(), job.Request)

	if job.Result != nil {
		job.Result <- &JobResult{
			Notification: notification,
			Error:        err,
		}
		close(job.Result)
	}

	if err != nil {
		p.log.Errorf("Failed to send notification to user %s: %v", job.Request.UserID, err)
	} else {
		p.log.Debugf("Notification sent to user %s", job.Request.UserID)
	}
}

// QueueLength returns current queue length
func (p *Pool) QueueLength() int {
	return len(p.jobs)
}

// ErrQueueFull is returned when the job queue is full
var ErrQueueFull = &QueueFullError{}

type QueueFullError struct{}

func (e *QueueFullError) Error() string {
	return "notification queue is full"
}

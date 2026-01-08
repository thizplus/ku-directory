package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
	"gofiber-template/pkg/logger"
)

type EventScheduler interface {
	Start()
	Stop()
	AddJob(id, cronExpr string, task func()) error
	RemoveJob(id string) error
	GetJob(id string) (*JobInfo, bool)
	ListJobs() map[string]*JobInfo
	IsRunning() bool
}

type JobInfo struct {
	ID       string
	CronExpr string
	Job      *gocron.Job
	IsActive bool
	LastRun  *time.Time
	NextRun  *time.Time
}

type GocronScheduler struct {
	scheduler *gocron.Scheduler
	jobs      map[string]*JobInfo
	mu        sync.RWMutex
	running   bool
}

func NewEventScheduler() EventScheduler {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.SingletonModeAll()

	return &GocronScheduler{
		scheduler: scheduler,
		jobs:      make(map[string]*JobInfo),
		running:   false,
	}
}

func (s *GocronScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		logger.SchedulerWarn("start", "Scheduler is already running", nil)
		return
	}

	s.scheduler.StartAsync()
	s.running = true
	logger.Scheduler("started", "Event scheduler started", nil)
}

func (s *GocronScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		logger.SchedulerWarn("stop", "Scheduler is not running", nil)
		return
	}

	s.scheduler.Stop()
	s.running = false
	logger.Scheduler("stopped", "Event scheduler stopped", nil)
}

func (s *GocronScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *GocronScheduler) AddJob(id, cronExpr string, task func()) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[id]; exists {
		return fmt.Errorf("job with ID %s already exists", id)
	}

	job, err := s.scheduler.Cron(cronExpr).Do(func() {
		now := time.Now()
		logger.Scheduler("job_executing", "Executing job", map[string]interface{}{"job_id": id, "time": now.Format(time.RFC3339)})

		// Update last run time
		s.mu.Lock()
		if jobInfo, exists := s.jobs[id]; exists {
			jobInfo.LastRun = &now
			if jobInfo.Job != nil {
				nextRun := jobInfo.Job.NextRun()
				jobInfo.NextRun = &nextRun
			}
		}
		s.mu.Unlock()

		// Execute the task
		task()
	})

	if err != nil {
		return fmt.Errorf("failed to create job: %v", err)
	}

	nextRun := job.NextRun()
	s.jobs[id] = &JobInfo{
		ID:       id,
		CronExpr: cronExpr,
		Job:      job,
		IsActive: true,
		LastRun:  nil,
		NextRun:  &nextRun,
	}

	logger.Scheduler("job_added", "Job added", map[string]interface{}{"job_id": id, "cron_expr": cronExpr, "next_run": nextRun.Format(time.RFC3339)})
	return nil
}

func (s *GocronScheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobInfo, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	if jobInfo.Job != nil {
		s.scheduler.RemoveByReference(jobInfo.Job)
	}

	delete(s.jobs, id)
	logger.Scheduler("job_removed", "Job removed", map[string]interface{}{"job_id": id})
	return nil
}

func (s *GocronScheduler) GetJob(id string) (*JobInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobInfo, exists := s.jobs[id]
	if !exists {
		return nil, false
	}

	// Create a copy to avoid race conditions
	info := &JobInfo{
		ID:       jobInfo.ID,
		CronExpr: jobInfo.CronExpr,
		Job:      jobInfo.Job,
		IsActive: jobInfo.IsActive,
	}

	if jobInfo.LastRun != nil {
		lastRun := *jobInfo.LastRun
		info.LastRun = &lastRun
	}

	if jobInfo.NextRun != nil {
		nextRun := *jobInfo.NextRun
		info.NextRun = &nextRun
	}

	// Update next run if job exists
	if jobInfo.Job != nil {
		nextRun := jobInfo.Job.NextRun()
		info.NextRun = &nextRun
	}

	return info, true
}

func (s *GocronScheduler) ListJobs() map[string]*JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make(map[string]*JobInfo)
	for id, jobInfo := range s.jobs {
		info := &JobInfo{
			ID:       jobInfo.ID,
			CronExpr: jobInfo.CronExpr,
			Job:      jobInfo.Job,
			IsActive: jobInfo.IsActive,
		}

		if jobInfo.LastRun != nil {
			lastRun := *jobInfo.LastRun
			info.LastRun = &lastRun
		}

		if jobInfo.NextRun != nil {
			nextRun := *jobInfo.NextRun
			info.NextRun = &nextRun
		}

		// Update next run if job exists
		if jobInfo.Job != nil {
			nextRun := jobInfo.Job.NextRun()
			info.NextRun = &nextRun
		}

		jobs[id] = info
	}

	return jobs
}

// Helper function to validate cron expression
func ValidateCronExpression(cronExpr string) error {
	scheduler := gocron.NewScheduler(time.UTC)
	_, err := scheduler.Cron(cronExpr).Do(func() {})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}
	return nil
}

// Helper function to get next run time from cron expression
func GetNextRunTime(cronExpr string) (*time.Time, error) {
	scheduler := gocron.NewScheduler(time.UTC)
	job, err := scheduler.Cron(cronExpr).Do(func() {})
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %v", err)
	}

	nextRun := job.NextRun()
	return &nextRun, nil
}

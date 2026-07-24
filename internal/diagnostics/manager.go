package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

var (
	ErrQueueFull          = errors.New("diagnostic queue is full")
	ErrRunAlreadyFinished = errors.New("diagnostic run is already finished")
)

type Manager struct {
	repository       RunRepository
	events           EventPublisher
	logger           *slog.Logger
	queue            chan uuid.UUID
	probes           map[CheckType]Probe
	probeConcurrency int
	workers          int
	rootContext      context.Context
	stop             context.CancelFunc
	waitGroup        sync.WaitGroup
	activeMutex      sync.Mutex
	active           map[uuid.UUID]context.CancelFunc
}

func NewManager(
	repository RunRepository,
	events EventPublisher,
	logger *slog.Logger,
	workers int,
	queueSize int,
	probeConcurrency int,
	probes ...Probe,
) *Manager {
	registered := make(map[CheckType]Probe, len(probes))
	for _, item := range probes {
		registered[item.Type()] = item
	}
	rootContext, cancel := context.WithCancel(context.Background())
	return &Manager{
		repository:       repository,
		events:           events,
		logger:           logger,
		queue:            make(chan uuid.UUID, queueSize),
		probes:           registered,
		probeConcurrency: probeConcurrency,
		workers:          workers,
		rootContext:      rootContext,
		stop:             cancel,
		active:           make(map[uuid.UUID]context.CancelFunc),
	}
}

func (m *Manager) Start() {
	for index := 0; index < m.workers; index++ {
		m.waitGroup.Add(1)
		go m.worker(index)
	}
}

func (m *Manager) Enqueue(runID uuid.UUID) error {
	select {
	case <-m.rootContext.Done():
		return context.Canceled
	case m.queue <- runID:
		return nil
	default:
		return ErrQueueFull
	}
}

func (m *Manager) Supports(check CheckType) bool {
	_, exists := m.probes[check]
	return exists
}

func (m *Manager) Cancel(ctx context.Context, runID uuid.UUID) error {
	run, err := m.repository.GetByID(ctx, runID)
	if err != nil {
		return err
	}
	if terminal(run.Status) {
		return ErrRunAlreadyFinished
	}

	m.activeMutex.Lock()
	cancel := m.active[runID]
	m.activeMutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if err := m.repository.UpdateStatus(ctx, runID, RunCancelled); err != nil {
		return err
	}
	m.events.Publish(runID, RunEvent{
		Type: EventRunCancelled, RunID: runID, Status: string(RunCancelled), Timestamp: time.Now().UTC(),
	})
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.stop()
	done := make(chan struct{})
	go func() {
		m.waitGroup.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) worker(index int) {
	defer m.waitGroup.Done()
	for {
		select {
		case <-m.rootContext.Done():
			return
		case runID := <-m.queue:
			if err := m.execute(runID); err != nil {
				m.logger.Error("execute diagnostic run", "worker", index, "run_id", runID, "error", err)
			}
		}
	}
}

func (m *Manager) execute(runID uuid.UUID) error {
	run, err := m.repository.GetByID(m.rootContext, runID)
	if err != nil {
		return err
	}
	if terminal(run.Status) {
		return nil
	}
	parsedTarget, err := target.Parse(run.TargetInput)
	if err != nil {
		return fmt.Errorf("parse stored target: %w", err)
	}
	run.Target = parsedTarget
	runTimeout := time.Duration(run.Options.TimeoutMS) * time.Millisecond
	runContext, cancel := context.WithTimeout(m.rootContext, runTimeout)
	m.activeMutex.Lock()
	m.active[runID] = cancel
	m.activeMutex.Unlock()
	defer func() {
		cancel()
		m.activeMutex.Lock()
		delete(m.active, runID)
		m.activeMutex.Unlock()
	}()

	if err := m.repository.UpdateStatus(runContext, runID, RunRunning); err != nil {
		return err
	}
	m.events.Publish(runID, RunEvent{
		Type: EventRunStarted, RunID: runID, Status: string(RunRunning), Timestamp: time.Now().UTC(),
	})

	semaphore := make(chan struct{}, m.probeConcurrency)
	results := make(chan CheckResult, len(run.RequestedChecks))
	var waitGroup sync.WaitGroup

	for _, check := range run.RequestedChecks {
		item, exists := m.probes[check]
		if !exists {
			continue
		}
		waitGroup.Add(1)
		go func(item Probe) {
			defer waitGroup.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-runContext.Done():
				return
			}

			m.events.Publish(runID, RunEvent{
				Type: EventCheckStarted, RunID: runID, Check: item.Type(),
				Status: string(CheckRunning), Timestamp: time.Now().UTC(),
			})
			timeout := time.Duration(run.Options.TimeoutMS) * time.Millisecond
			probeContext, probeCancel := context.WithTimeout(runContext, timeout)
			result := item.Run(probeContext, run.Target, run.Options)
			probeCancel()
			result.RunID = runID
			saveContext, saveCancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer saveCancel()
			if err := m.repository.SaveResult(saveContext, result); err != nil {
				m.logger.Error("save probe result", "run_id", runID, "probe", item.Type(), "error", err)
			}
			results <- result
		}(item)
	}

	go func() {
		waitGroup.Wait()
		close(results)
	}()

	collected := make([]CheckResult, 0, len(run.RequestedChecks))
	for result := range results {
		collected = append(collected, result)
		eventType := EventCheckCompleted
		if result.Status == CheckFailed {
			eventType = EventCheckFailed
		}
		m.events.Publish(runID, RunEvent{
			Type: eventType, RunID: runID, Check: result.Type,
			Status: string(result.Status), Duration: result.DurationMS,
			Timestamp: time.Now().UTC(), Result: &result,
		})
		m.logger.Info("probe completed",
			"run_id", runID,
			"probe", result.Type,
			"target", run.NormalizedHost,
			"status", result.Status,
			"duration_ms", result.DurationMS,
		)
	}

	status := CalculateRunStatus(collected, runContext.Err() != nil)
	updateContext, updateCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer updateCancel()
	if err := m.repository.UpdateStatus(updateContext, runID, status); err != nil {
		return fmt.Errorf("finish run: %w", err)
	}
	eventType := EventRunCompleted
	if status == RunCancelled {
		eventType = EventRunCancelled
	}
	m.events.Publish(runID, RunEvent{
		Type: eventType, RunID: runID, Status: string(status), Timestamp: time.Now().UTC(),
	})
	return nil
}

func terminal(status RunStatus) bool {
	switch status {
	case RunCompleted, RunPartial, RunFailed, RunCancelled, RunInterrupted:
		return true
	default:
		return false
	}
}

package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/profile"
)

const maxBuildFailures = int32(3)

type BuildRepository interface {
	// ClaimProfileBuildTask claims at most one eligible task. A nil task means
	// no work is currently available.
	ClaimProfileBuildTask(context.Context, time.Duration) (*profile.BuildTask, error)
	RenewProfileBuildTaskLease(context.Context, profile.BuildTask, time.Duration) (bool, error)
	GetProfileBuildInput(context.Context, int64) (profile.BuildInput, error)
	// CommitProfile atomically inserts the unique profile and marks Task
	// succeeded. An existing equal hash is an idempotent success; an existing
	// different hash must return profile.ErrContentConflict.
	CommitProfile(context.Context, CommitProfileCommand) (CommitProfileResult, error)
	RetryProfileBuildTask(context.Context, RetryProfileBuildTaskCommand) (bool, error)
	FailProfileBuildTask(context.Context, FailProfileBuildTaskCommand) (bool, error)
}

type CommitProfileCommand struct {
	Task    profile.BuildTask
	Profile profile.ProjectProfile
}

type CommitProfileResult struct {
	Profile profile.ProjectProfile
	Created bool
}

type RetryProfileBuildTaskCommand struct {
	Task         profile.BuildTask
	FailureCount int32
	LastError    string
	AvailableAt  time.Time
}

type FailProfileBuildTaskCommand struct {
	Task         profile.BuildTask
	FailureCount int32
	LastError    string
	FailedAt     time.Time
}

type BuilderOptions struct {
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	Now               func() time.Time
}

type Builder struct {
	repository BuildRepository
	options    BuilderOptions
}

func NewBuilder(repository BuildRepository, options BuilderOptions) *Builder {
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = 90 * time.Second
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = 30 * time.Second
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Builder{repository: repository, options: options}
}

// RunOnce claims and processes at most one task so no claimed item waits behind
// another profile build while its lease is already counting down.
func (builder *Builder) RunOnce(ctx context.Context) (int, error) {
	if builder == nil || builder.repository == nil {
		return 0, fmt.Errorf("token profile builder application is not configured")
	}
	task, err := builder.repository.ClaimProfileBuildTask(ctx, builder.options.LeaseDuration)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, nil
	}
	if err := builder.process(ctx, *task); err != nil {
		return 0, err
	}
	return 1, nil
}

func (builder *Builder) process(ctx context.Context, task profile.BuildTask) error {
	workCtx, cancel := context.WithCancel(ctx)
	heartbeatDone := make(chan error, 1)
	go builder.heartbeat(workCtx, cancel, task, heartbeatDone)

	input, buildErr := builder.repository.GetProfileBuildInput(workCtx, task.ProjectID)
	var item profile.ProjectProfile
	if buildErr == nil {
		item, buildErr = profile.Build(input, builder.options.Now().UTC())
	}
	cancel()
	heartbeatErr := <-heartbeatDone
	if heartbeatErr != nil {
		return heartbeatErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Give the fenced terminal transition a fresh full lease window and reject a
	// build whose claim was lost after its last heartbeat.
	renewed, err := builder.repository.RenewProfileBuildTaskLease(ctx, task, builder.options.LeaseDuration)
	if err != nil {
		return fmt.Errorf("renew token profile build lease before completion: %w", err)
	}
	if !renewed {
		return profile.ErrBuildTaskClaimLost
	}
	if buildErr != nil {
		if errors.Is(buildErr, profile.ErrBuildTaskClaimLost) || ctx.Err() != nil {
			return buildErr
		}
		return builder.recordFailure(ctx, task, buildErr)
	}

	// The commit is one short fenced database transaction. Stop lease renewal
	// first so a successful commit cannot race a heartbeat that observes the
	// now-terminal task and falsely reports a lost claim.
	_, commitErr := builder.repository.CommitProfile(ctx, CommitProfileCommand{Task: task, Profile: item})
	if commitErr == nil {
		return nil
	}
	if errors.Is(commitErr, profile.ErrBuildTaskClaimLost) || ctx.Err() != nil {
		return commitErr
	}
	return builder.recordFailure(ctx, task, commitErr)
}

func (builder *Builder) heartbeat(ctx context.Context, cancel context.CancelFunc, task profile.BuildTask, done chan<- error) {
	ticker := time.NewTicker(builder.options.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			done <- nil
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				done <- nil
				return
			}
			applied, err := builder.repository.RenewProfileBuildTaskLease(ctx, task, builder.options.LeaseDuration)
			if err != nil {
				if ctx.Err() != nil {
					done <- nil
					return
				}
				cancel()
				done <- fmt.Errorf("renew token profile build lease: %w", err)
				return
			}
			if !applied {
				if ctx.Err() != nil {
					done <- nil
					return
				}
				cancel()
				done <- profile.ErrBuildTaskClaimLost
				return
			}
		}
	}
}

func (builder *Builder) recordFailure(ctx context.Context, task profile.BuildTask, failure error) error {
	failedAt := builder.options.Now().UTC()
	failureCount := task.FailureCount + 1
	if failureCount < maxBuildFailures {
		backoff := time.Duration(failureCount) * time.Second
		applied, err := builder.repository.RetryProfileBuildTask(ctx, RetryProfileBuildTaskCommand{
			Task: task, FailureCount: failureCount, LastError: failure.Error(),
			AvailableAt: failedAt.Add(backoff),
		})
		if err != nil {
			return err
		}
		if !applied {
			return profile.ErrBuildTaskClaimLost
		}
		return nil
	}
	applied, err := builder.repository.FailProfileBuildTask(ctx, FailProfileBuildTaskCommand{
		Task: task, FailureCount: failureCount, LastError: failure.Error(), FailedAt: failedAt,
	})
	if err != nil {
		return err
	}
	if !applied {
		return profile.ErrBuildTaskClaimLost
	}
	return nil
}

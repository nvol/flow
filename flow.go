package flow

import (
	"context"
	"errors"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// global break
var (
	GlobalBreakCtx  context.Context
	GlobalBreakFunc context.CancelFunc
)

func Run(
	work func(
		breakCtx context.Context,
		moduleWg *sync.WaitGroup,
		selfRestartFunc context.CancelFunc,
	),
	maxDurToBreak time.Duration,
	logger Logger,
) {
	if logger == nil {
		logger = new(defaultLogger)
	}

	logger.Info("let's begin!")
	defer logger.Info("goodbye!")

	GlobalBreakCtx, GlobalBreakFunc = signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer GlobalBreakFunc()

flowLoop:
	for {
		moduleWg := new(sync.WaitGroup)

		logger.Info("starting a work")

		restartCtx, restartFunc := context.WithCancel(GlobalBreakCtx)
		breakCtx, breakFunc := context.WithCancel(GlobalBreakCtx)
		go work(breakCtx, moduleWg, restartFunc)

		needRestart := false
		select {
		case <-restartCtx.Done():
			needRestart = true // don't forget to restart the work
		case <-GlobalBreakCtx.Done():
		}

		breakFunc() // send break signal to the worker anyway (break or restart)

		workDone := make(chan struct{})
		go func() {
			moduleWg.Wait()
			close(workDone)
		}()

	wgDrainerLoop:
		for {
			select {
			case <-workDone:
				logger.Info("the work is done")
				if needRestart {
					// just restart the work
					break wgDrainerLoop
				}
				// break totally
				break flowLoop
			case <-time.After(maxDurToBreak):
				logger.Info("ATTENTION: module wait-group was not unlocked during " + maxDurToBreak.String())
				panic(errors.New("some work still in progress, cannot shutdown gracefully"))
			}
		}
		// ---
		logger.Info("will restart the work in 1s")

		// breakable delay between restarts
		select {
		case <-time.After(time.Second):
		case <-GlobalBreakCtx.Done():
			// break totally
			break flowLoop
		}
	}
	// ---
	logger.Info("shutting down gracefully")
}

func BreakForcibly() {
	if GlobalBreakFunc != nil {
		GlobalBreakFunc()
	}
}

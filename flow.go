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

	logger.Info("Let's begin!")
	defer logger.Info("Goodbye!")

	GlobalBreakCtx, GlobalBreakFunc = signal.NotifyContext(context.Background(), syscall.SIGINT)
	defer GlobalBreakFunc()

flowLoop:
	for {
		moduleWg := &sync.WaitGroup{}

		logger.Info("Starting a work...")

		restartCtx, restartFunc := context.WithCancel(GlobalBreakCtx)
		breakCtx, breakFunc := context.WithCancel(GlobalBreakCtx)
		go work(breakCtx, moduleWg, restartFunc)

		needRestart := false
		select {
		case <-restartCtx.Done():
			logger.Info("Got a signal to restart the work")
			needRestart = true // don't forget to restart the work
		case <-GlobalBreakCtx.Done():
			logger.Info("Got a signal to finish the work")
		}

		breakFunc() // send break signal to the worker anyway (break or restart)

		workDone := make(chan struct{})
		go func() {
			logger.Info("Waiting for the module wait-group to get unlocked...")
			moduleWg.Wait()
			close(workDone)
		}()

	wgDrainerLoop:
		for {
			select {
			case <-workDone:
				logger.Info("The work is done (module wait-group unlocked)")
				if needRestart {
					// just restart the work
					break wgDrainerLoop
				}
				// break totally
				break flowLoop
			case <-time.After(maxDurToBreak):
				logger.Info("ATTENTION: Module wait-group was not unlocked during " + maxDurToBreak.String())
				panic(errors.New("some work still in progress, cannot shutdown gracefully"))
			}
		}
		// ---
		if GlobalBreakCtx.Err() == nil {
			logger.Info("Will restart the work in 1s")
		}

		// breakable delay before restart
		select {
		case <-time.After(time.Second):
		case <-GlobalBreakCtx.Done():
			// break totally
			logger.Info("Got a break signal during the delay before the work restart")
			break flowLoop
		}
	}
}

func BreakForcibly() {
	if GlobalBreakFunc != nil {
		GlobalBreakFunc()
	}
}

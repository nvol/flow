package examples

import (
	"context"
	"errors"
	"fmt"
	"nvol/flow"
	"sync"
	"testing"
	"time"
)

func TestMostSimpleCase(t *testing.T) {
	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, _ context.CancelFunc) {
		// add work itself to the module wait-group
		moduleWg.Add(1)
		defer moduleWg.Done()

		// do something...

		// run some goroutines
		moduleWg.Add(1)
		go func() {
			defer moduleWg.Done()

			// do something...

			<-breakCtx.Done()
		}()

		// do something...

		<-breakCtx.Done()
	}

	go func() {
		time.Sleep(time.Second)
		flow.BreakForcibly()
	}()

	flow.Run(work, time.Second*3, nil)
}

func TestOneRelaunchAndOneBreak(t *testing.T) {
	launchCount := 0
	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, restartItself context.CancelFunc) {
		// add work itself to the module wait-group
		moduleWg.Add(1)
		defer moduleWg.Done()

		launchCount++

		// do something...

		// run some goroutines
		moduleWg.Add(1)
		go func() {
			defer moduleWg.Done()

			// do something...

			<-breakCtx.Done()
		}()

		// do something...
		time.Sleep(time.Second)

		// first time - relaunch, second time - break totally
		if launchCount == 1 {
			restartItself()
		} else {
			flow.BreakForcibly()
		}

		<-breakCtx.Done()
	}

	flow.Run(work, time.Second*3, nil)

	if launchCount != 2 {
		t.Errorf("expected 2 launchs, got %v", launchCount)
	}
}

func TestInnerBreak(t *testing.T) {
	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		moduleWg.Add(1)
		defer moduleWg.Done()

		moduleWg.Add(1)
		go func() {
			defer moduleWg.Done()
			select {
			case <-time.After(time.Hour):
				panic(errors.New("must not be"))
			case <-breakCtx.Done():
				fmt.Println("work: inner goroutine stopped")
			}
		}()

		time.Sleep(time.Second)
		flow.BreakForcibly()
	}

	flow.Run(work, time.Second*3, nil)
}

func TestOuterBreak(t *testing.T) {
	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		moduleWg.Add(1)
		defer moduleWg.Done()

		moduleWg.Add(1)
		go func() {
			defer moduleWg.Done()
			select {
			case <-time.After(time.Hour):
				panic(errors.New("must not be"))
			case <-breakCtx.Done():
				fmt.Println("work: inner goroutine stopped")
			}
		}()

		<-breakCtx.Done()
	}

	go func() {
		time.Sleep(time.Second * 2)
		flow.BreakForcibly()
	}()
	flow.Run(work, time.Second*3, nil)
}

func TestNormalRestart(t *testing.T) {
	launchCount := 0
	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		launchCount++
		moduleWg.Add(1)
		defer moduleWg.Done()

		moduleWg.Add(1)
		go func() {
			defer moduleWg.Done()
			select {
			case <-time.After(time.Hour):
				panic(errors.New("must not be"))
			case <-breakCtx.Done():
				fmt.Println("work: inner goroutine stopped")
			}
		}()

		if launchCount == 3 {
			flow.BreakForcibly() // inner break
		}

		select {
		case <-time.After(time.Millisecond * 1500):
		case <-breakCtx.Done():
			fmt.Println("work: stop signal caught")
			return
		}

		selfRestartFunc()
	}

	flow.Run(work, time.Second*3, nil)

	if launchCount != 3 {
		t.Errorf("expected 3 launchs, got %v", launchCount)
	}
}

func TestTryingToBreakNotBalancedWork(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic expected")
		} else {
			t.Logf("it's ok, got an expected panic: %v", r)
		}
	}()

	work := func(breakCtx context.Context, globalWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		globalWg.Add(1)
		time.Sleep(time.Second)
		if breakCtx.Err() != nil {
			fmt.Println("work: got a break")
			globalWg.Done()
			return
		}
		fmt.Println("work: finishing work with module wait-group not drained")
		// globalWg.Done() missed!
		return
	}

	go func() {
		time.Sleep(time.Second * 2)
		flow.BreakForcibly()
	}()
	flow.Run(work, time.Second*3, nil)
}

func TestTryingToBreakSuspendedWork(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic expected")
		} else {
			t.Logf("it's ok, got an expected panic: %v", r)
		}
	}()

	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		moduleWg.Add(1)
		defer moduleWg.Done()

		time.Sleep(time.Second)
		if breakCtx.Err() != nil {
			fmt.Println("work: got a break")
			return
		}
		fmt.Println("work: suspending for a long time...")
		time.Sleep(time.Second * 20) // works too long (more than 3s)
	}

	go func() {
		time.Sleep(time.Second * 2)
		flow.BreakForcibly()
	}()
	flow.Run(work, time.Second*3, nil)
}

func TestTryingToRestartNotBalancedWork(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic expected")
		} else {
			t.Logf("it's ok, got an expected panic: %v", r)
		}
	}()

	work := func(breakCtx context.Context, globalWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		globalWg.Add(1)
		time.Sleep(time.Second)
		if breakCtx.Err() != nil {
			fmt.Println("work: got a break")
			globalWg.Done()
			return
		}
		selfRestartFunc() // restart
		fmt.Println("work: trying to restart with module wait-group not drained")
		// globalWg.Done() missed!
		return
	}

	go func() {
		time.Sleep(time.Second * 10)
		flow.BreakForcibly()
	}()
	flow.Run(work, time.Second*3, nil)
}

func TestTryingToRestartSuspendedWork(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("panic expected")
		} else {
			t.Logf("it's ok, got an expected panic: %v", r)
		}
	}()

	work := func(breakCtx context.Context, moduleWg *sync.WaitGroup, selfRestartFunc context.CancelFunc) {
		fmt.Println("work: the work started")
		defer fmt.Println("work: the end of func is reached")
		moduleWg.Add(1)
		defer moduleWg.Done()

		time.Sleep(time.Second)
		if breakCtx.Err() != nil {
			fmt.Println("work: got a break")
			return
		}
		selfRestartFunc() // restart
		fmt.Println("work: trying to restart")
		time.Sleep(time.Second * 4) // stops too long (more than 3s)
	}

	go func() {
		time.Sleep(time.Second * 10)
		flow.BreakForcibly()
	}()
	flow.Run(work, time.Second*3, nil)
}

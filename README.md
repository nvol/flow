# Simple program flow controller

## Features

- run the work
- gracefully shutdown the work with help of external break signal
- gracefully shutdown the work from itself
- gracefully restart the work from itself

## Usage

### Example 1 - the most simple case

```go
package main

import (
	"context"
	"sync"
	"time"

	"github.com/nvol/flow"
)

func main() {
	flow.Run(work, time.Second*3, nil)
}

func work(breakCtx context.Context, moduleWg *sync.WaitGroup, _ context.CancelFunc) {
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
```

## Example 2 - restart the work from itself (once) and then break the work from itself

```go
package main

import (
	"context"
	"sync"
	"time"

	"github.com/nvol/flow"
)

var launchCount int

func main() {
	flow.Run(work, time.Second*3, nil)
}

func work(breakCtx context.Context, moduleWg *sync.WaitGroup, restartItself context.CancelFunc) {
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

	// first time - relaunch, second time - break totally
	if launchCount == 1 {
		restartItself()
    } else {
		flow.BreakForcibly()
    }
	
	<-breakCtx.Done()
}
```

package registry

import (
	"context"
	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/weaveworks/flux"
	"os"
	"sync"
	"testing"
	"time"
)

func TestQueue_AllContainersShouldGetAdded(t *testing.T) {
	queue := NewQueue(
		func() []flux.ImageID {
			id, _ := flux.ParseImageID("test/image")
			return []flux.ImageID{id, id}
		},
		log.NewLogfmtLogger(os.Stderr),
		1*time.Millisecond,
	)

	shutdown := make(chan struct{})
	shutdownWg := &sync.WaitGroup{}
	shutdownWg.Add(1)
	go queue.Loop(shutdown, shutdownWg)
	defer func() {
		shutdown <- struct{}{}
		shutdownWg.Wait()
	}()

	time.Sleep(10 * time.Millisecond)
	if len(queue.Queue()) != 2 {
		t.Fatal("Should have randomly added two containers to queue")
	}
}

func TestQueue_NoContainers(t *testing.T) {
	queue := NewQueue(
		func() []flux.ImageID {
			return []flux.ImageID{}
		},
		log.NewLogfmtLogger(os.Stderr),
		1*time.Millisecond,
	)

	shutdown := make(chan struct{})
	shutdownWg := &sync.WaitGroup{}
	shutdownWg.Add(1)
	go queue.Loop(shutdown, shutdownWg)
	defer func() {
		shutdown <- struct{}{}
		shutdownWg.Wait()
	}()

	time.Sleep(10 * time.Millisecond)
	if len(queue.Queue()) != 0 {
		t.Fatal("There were no containers, so there should be no repositories in the queue")
	}
}

func TestWarming_ExpiryBuffer(t *testing.T) {
	testTime := time.Now()
	for _, x := range []struct {
		expiresIn, buffer time.Duration
		expectedResult    bool
	}{
		{time.Minute, time.Second, false},
		{time.Second, time.Minute, true},
	} {
		if withinExpiryBuffer(testTime.Add(x.expiresIn), x.buffer) != x.expectedResult {
			t.Fatalf("Should return %t", x.expectedResult)
		}
	}
}

func TestName(t *testing.T) {
	err := errors.Wrap(context.DeadlineExceeded, "getting remote manifest")
	t.Log(err.Error())
	err = errors.Cause(err)
	if err == context.DeadlineExceeded {
		t.Log("OK")
	} else {
		t.Log("Not OK")
	}
}

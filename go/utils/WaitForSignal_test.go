package utils_test

import (
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
)

func TestWaitForSignal(t *testing.T) {
	m := sync.Mutex{}
	var waitCompleted atomic.Bool

	go func() {
		utils.WaitForSignal()
		m.Lock()
		waitCompleted.Store(true)
		m.Unlock()
	}()
	pid := os.Getpid()
	time.Sleep(time.Second)

	// signal.Notify()
	syscall.Kill(pid, syscall.SIGTERM)
	time.Sleep(time.Millisecond * 10)
	m.Lock()
	defer m.Unlock()
	assert.True(t, waitCompleted.Load())
}

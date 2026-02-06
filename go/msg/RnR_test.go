package msg_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/msg"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const DefaultResponseTimeout = time.Second * 10

func TestRnROpenClose(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)
	rnrChan.Close(corrID)
}

func TestRnRWaitAfterOpen(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
	require.True(t, handled)

	hasResponse, rx := rnrChan.WaitForResponse(corrID, DefaultResponseTimeout)
	require.True(t, hasResponse)
	require.NotEmpty(t, rx)
	require.Equal(t, corrID, rx.CorrelationID)
	require.Equal(t, resp.ThingID, rx.ThingID)

	rnrChan.Close(corrID)
}

func TestRnRWaitNoOpenFails(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()
	// rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
	require.False(t, handled)
	// should fail immediately as corrID doesn't exist
	hasResponse, rx := rnrChan.WaitForResponse(corrID, DefaultResponseTimeout)
	require.False(t, hasResponse)
	require.Empty(t, rx)

	rnrChan.Close(corrID)
}

func TestRnROpenWaitCallback(t *testing.T) {
	var rxData atomic.Bool

	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
	require.True(t, handled)

	rnrChan.WaitWithCallback(corrID, DefaultResponseTimeout, func(resp *msg.ResponseMessage) error {
		rxData.Store(true)
		return nil
	})
	time.Sleep(time.Millisecond)
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	assert.True(t, rxData.Load())
}

// Test receiving a response before open
func TestRnRResponseNotHandled(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()

	handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
	require.False(t, handled)
}

// Test receiving a response twice
func TestRnRResponseTwice(t *testing.T) {
	corrID := "123"
	var rxData atomic.Bool
	shortTimeout := time.Second

	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()

	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp, shortTimeout)
	assert.True(t, handled)

	// the second one fails with an error in the log
	handled = rnrChan.HandleResponse(resp, shortTimeout)
	require.False(t, handled)

	// this will close the channel
	rnrChan.WaitWithCallback(corrID, shortTimeout, func(resp *msg.ResponseMessage) error {
		rxData.Store(true)
		return nil
	})
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	assert.True(t, rxData.Load())

}

func TestRnRTimeout(t *testing.T) {
	var rxData atomic.Bool
	corrID := "123"
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)

	// use a short timeout for this test
	timeout := time.Second

	hasResponse, rx := rnrChan.WaitForResponse(corrID, timeout)
	assert.False(t, hasResponse)
	assert.Empty(t, rx)

	// try again with callback
	rnrChan.WaitWithCallback(corrID, timeout, func(resp *msg.ResponseMessage) error {
		rxData.Store(true)
		return nil
	})

	rnrChan.Close(corrID)
	assert.False(t, rxData.Load())
}

func TestRnRDoubleClose(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)

	rnrChan.Close(corrID)
	rnrChan.Close(corrID)
}

func TestRnRCloseAll(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)
	assert.Equal(t, 1, rnrChan.Len())

	rnrChan.CloseAll()
	assert.Equal(t, 0, rnrChan.Len())
}

func TestRnRDoubleOpen(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan()
	rnrChan.Open(corrID)
	assert.Panics(t, func() {
		rnrChan.Open(corrID)
	})
	rnrChan.Close(corrID)
}

func TestRnRNoOpenWaitCallback(t *testing.T) {
	var rxData atomic.Bool

	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan()

	rnrChan.WaitWithCallback(corrID, DefaultResponseTimeout, func(resp *msg.ResponseMessage) error {
		rxData.Store(true)
		return nil
	})
	handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	require.True(t, handled)

	require.True(t, rxData.Load())
}

func Benchmark_RnRBulkWaitCallback(b *testing.B) {
	var rxCount atomic.Int32

	utils.SetLogging("warning", "")

	b.Run("waitwithcallback", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			corrID := fmt.Sprintf("corr-%d", n)

			resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
			rnrChan := msg.NewRnRChan()
			rnrChan.Open(corrID)
			go func() {
				handled := rnrChan.HandleResponse(resp, DefaultResponseTimeout)
				require.True(b, handled)
			}()
			rnrChan.WaitWithCallback(corrID, DefaultResponseTimeout, func(resp *msg.ResponseMessage) error {
				rxCount.Add(1)
				return nil
			})
		}
	})

	time.Sleep(time.Millisecond)
	b.Logf("Ran %d tests", rxCount.Load())
	// assert.Equal(b, int32(nrCalls), rxCount.Load())
}

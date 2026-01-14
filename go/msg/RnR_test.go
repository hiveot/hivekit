package msg_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/lib/logging"
	"github.com/hiveot/hivekit/go/msg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRnROpenClose(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan(0)
	rnrChan.Open(corrID)
	rnrChan.Close(corrID)
}

func TestRnRWaitAfterOpen(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(0)
	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp)
	require.True(t, handled)

	hasResponse, rx := rnrChan.WaitForResponse(corrID)
	require.True(t, hasResponse)
	require.NotEmpty(t, rx)
	require.Equal(t, corrID, rx.CorrelationID)
	require.Equal(t, resp.ThingID, rx.ThingID)

	rnrChan.Close(corrID)
}

func TestRnRWaitNoOpenFails(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(0)
	// rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp)
	require.False(t, handled)

	hasResponse, rx := rnrChan.WaitForResponse(corrID)
	require.False(t, hasResponse)
	require.Empty(t, rx)

	rnrChan.Close(corrID)
}

func TestRnROpenWaitCallback(t *testing.T) {
	rxData := false

	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(0)
	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp)
	require.True(t, handled)

	rnrChan.WaitWithCallback(corrID, func(resp *msg.ResponseMessage) error {
		rxData = true
		return nil
	})
	time.Sleep(time.Millisecond)
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	assert.True(t, rxData)
}

// Test receiving a response before open
func TestRnRResponseNotHandled(t *testing.T) {
	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(0)

	handled := rnrChan.HandleResponse(resp)
	require.False(t, handled)
}

// Test receiving a response twice
func TestRnRResponseTwice(t *testing.T) {
	corrID := "123"
	rxData := false

	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(time.Millisecond * 3)

	rnrChan.Open(corrID)

	handled := rnrChan.HandleResponse(resp)
	assert.True(t, handled)

	// the second one fails with an error in the log
	handled = rnrChan.HandleResponse(resp)
	require.False(t, handled)

	// this will close the channel
	rnrChan.WaitWithCallback(corrID, func(resp *msg.ResponseMessage) error {
		rxData = true
		return nil
	})
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	assert.True(t, rxData)

}

func TestRnRTimeout(t *testing.T) {
	rxData := false
	corrID := "123"
	rnrChan := msg.NewRnRChan(time.Millisecond * 10)
	rnrChan.Open(corrID)

	hasResponse, rx := rnrChan.WaitForResponse(corrID)
	assert.False(t, hasResponse)
	assert.Empty(t, rx)

	// try again with callback
	rnrChan.WaitWithCallback(corrID, func(resp *msg.ResponseMessage) error {
		rxData = true
		return nil
	})

	rnrChan.Close(corrID)
	assert.False(t, rxData)
}

func TestRnRDoubleClose(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan(time.Millisecond)
	rnrChan.Open(corrID)

	rnrChan.Close(corrID)
	rnrChan.Close(corrID)
}

func TestRnRCloseAll(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan(time.Millisecond)
	rnrChan.Open(corrID)
	assert.Equal(t, 1, rnrChan.Len())

	rnrChan.CloseAll()
	assert.Equal(t, 0, rnrChan.Len())
}

func TestRnRDoubleOpen(t *testing.T) {
	corrID := "123"
	rnrChan := msg.NewRnRChan(time.Millisecond)
	rnrChan.Open(corrID)
	assert.Panics(t, func() {
		rnrChan.Open(corrID)
	})
	rnrChan.Close(corrID)
}

func TestRnRNoOpenWaitCallback(t *testing.T) {
	rxData := false

	corrID := "123"
	resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
	rnrChan := msg.NewRnRChan(0)

	rnrChan.WaitWithCallback(corrID, func(resp *msg.ResponseMessage) error {
		rxData = true
		return nil
	})
	handled := rnrChan.HandleResponse(resp)
	// response is handled in a separate goroutine
	time.Sleep(time.Millisecond)
	require.True(t, handled)

	require.True(t, rxData)
}

func Benchmark_RnRBulkWaitCallback(b *testing.B) {
	var rxCount atomic.Int32

	logging.SetLogging("warning", "")

	b.Run("waitwithcallback", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			corrID := fmt.Sprintf("corr-%d", n)

			resp := msg.NewResponseMessage("op1", "thing1", "name", nil, nil, corrID)
			rnrChan := msg.NewRnRChan(0)
			rnrChan.Open(corrID)
			go func() {
				handled := rnrChan.HandleResponse(resp)
				require.True(b, handled)
			}()
			rnrChan.WaitWithCallback(corrID, func(resp *msg.ResponseMessage) error {
				rxCount.Add(1)
				return nil
			})
		}
	})

	time.Sleep(time.Millisecond)
	b.Logf("Ran %d tests", rxCount.Load())
	// assert.Equal(b, int32(nrCalls), rxCount.Load())
}

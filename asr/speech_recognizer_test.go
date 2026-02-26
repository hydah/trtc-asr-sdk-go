package asr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hydah/trtc-asr-sdk-go/common"
)

type failEvent struct {
	resp *SpeechRecognitionResponse
	err  error
}

type testListener struct {
	mu         sync.Mutex
	startN     int
	sentenceN  int
	changeN    int
	endN       int
	completeN  int
	failN      int
	failCh     chan failEvent
	completeCh chan *SpeechRecognitionResponse
}

func newTestListener() *testListener {
	return &testListener{
		failCh:     make(chan failEvent, 8),
		completeCh: make(chan *SpeechRecognitionResponse, 8),
	}
}

func (l *testListener) OnRecognitionStart(_ *SpeechRecognitionResponse) {
	l.mu.Lock()
	l.startN++
	l.mu.Unlock()
}

func (l *testListener) OnSentenceBegin(_ *SpeechRecognitionResponse) {
	l.mu.Lock()
	l.sentenceN++
	l.mu.Unlock()
}

func (l *testListener) OnRecognitionResultChange(_ *SpeechRecognitionResponse) {
	l.mu.Lock()
	l.changeN++
	l.mu.Unlock()
}

func (l *testListener) OnSentenceEnd(_ *SpeechRecognitionResponse) {
	l.mu.Lock()
	l.endN++
	l.mu.Unlock()
}

func (l *testListener) OnRecognitionComplete(resp *SpeechRecognitionResponse) {
	l.mu.Lock()
	l.completeN++
	l.mu.Unlock()
	select {
	case l.completeCh <- resp:
	default:
	}
}

func (l *testListener) OnFail(resp *SpeechRecognitionResponse, err error) {
	l.mu.Lock()
	l.failN++
	l.mu.Unlock()
	select {
	case l.failCh <- failEvent{resp: resp, err: err}:
	default:
	}
}

func newRecognizerForTest(listener SpeechRecognitionListener) *SpeechRecognizer {
	cred := common.NewCredential(1300000000, 1400000000, "test-secret")
	r := NewSpeechRecognizer(cred, "16k_zh_en", listener)
	r.SetWriteTimeout(200 * time.Millisecond)
	return r
}

func newWSPair(t *testing.T) (clientConn *websocket.Conn, serverConn *websocket.Conn, cleanup func()) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)
	serverErrCh := make(chan error, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErrCh <- err
			return
		}
		serverConnCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial websocket failed: %v", err)
	}

	var server *websocket.Conn
	select {
	case server = <-serverConnCh:
	case err = <-serverErrCh:
		_ = client.Close()
		srv.Close()
		t.Fatalf("upgrade websocket failed: %v", err)
	case <-time.After(2 * time.Second):
		_ = client.Close()
		srv.Close()
		t.Fatal("timeout waiting for server websocket connection")
	}

	cleanup = func() {
		if client != nil {
			_ = client.Close()
		}
		if server != nil {
			_ = server.Close()
		}
		srv.Close()
	}
	return client, server, cleanup
}

func expectASRErrorCode(t *testing.T, err error, wantCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", wantCode)
	}
	var asrErr *common.ASRError
	if !errors.As(err, &asrErr) {
		t.Fatalf("expected *common.ASRError, got %T (%v)", err, err)
	}
	if asrErr.Code != wantCode {
		t.Fatalf("error code = %d, want %d, error=%v", asrErr.Code, wantCode, asrErr)
	}
}

func TestWriteBeforeStartReturnsNotStarted(t *testing.T) {
	r := newRecognizerForTest(newTestListener())

	err := r.Write([]byte("abc"))
	expectASRErrorCode(t, err, common.ErrCodeNotStarted)
}

func TestStopWithNilConnectionReturnsNotStartedAndStops(t *testing.T) {
	r := newRecognizerForTest(newTestListener())
	atomic.StoreInt32(&r.state, stateRunning)
	r.conn = nil

	err := r.Stop()
	expectASRErrorCode(t, err, common.ErrCodeNotStarted)

	if got := atomic.LoadInt32(&r.state); got != stateStopped {
		t.Fatalf("state = %d, want %d", got, stateStopped)
	}
}

func TestStopSendFailureReturnsWriteFailedAndStops(t *testing.T) {
	listener := newTestListener()
	r := newRecognizerForTest(listener)
	client, _, cleanup := newWSPair(t)
	defer cleanup()

	r.conn = client
	atomic.StoreInt32(&r.state, stateRunning)

	_ = client.Close()

	err := r.Stop()
	expectASRErrorCode(t, err, common.ErrCodeWriteFailed)

	if got := atomic.LoadInt32(&r.state); got != stateStopped {
		t.Fatalf("state = %d, want %d", got, stateStopped)
	}
}

func TestReadLoopServerErrorTriggersOnFail(t *testing.T) {
	listener := newTestListener()
	r := newRecognizerForTest(listener)
	client, server, cleanup := newWSPair(t)
	defer cleanup()

	r.conn = client
	atomic.StoreInt32(&r.state, stateRunning)
	go r.readLoop()

	err := server.WriteMessage(websocket.TextMessage, []byte(`{"code":4006,"message":"quota exceeded","voice_id":"v1","result":{}}`))
	if err != nil {
		t.Fatalf("server write failed: %v", err)
	}

	select {
	case evt := <-listener.failCh:
		if evt.resp == nil {
			t.Fatal("expected non-nil response in OnFail")
		}
		if evt.resp.Code != 4006 {
			t.Fatalf("resp.Code = %d, want 4006", evt.resp.Code)
		}
		var asrErr *common.ASRError
		if !errors.As(evt.err, &asrErr) {
			t.Fatalf("expected *common.ASRError, got %T", evt.err)
		}
		if asrErr.Code != 4006 {
			t.Fatalf("OnFail error code = %d, want 4006", asrErr.Code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OnFail")
	}

	select {
	case <-r.doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for doneCh close")
	}
}

func TestReadLoopFinalTriggersOnRecognitionComplete(t *testing.T) {
	listener := newTestListener()
	r := newRecognizerForTest(listener)
	client, server, cleanup := newWSPair(t)
	defer cleanup()

	r.conn = client
	atomic.StoreInt32(&r.state, stateRunning)
	go r.readLoop()

	msg := `{"code":0,"message":"ok","voice_id":"v1","final":1,"result":{"slice_type":2,"index":0,"voice_text_str":"hello"}}`
	if err := server.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatalf("server write failed: %v", err)
	}

	select {
	case resp := <-listener.completeCh:
		if resp == nil {
			t.Fatal("expected non-nil response in OnRecognitionComplete")
		}
		if resp.Final != 1 {
			t.Fatalf("resp.Final = %d, want 1", resp.Final)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OnRecognitionComplete")
	}

	select {
	case <-r.doneCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for doneCh close")
	}
}

func TestConcurrentStopAndReadLoopCloseNoDeadlock(t *testing.T) {
	listener := newTestListener()
	r := newRecognizerForTest(listener)
	client, server, cleanup := newWSPair(t)
	defer cleanup()

	r.conn = client
	atomic.StoreInt32(&r.state, stateRunning)
	go r.readLoop()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		_, _, err := server.ReadMessage()
		if err != nil {
			return
		}
		_ = server.WriteMessage(websocket.TextMessage, []byte(`{"code":0,"message":"ok","voice_id":"v1","final":1,"result":{"slice_type":2}}`))
		_ = server.Close()
	}()

	stopErrCh := make(chan error, 1)
	go func() {
		stopErrCh <- r.Stop()
	}()

	select {
	case err := <-stopErrCh:
		if err != nil {
			t.Fatalf("Stop returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for Stop (possible deadlock)")
	}

	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server goroutine")
	}

	if got := atomic.LoadInt32(&r.state); got != stateStopped {
		t.Fatalf("state = %d, want %d", got, stateStopped)
	}
}

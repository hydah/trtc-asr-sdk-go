package asr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// sentenceEndCaptureListener records the last OnSentenceEnd payload so the
// decoded speaker fields can be asserted.
type sentenceEndCaptureListener struct {
	*testListener
	endCh chan *SpeechRecognitionResponse
}

func newSentenceEndCaptureListener() *sentenceEndCaptureListener {
	return &sentenceEndCaptureListener{
		testListener: newTestListener(),
		endCh:        make(chan *SpeechRecognitionResponse, 4),
	}
}

func (l *sentenceEndCaptureListener) OnSentenceEnd(resp *SpeechRecognitionResponse) {
	l.testListener.OnSentenceEnd(resp)
	select {
	case l.endCh <- resp:
	default:
	}
}

// TestReadLoopDecodesSpeakerDiarizationResult feeds the documented
// speaker_diarization=3 response shape and asserts the SDK exposes every
// speaker field (segment level, word level and role names).
func TestReadLoopDecodesSpeakerDiarizationResult(t *testing.T) {
	listener := newSentenceEndCaptureListener()
	r := newRecognizerForTest(listener)
	client, server, cleanup := newWSPair(t)
	defer cleanup()

	r.conn = client
	atomic.StoreInt32(&r.state, stateRunning)
	go r.readLoop()

	msg := `{"code":0,"message":"ok","voice_id":"v1","result":{
		"slice_type":2,"index":1,"start_time":3640,"end_time":6600,
		"voice_text_str":"你好 嗯我想咨询一下","word_size":3,
		"finish_silence_ms":800,"last_token_runtime_ms":42,
		"word_list":[
			{"word":"你","start_time":3640,"end_time":3760,"stable_flag":1,"speaker_id":1,"speaker_name":"teacher"},
			{"word":"好","start_time":3760,"end_time":3880,"stable_flag":1,"speaker_id":1,"speaker_name":"teacher"},
			{"word":"嗯","start_time":5400,"end_time":5550,"stable_flag":1,"speaker_id":2,"speaker_name":"student"}],
		"speaker_segments":[
			{"speaker_id":1,"speaker_name":"teacher","start_time":3640,"end_time":3880,"text":"你好","word_start":0,"word_end":1,"stable_flag":1},
			{"speaker_id":2,"speaker_name":"student","start_time":5400,"end_time":6600,"text":"嗯我想咨询一下","stable_flag":0}]}}`

	if err := server.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatalf("server write failed: %v", err)
	}

	var resp *SpeechRecognitionResponse
	select {
	case resp = <-listener.endCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OnSentenceEnd")
	}

	if got := len(resp.Result.SpeakerSegments); got != 2 {
		t.Fatalf("len(SpeakerSegments) = %d, want 2", got)
	}
	first := resp.Result.SpeakerSegments[0]
	if first.SpeakerID != 1 || first.SpeakerName != "teacher" {
		t.Errorf("segment[0] speaker = (%d, %q), want (1, teacher)", first.SpeakerID, first.SpeakerName)
	}
	if first.Text != "你好" || first.StartTime != 3640 || first.EndTime != 3880 || first.StableFlag != 1 {
		t.Errorf("segment[0] = %+v", first)
	}
	if first.WordStart == nil || *first.WordStart != 0 || first.WordEnd == nil || *first.WordEnd != 1 {
		t.Errorf("segment[0] word range = (%v, %v), want (0, 1)", first.WordStart, first.WordEnd)
	}

	second := resp.Result.SpeakerSegments[1]
	// word_info-less segments omit the indexes; nil must be preserved so the
	// caller can tell "no index" from index 0.
	if second.WordStart != nil || second.WordEnd != nil {
		t.Errorf("segment[1] word range = (%v, %v), want (nil, nil)", second.WordStart, second.WordEnd)
	}
	if second.StableFlag != 0 {
		t.Errorf("segment[1].StableFlag = %d, want 0", second.StableFlag)
	}

	if got := len(resp.Result.WordList); got != 3 {
		t.Fatalf("len(WordList) = %d, want 3", got)
	}
	if w := resp.Result.WordList[2]; w.SpeakerID != 2 || w.SpeakerName != "student" {
		t.Errorf("word[2] speaker = (%d, %q), want (2, student)", w.SpeakerID, w.SpeakerName)
	}

	if resp.Result.FinishSilenceMs != 800 {
		t.Errorf("FinishSilenceMs = %d, want 800", resp.Result.FinishSilenceMs)
	}
	if resp.Result.LastTokenRuntimeMs != 42 {
		t.Errorf("LastTokenRuntimeMs = %d, want 42", resp.Result.LastTokenRuntimeMs)
	}
	// The sentence-level speaker stays absent on this engine; a nil pointer
	// distinguishes that from speaker 0.
	if resp.Result.SpeakerID != nil {
		t.Errorf("Result.SpeakerID = %d, want nil (absent)", *resp.Result.SpeakerID)
	}
}

// TestConnectSendsSpeakerAndVadParams verifies the recognizer options reach the
// WebSocket handshake query string.
func TestConnectSendsSpeakerAndVadParams(t *testing.T) {
	var (
		mu       sync.Mutex
		gotQuery url.Values
		headers  http.Header
	)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		gotQuery = req.URL.Query()
		headers = req.Header.Clone()
		mu.Unlock()
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer srv.Close()

	r := newRecognizerForTest(newTestListener())
	r.endpoint = strings.Replace(srv.URL, "http://", "ws://", 1)
	r.SetVoiceID("voice-diarization")
	r.SetWordInfo(1)
	r.SetSpeakerDiarization(SpeakerDiarizationVoiceprint)
	r.SetSpeakerNumber(2)
	r.SetSpeakerRoles([]SpeakerRole{{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"}})
	r.SetVoiceprintIDs([]string{"vp-1"})
	r.SetVadLevel(0)
	r.SetNoiseThreshold(1.5)
	r.SetFilterEmptyResult(0)
	r.SetHotwordList("腾讯云|5")
	r.SetReplaceTextID("replace-1")
	r.SetInputSampleRate(8000)

	if err := r.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer r.Stop()

	mu.Lock()
	q := gotQuery
	gotHeaders := headers
	mu.Unlock()

	want := map[string]string{
		"speaker_diarization": "3",
		"speaker_number":      "2",
		"voiceprintids":       `["vp-1"]`,
		"vad_level":           "0",
		"noise_threshold":     "1.500",
		"filter_empty_result": "0",
		"hotword_list":        "腾讯云|5",
		"replace_text_id":     "replace-1",
		"input_sample_rate":   "8000",
		"word_info":           "1",
		// Authentication identity travels in the query string (browser
		// WebSocket clients cannot attach custom headers).
		"sdkappid": "1400000000",
	}
	for key, wantValue := range want {
		if got := q.Get(key); got != wantValue {
			t.Errorf("query %s = %q, want %q", key, got, wantValue)
		}
	}

	// signature and usersig carry the same UserSig value.
	if sig, sigQ := q.Get("signature"), q.Get("usersig"); sig == "" || sigQ == "" || sig != sigQ {
		t.Errorf("signature/usersig = %q/%q, want equal non-empty values", sig, sigQ)
	}

	// No auth identity in headers any more.
	if got := gotHeaders.Get("X-TRTC-SdkAppId"); got != "" {
		t.Errorf("X-TRTC-SdkAppId header = %q, want empty (query-only auth)", got)
	}
	if got := gotHeaders.Get("X-TRTC-UserSig"); got != "" {
		t.Errorf("X-TRTC-UserSig header = %q, want empty (query-only auth)", got)
	}

	var roles []SpeakerRole
	if err := json.Unmarshal([]byte(q.Get("speaker_roles")), &roles); err != nil {
		t.Fatalf("speaker_roles is not valid JSON: %v (raw %q)", err, q.Get("speaker_roles"))
	}
	if len(roles) != 1 || roles[0].RoleName != "teacher" {
		t.Errorf("speaker_roles decoded as %+v", roles)
	}
}

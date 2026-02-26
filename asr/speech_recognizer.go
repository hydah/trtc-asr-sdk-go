// Package asr provides the TRTC-ASR real-time speech recognition client.
//
// Usage:
//
//	credential := common.NewCredential(appID, sdkAppID, secretKey)
//	listener := &MyListener{}
//	recognizer := asr.NewSpeechRecognizer(credential, engineModelType, listener)
//	recognizer.Start()
//	recognizer.Write(audioData)
//	recognizer.Stop()
package asr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hydah/trtc-asr-sdk-go/common"
)

// Endpoint is the production WebSocket endpoint for the TRTC-ASR service.
const Endpoint = "wss://asr.cloud-rtc.com"

// Recognizer states.
const (
	stateIdle     int32 = 0
	stateStarting int32 = 1
	stateRunning  int32 = 2
	stateStopping int32 = 3
	stateStopped  int32 = 4
)

// SpeechRecognitionListener defines the callback interface for speech recognition events.
type SpeechRecognitionListener interface {
	// OnRecognitionStart is called when the recognition session starts successfully.
	OnRecognitionStart(response *SpeechRecognitionResponse)
	// OnSentenceBegin is called when a new sentence begins.
	OnSentenceBegin(response *SpeechRecognitionResponse)
	// OnRecognitionResultChange is called when intermediate recognition results are available.
	OnRecognitionResultChange(response *SpeechRecognitionResponse)
	// OnSentenceEnd is called when a sentence ends with the final result.
	OnSentenceEnd(response *SpeechRecognitionResponse)
	// OnRecognitionComplete is called when the entire recognition session completes.
	OnRecognitionComplete(response *SpeechRecognitionResponse)
	// OnFail is called when an error occurs during recognition.
	OnFail(response *SpeechRecognitionResponse, err error)
}

// SpeechRecognitionResponse represents a response message from the ASR service.
type SpeechRecognitionResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	VoiceID   string `json:"voice_id"`
	MessageID string `json:"message_id"`
	Final     int    `json:"final"`
	Result    Result `json:"result"`
}

// Result contains the speech recognition result details.
type Result struct {
	SliceType   int        `json:"slice_type"`
	Index       int        `json:"index"`
	StartTime   int        `json:"start_time"`
	EndTime     int        `json:"end_time"`
	VoiceTextStr string   `json:"voice_text_str"`
	WordSize    int        `json:"word_size"`
	WordList    []WordInfo `json:"word_list"`
}

// WordInfo contains word-level recognition details.
type WordInfo struct {
	Word         string `json:"word"`
	StartTime    int    `json:"start_time"`
	EndTime      int    `json:"end_time"`
	StableFlag   int    `json:"stable_flag"`
}

// SpeechRecognizer is the main client for real-time speech recognition.
type SpeechRecognizer struct {
	credential *common.Credential
	listener   SpeechRecognitionListener
	conn       *websocket.Conn

	// Configuration
	endpoint        string
	engineModelType string
	voiceFormat     int
	needVad         int
	convertNumMode  int
	hotwordID       string
	customizationID string
	filterDirty     int
	filterModal     int
	filterPunc      int
	wordInfo        int
	vadSilenceTime  int
	maxSpeakTime    int
	voiceID         string

	// State management
	state    int32
	mu       sync.Mutex
	doneCh   chan struct{}
	errCh    chan error

	// Write timeout
	writeTimeout time.Duration
}

// NewSpeechRecognizer creates a new SpeechRecognizer instance.
//
// Parameters:
//   - credential: TRTC authentication credential
//   - engineModelType: recognition engine model (e.g., "16k_zh", "8k_zh", "16k_zh_en")
//   - listener: callback listener for recognition events
func NewSpeechRecognizer(
	credential *common.Credential,
	engineModelType string,
	listener SpeechRecognitionListener,
) *SpeechRecognizer {
	return &SpeechRecognizer{
		credential:      credential,
		listener:        listener,
		endpoint:        Endpoint,
		engineModelType: engineModelType,
		voiceFormat:     1, // PCM
		needVad:         1,
		convertNumMode:  1,
		writeTimeout:    5 * time.Second,
		doneCh:          make(chan struct{}),
		errCh:           make(chan error, 1),
	}
}

// SetVoiceFormat sets the audio encoding format.
// 1: PCM (default), 0: disable VAD
func (r *SpeechRecognizer) SetVoiceFormat(format int) {
	r.voiceFormat = format
}

// SetNeedVad sets whether to enable VAD (Voice Activity Detection).
// 0: disable, 1: enable (default)
func (r *SpeechRecognizer) SetNeedVad(needVad int) {
	r.needVad = needVad
}

// SetConvertNumMode sets the number conversion mode.
// 0: no conversion, 1: smart conversion (default), 3: math conversion
func (r *SpeechRecognizer) SetConvertNumMode(mode int) {
	r.convertNumMode = mode
}

// SetHotwordID sets the hotword list ID for biasing recognition.
func (r *SpeechRecognizer) SetHotwordID(id string) {
	r.hotwordID = id
}

// SetCustomizationID sets the custom language model ID.
func (r *SpeechRecognizer) SetCustomizationID(id string) {
	r.customizationID = id
}

// SetFilterDirty sets the profanity filter mode.
// 0: no filter (default), 1: filter, 2: replace with *
func (r *SpeechRecognizer) SetFilterDirty(mode int) {
	r.filterDirty = mode
}

// SetFilterModal sets the modal particle filter mode.
// 0: no filter (default), 1: partial filter, 2: strict filter
func (r *SpeechRecognizer) SetFilterModal(mode int) {
	r.filterModal = mode
}

// SetFilterPunc sets the sentence-ending punctuation filter mode.
// 0: no filter (default), 1: filter
func (r *SpeechRecognizer) SetFilterPunc(mode int) {
	r.filterPunc = mode
}

// SetWordInfo sets whether to show word-level timing information.
// 0: no (default), 1: yes
func (r *SpeechRecognizer) SetWordInfo(mode int) {
	r.wordInfo = mode
}

// SetVadSilenceTime sets the silence detection threshold in milliseconds.
// Range: 240-1000, default: 1000
func (r *SpeechRecognizer) SetVadSilenceTime(ms int) {
	r.vadSilenceTime = ms
}

// SetMaxSpeakTime sets the maximum speech time in milliseconds.
// Range: 5000-90000, default: 60000
func (r *SpeechRecognizer) SetMaxSpeakTime(ms int) {
	r.maxSpeakTime = ms
}

// SetVoiceID sets a custom voice ID. If not set, a UUID will be generated.
func (r *SpeechRecognizer) SetVoiceID(id string) {
	r.voiceID = id
}

// SetWriteTimeout sets the timeout for writing audio data.
func (r *SpeechRecognizer) SetWriteTimeout(timeout time.Duration) {
	r.writeTimeout = timeout
}

// Start initiates the WebSocket connection and begins the recognition session.
// It returns an error if the connection fails or the recognizer is already running.
func (r *SpeechRecognizer) Start() error {
	if !atomic.CompareAndSwapInt32(&r.state, stateIdle, stateStarting) {
		return common.NewASRError(common.ErrCodeAlreadyStarted, "recognizer already started")
	}

	if err := r.connect(); err != nil {
		atomic.StoreInt32(&r.state, stateIdle)
		return err
	}

	atomic.StoreInt32(&r.state, stateRunning)

	// Start reading responses in background
	go r.readLoop()

	return nil
}

// Write sends audio data to the ASR service for recognition.
// The data should be in the format specified by SetVoiceFormat (default: PCM).
func (r *SpeechRecognizer) Write(data []byte) error {
	if atomic.LoadInt32(&r.state) != stateRunning {
		return common.NewASRError(common.ErrCodeNotStarted, "recognizer not running")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.conn == nil {
		return common.NewASRError(common.ErrCodeNotStarted, "connection not established")
	}

	r.conn.SetWriteDeadline(time.Now().Add(r.writeTimeout))
	if err := r.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return common.NewASRErrorf(common.ErrCodeWriteFailed, "write audio data failed: %v", err)
	}

	return nil
}

// Stop gracefully stops the recognition session.
// It sends the end signal and waits for the server to complete.
func (r *SpeechRecognizer) Stop() error {
	if !atomic.CompareAndSwapInt32(&r.state, stateRunning, stateStopping) {
		return common.NewASRError(common.ErrCodeNotStarted, "recognizer not running")
	}

	r.mu.Lock()
	conn := r.conn
	r.mu.Unlock()

	if conn == nil {
		atomic.StoreInt32(&r.state, stateStopped)
		return common.NewASRError(common.ErrCodeNotStarted, "connection not established")
	}

	// Send end signal: empty text message
	endMsg := map[string]string{"type": "end"}
	data, _ := json.Marshal(endMsg)
	r.mu.Lock()
	conn.SetWriteDeadline(time.Now().Add(r.writeTimeout))
	err := conn.WriteMessage(websocket.TextMessage, data)
	r.mu.Unlock()

	if err != nil {
		r.close()
		atomic.StoreInt32(&r.state, stateStopped)
		return common.NewASRErrorf(common.ErrCodeWriteFailed, "send end signal failed: %v", err)
	}

	// Wait for readLoop to finish with timeout
	select {
	case <-r.doneCh:
	case <-time.After(10 * time.Second):
		r.close()
	}

	atomic.StoreInt32(&r.state, stateStopped)
	return nil
}

func (r *SpeechRecognizer) connect() error {
	voiceID := r.voiceID
	if voiceID == "" {
		voiceID = uuid.New().String()
		r.voiceID = voiceID
	}

	// Generate UserSig if not already set
	if r.credential.UserSig == "" {
		userSig, err := common.GenUserSig(r.credential.SdkAppID, r.credential.SecretKey, voiceID, 86400)
		if err != nil {
			return common.NewASRErrorf(common.ErrCodeAuthFailed, "generate user sig failed: %v", err)
		}
		r.credential.UserSig = userSig
	}

	// Build request parameters (AppID is used for URL secretid parameter)
	sigParams := common.NewSignatureParams(r.credential.AppID, r.engineModelType, voiceID)
	sigParams.VoiceFormat = r.voiceFormat
	sigParams.NeedVad = r.needVad
	sigParams.ConvertNumMode = r.convertNumMode
	sigParams.HotwordID = r.hotwordID
	sigParams.CustomizationID = r.customizationID
	sigParams.FilterDirty = r.filterDirty
	sigParams.FilterModal = r.filterModal
	sigParams.FilterPunc = r.filterPunc
	sigParams.WordInfo = r.wordInfo
	sigParams.VadSilenceTime = r.vadSilenceTime
	sigParams.MaxSpeakTime = r.maxSpeakTime

	// Per protocol: signature = UserSig
	queryString := sigParams.BuildQueryStringWithSignature(r.credential.UserSig)
	// URL path uses Tencent Cloud AppID (not SdkAppID)
	wsURL := fmt.Sprintf("%s/asr/v2/%d?%s", r.endpoint, r.credential.AppID, queryString)

	// Build WebSocket headers
	header := http.Header{}
	header.Set("X-TRTC-SdkAppId", fmt.Sprintf("%d", r.credential.SdkAppID))
	header.Set("X-TRTC-UserSig", r.credential.UserSig)

	// Create WebSocket dialer
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return common.NewASRErrorf(common.ErrCodeConnectFailed, "websocket dial failed: %v", err)
	}

	r.conn = conn
	return nil
}

func (r *SpeechRecognizer) readLoop() {
	defer func() {
		close(r.doneCh)
		r.close()
	}()

	for {
		_, message, err := r.conn.ReadMessage()
		if err != nil {
			if atomic.LoadInt32(&r.state) >= stateStopping {
				return
			}
			r.listener.OnFail(nil, common.NewASRErrorf(common.ErrCodeReadFailed, "read message failed: %v", err))
			return
		}

		var resp SpeechRecognitionResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			r.listener.OnFail(nil, common.NewASRErrorf(common.ErrCodeReadFailed, "unmarshal response failed: %v", err))
			continue
		}

		if resp.Code != 0 {
			r.listener.OnFail(&resp, common.NewASRError(resp.Code, resp.Message))
			return
		}

		r.dispatchEvent(&resp)

		// Check if recognition is complete
		if resp.Final == 1 {
			r.listener.OnRecognitionComplete(&resp)
			return
		}
	}
}

func (r *SpeechRecognizer) dispatchEvent(resp *SpeechRecognitionResponse) {
	switch resp.Result.SliceType {
	case 0:
		r.listener.OnSentenceBegin(resp)
	case 1:
		r.listener.OnRecognitionResultChange(resp)
	case 2:
		r.listener.OnSentenceEnd(resp)
	default:
		// If Final=1 and no slice_type, it's a completion event
		if resp.Final == 1 {
			return // handled in readLoop
		}
		// First message is usually the start event
		r.listener.OnRecognitionStart(resp)
	}
}

func (r *SpeechRecognizer) close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}



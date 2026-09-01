package asr

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSentenceRecognizer_Recognize_CustomizationAndLanguageBody asserts the
// sentence-recognition fields are serialized with the server-side names.
func TestSentenceRecognizer_Recognize_CustomizationAndLanguageBody(t *testing.T) {
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body failed: %v", err)
		}
		rawBody = body

		w.Header().Set("Content-Type", "application/json")
		resp := SentenceRecognitionResponse{Response: &SentenceRecognitionResult{
			Result:    "ok",
			RequestId: "req-1",
		}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	recognizer := NewSentenceRecognizer(newTestCredential())
	recognizer.SetEndpoint(server.URL)

	_, err := recognizer.Recognize(&SentenceRecognitionRequest{
		EngServiceType:  "16k_zh",
		SourceType:      SourceTypeURL,
		VoiceFormat:     "wav",
		Url:             "https://example.com/a.wav",
		CustomizationID: "custom-1",
		Language:        "zh",
	})
	if err != nil {
		t.Fatalf("Recognize failed: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatalf("request body is not valid JSON: %v (raw %s)", err, rawBody)
	}

	if body["CustomizationId"] != "custom-1" {
		t.Errorf("CustomizationId = %v, want custom-1", body["CustomizationId"])
	}
	if body["Language"] != "zh" {
		t.Errorf("Language = %v, want zh", body["Language"])
	}
	// Sentence recognition does not support speaker diarization; the request
	// struct must not grow those fields.
	if _, ok := body["SpeakerDiarization"]; ok {
		t.Errorf("SpeakerDiarization must not be sent by sentence recognition: %s", rawBody)
	}
}

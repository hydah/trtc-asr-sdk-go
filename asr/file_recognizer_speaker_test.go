package asr

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFileRecognizer_CreateTask_SpeakerDiarizationBody asserts the new
// diarization / VAD tuning fields are serialized with the server-side names
// and that an explicit zero threshold is still sent.
func TestFileRecognizer_CreateTask_SpeakerDiarizationBody(t *testing.T) {
	var rawBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body failed: %v", err)
		}
		rawBody = body

		resp := CreateRecTaskResponse{}
		resp.Response.Data = &CreateRecTaskData{RecTaskId: "task-1"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	recognizer := NewFileRecognizer(newTestCredential())
	recognizer.SetEndpoint(server.URL)

	vadLevel := 0
	noiseThreshold := 0.0
	req := &CreateRecTaskRequest{
		EngineModelType:    "16k_zh",
		ChannelNum:         1,
		ResTextFormat:      1,
		SourceType:         SourceTypeURL,
		Url:                "https://example.com/audio.wav",
		SpeakerDiarization: SpeakerDiarizationVoiceprint,
		SpeakerNumber:      2,
		SpeakerRoles:       []SpeakerRole{{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"}},
		VoiceprintIds:      []string{"vp-1"},
		VadSilenceMs:       800,
		VadLevel:           &vadLevel,
		NoiseThreshold:     &noiseThreshold,
		Language:           "zh",
		ReplaceTextId:      "replace-1",
	}

	if _, err := recognizer.CreateTask(req); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rawBody, &body); err != nil {
		t.Fatalf("request body is not valid JSON: %v (raw %s)", err, rawBody)
	}

	wantNumbers := map[string]float64{
		"SpeakerDiarization": 3,
		"SpeakerNumber":      2,
		"VadSilenceMs":       800,
		// Explicit zeros must survive: *int / *float64 keep them out of
		// omitempty's blind spot.
		"VadLevel":       0,
		"NoiseThreshold": 0,
	}
	for key, want := range wantNumbers {
		got, ok := body[key]
		if !ok {
			t.Errorf("request body missing %s: %s", key, rawBody)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", key, got, want)
		}
	}

	if body["Language"] != "zh" || body["ReplaceTextId"] != "replace-1" {
		t.Errorf("Language/ReplaceTextId = %v/%v", body["Language"], body["ReplaceTextId"])
	}

	roles, ok := body["SpeakerRoles"].([]interface{})
	if !ok || len(roles) != 1 {
		t.Fatalf("SpeakerRoles = %v", body["SpeakerRoles"])
	}
	role := roles[0].(map[string]interface{})
	if role["RoleName"] != "teacher" || role["AudioUrl"] != "https://example.com/a.wav" {
		t.Errorf("SpeakerRoles[0] = %v", role)
	}

	ids, ok := body["VoiceprintIds"].([]interface{})
	if !ok || len(ids) != 1 || ids[0] != "vp-1" {
		t.Errorf("VoiceprintIds = %v", body["VoiceprintIds"])
	}
}

func TestFileRecognizer_CreateTask_RejectsInvalidDiarization(t *testing.T) {
	recognizer := NewFileRecognizer(newTestCredential())

	req := &CreateRecTaskRequest{
		EngineModelType: "16k_zh",
		ChannelNum:      1,
		SourceType:      SourceTypeURL,
		Url:             "https://example.com/audio.wav",
		SpeakerRoles:    []SpeakerRole{{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"}},
	}

	_, err := recognizer.CreateTask(req)
	if err == nil {
		t.Fatal("expected error when SpeakerRoles is used without SpeakerDiarization=3")
	}
	if !strings.Contains(err.Error(), "require SpeakerDiarization=3") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestFileRecognizer_DescribeTaskStatus_SpeakerFields covers the response side:
// per-sentence speaker number, enrolled role name and stereo channel id.
func TestFileRecognizer_DescribeTaskStatus_SpeakerFields(t *testing.T) {
	const respBody = `{"Response":{"RequestId":"req-1","Data":{
		"RecTaskId":"task-1","Status":2,"StatusStr":"success","Progress":100,
		"AudioDuration":12.5,"Result":"你好\n嗯",
		"ResultDetail":[
			{"FinalSentence":"你好","StartMs":0,"EndMs":1200,"WordsNum":2,
			 "Words":[{"Word":"你","OffsetStartMs":0,"OffsetEndMs":120}],
			 "SpeakerId":1,"SpeakerRoleName":"teacher","Language":"zh"},
			{"FinalSentence":"嗯","StartMs":1300,"EndMs":1500,"SpeakerId":2,"ChannelId":2}]}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	defer server.Close()

	recognizer := NewFileRecognizer(newTestCredential())
	recognizer.SetEndpoint(server.URL)

	status, err := recognizer.DescribeTaskStatus("task-1")
	if err != nil {
		t.Fatalf("DescribeTaskStatus failed: %v", err)
	}

	if status.Progress != 100 {
		t.Errorf("Progress = %d, want 100", status.Progress)
	}
	if len(status.ResultDetail) != 2 {
		t.Fatalf("len(ResultDetail) = %d, want 2", len(status.ResultDetail))
	}

	first := status.ResultDetail[0]
	if first.SpeakerId != 1 || first.SpeakerRoleName != "teacher" {
		t.Errorf("detail[0] speaker = (%d, %q), want (1, teacher)", first.SpeakerId, first.SpeakerRoleName)
	}
	if first.Language != "zh" {
		t.Errorf("detail[0].Language = %q, want zh", first.Language)
	}

	// Stereo recordings report the channel instead of a clustered speaker.
	if second := status.ResultDetail[1]; second.ChannelId != 2 {
		t.Errorf("detail[1].ChannelId = %d, want 2", second.ChannelId)
	}
}

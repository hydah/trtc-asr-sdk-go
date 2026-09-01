package common

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// queryValues parses the built query string so assertions read the decoded
// value (speaker_roles is URL-encoded JSON).
func queryValues(t *testing.T, qs string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(qs)
	if err != nil {
		t.Fatalf("ParseQuery(%q) failed: %v", qs, err)
	}
	return values
}

func TestBuildQueryString_OmitsUnsetOptionalParams(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	qs := params.BuildQueryString()

	for _, key := range []string{
		"speaker_diarization", "speaker_number", "speaker_roles", "voiceprintids",
		"noise_threshold", "vad_level", "filter_empty_result", "hotword_list",
		"replace_text_id", "input_sample_rate",
	} {
		if strings.Contains(qs, key+"=") {
			t.Errorf("query should not contain %s when unset: %s", key, qs)
		}
	}
}

func TestBuildQueryString_SpeakerDiarizationCluster(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	params.SpeakerDiarization = SpeakerDiarizationCluster
	// The speaker count hint feeds online clustering in both modes.
	params.SpeakerNumber = 2
	// Enrollment input only applies to mode 3 and must not leak into mode 1.
	params.SpeakerRoles = []SpeakerRole{{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"}}
	params.VoiceprintIDs = []string{"vp-1"}

	q := queryValues(t, params.BuildQueryString())

	if got := q.Get("speaker_diarization"); got != "1" {
		t.Errorf("speaker_diarization = %q, want 1", got)
	}
	if got := q.Get("speaker_number"); got != "2" {
		t.Errorf("speaker_number = %q, want 2", got)
	}
	for _, key := range []string{"speaker_roles", "voiceprintids"} {
		if _, ok := q[key]; ok {
			t.Errorf("%s should be omitted for speaker_diarization=1, got %q", key, q.Get(key))
		}
	}
}

func TestBuildQueryString_SpeakerDiarizationVoiceprint(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	params.SpeakerDiarization = SpeakerDiarizationVoiceprint
	params.SpeakerRoles = []SpeakerRole{
		{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"},
		{RoleName: "student", AudioUrl: "https://example.com/b.wav"},
	}
	params.VoiceprintIDs = []string{"vp-1", "vp-2"}

	params.SpeakerNumber = 0 // auto detection stays the server default

	q := queryValues(t, params.BuildQueryString())

	if got := q.Get("speaker_diarization"); got != "3" {
		t.Errorf("speaker_diarization = %q, want 3", got)
	}
	// 0 means auto detection; the server applies the same default, so the
	// parameter is omitted instead of being pinned to zero.
	if _, ok := q["speaker_number"]; ok {
		t.Errorf("speaker_number should be omitted when 0, got %q", q.Get("speaker_number"))
	}

	var roles []SpeakerRole
	if err := json.Unmarshal([]byte(q.Get("speaker_roles")), &roles); err != nil {
		t.Fatalf("speaker_roles is not valid JSON: %v (raw %q)", err, q.Get("speaker_roles"))
	}
	if len(roles) != 2 || roles[0].RoleName != "teacher" || roles[1].AudioUrl != "https://example.com/b.wav" {
		t.Errorf("speaker_roles decoded as %+v", roles)
	}

	var ids []string
	if err := json.Unmarshal([]byte(q.Get("voiceprintids")), &ids); err != nil {
		t.Fatalf("voiceprintids is not valid JSON: %v (raw %q)", err, q.Get("voiceprintids"))
	}
	if len(ids) != 2 || ids[0] != "vp-1" {
		t.Errorf("voiceprintids decoded as %v", ids)
	}
}

func TestBuildQueryString_TriStateVadTuning(t *testing.T) {
	zero := 0
	threshold := 0.0

	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	params.VadLevel = &zero
	params.NoiseThreshold = &threshold
	params.FilterEmptyResult = &zero

	q := queryValues(t, params.BuildQueryString())

	// An explicit 0 differs from "unset": the server defaults vad_level to 1
	// and filter_empty_result to 1, so both must reach the wire.
	if got := q.Get("vad_level"); got != "0" {
		t.Errorf("vad_level = %q, want 0", got)
	}
	if got := q.Get("filter_empty_result"); got != "0" {
		t.Errorf("filter_empty_result = %q, want 0", got)
	}
	if got := q.Get("noise_threshold"); got != "0.000" {
		t.Errorf("noise_threshold = %q, want 0.000", got)
	}

	threshold = 1.5
	params.NoiseThreshold = &threshold
	q = queryValues(t, params.BuildQueryString())
	if got := q.Get("noise_threshold"); got != "1.500" {
		t.Errorf("noise_threshold = %q, want 1.500", got)
	}
}

func TestBuildQueryString_AdvancedOptionalParams(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	params.HotwordList = "腾讯云|5,ASR|11"
	params.ReplaceTextID = "replace-1"
	params.InputSampleRate = 8000

	q := queryValues(t, params.BuildQueryString())

	want := map[string]string{
		"hotword_list":      "腾讯云|5,ASR|11",
		"replace_text_id":   "replace-1",
		"input_sample_rate": "8000",
	}
	for key, wantValue := range want {
		if got := q.Get(key); got != wantValue {
			t.Errorf("%s = %q, want %q", key, got, wantValue)
		}
	}
}

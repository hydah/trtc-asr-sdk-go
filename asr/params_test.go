package asr

import (
	"math"
	"strings"
	"testing"
)

func TestValidateSpeakerDiarization(t *testing.T) {
	validRole := SpeakerRole{RoleName: "teacher", AudioUrl: "https://example.com/a.wav"}

	tests := []struct {
		name          string
		mode          int
		speakerNumber int
		roles         []SpeakerRole
		voiceprintIDs []string
		wantErr       string
	}{
		{name: "off", mode: SpeakerDiarizationOff},
		{name: "cluster", mode: SpeakerDiarizationCluster},
		{name: "cluster with number hint", mode: SpeakerDiarizationCluster, speakerNumber: 2},
		{
			name:          "voiceprint with enrollment",
			mode:          SpeakerDiarizationVoiceprint,
			speakerNumber: 2,
			roles:         []SpeakerRole{validRole},
			voiceprintIDs: []string{"vp-1"},
		},
		{name: "unsupported mode", mode: 2, wantErr: "SpeakerDiarization must be 0"},
		{name: "negative speaker number", mode: SpeakerDiarizationCluster, speakerNumber: -1, wantErr: "SpeakerNumber must be >= 0"},
		{
			name:    "roles without voiceprint mode",
			mode:    SpeakerDiarizationCluster,
			roles:   []SpeakerRole{validRole},
			wantErr: "require SpeakerDiarization=3",
		},
		{
			name:          "voiceprint ids without voiceprint mode",
			mode:          SpeakerDiarizationOff,
			voiceprintIDs: []string{"vp-1"},
			wantErr:       "require SpeakerDiarization=3",
		},
		{
			name:    "empty role name",
			mode:    SpeakerDiarizationVoiceprint,
			roles:   []SpeakerRole{{AudioUrl: "https://example.com/a.wav"}},
			wantErr: "RoleName is empty",
		},
		{
			name:    "empty audio url",
			mode:    SpeakerDiarizationVoiceprint,
			roles:   []SpeakerRole{{RoleName: "teacher"}},
			wantErr: "AudioUrl is empty",
		},
		{
			name:    "non http scheme",
			mode:    SpeakerDiarizationVoiceprint,
			roles:   []SpeakerRole{{RoleName: "teacher", AudioUrl: "file:///etc/passwd"}},
			wantErr: "must use http or https",
		},
		{
			name:    "url without host",
			mode:    SpeakerDiarizationVoiceprint,
			roles:   []SpeakerRole{{RoleName: "teacher", AudioUrl: "https:///a.wav"}},
			wantErr: "has no host",
		},
		// This SDK is customer-facing: internal hosts belong to the caller's own
		// network and stay fetchable for the service, so no SSRF-style blocking.
		{
			name:    "internal host allowed",
			mode:    SpeakerDiarizationVoiceprint,
			roles:   []SpeakerRole{{RoleName: "teacher", AudioUrl: "http://192.168.1.10/a.wav"}},
			wantErr: "",
		},
		{
			name:          "empty voiceprint id",
			mode:          SpeakerDiarizationVoiceprint,
			voiceprintIDs: []string{""},
			wantErr:       "VoiceprintIds[0] is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpeakerDiarization(tt.mode, tt.speakerNumber, tt.roles, tt.voiceprintIDs)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("expected no error, got %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateVadTuning(t *testing.T) {
	ptrInt := func(v int) *int { return &v }
	ptrFloat := func(v float64) *float64 { return &v }

	tests := []struct {
		name      string
		vadLevel  *int
		threshold *float64
		wantErr   string
	}{
		{name: "unset"},
		{name: "high recall", vadLevel: ptrInt(0)},
		{name: "far field", vadLevel: ptrInt(1)},
		{name: "threshold lower bound", threshold: ptrFloat(0)},
		{name: "threshold upper bound", threshold: ptrFloat(4)},
		{name: "invalid level", vadLevel: ptrInt(2), wantErr: "VadLevel must be 0"},
		{name: "threshold too small", threshold: ptrFloat(-0.5), wantErr: "NoiseThreshold must be between"},
		{name: "threshold too large", threshold: ptrFloat(4.5), wantErr: "NoiseThreshold must be between"},
		{name: "threshold NaN", threshold: ptrFloat(math.NaN()), wantErr: "NoiseThreshold must be between"},
		{name: "threshold Inf", threshold: ptrFloat(math.Inf(1)), wantErr: "NoiseThreshold must be between"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVadTuning(tt.vadLevel, tt.threshold)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("expected no error, got %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestSpeechRecognizerStartRejectsInvalidOptions(t *testing.T) {
	r := newRecognizerForTest(newTestListener())
	r.SetSpeakerDiarization(2)

	err := r.Start()
	if err == nil {
		t.Fatal("expected Start to reject an invalid diarization mode")
	}
	if !strings.Contains(err.Error(), "SpeakerDiarization must be 0") {
		t.Fatalf("unexpected error: %v", err)
	}
	// A rejected Start must leave the recognizer reusable after fixing options.
	if got := r.state; got != stateIdle {
		t.Fatalf("state = %d, want stateIdle(%d)", got, stateIdle)
	}
}

func TestSpeechRecognizerStartRejectsNoiseThresholdOutOfRange(t *testing.T) {
	r := newRecognizerForTest(newTestListener())
	r.SetNoiseThreshold(5)

	err := r.Start()
	if err == nil {
		t.Fatal("expected Start to reject an out-of-range noise threshold")
	}
	if !strings.Contains(err.Error(), "NoiseThreshold must be between") {
		t.Fatalf("unexpected error: %v", err)
	}
}

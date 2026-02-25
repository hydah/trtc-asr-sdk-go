package common

import (
	"strings"
	"testing"
)

func TestNewSignatureParams(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")

	if params.AppID != 1300403317 {
		t.Errorf("AppID = %d, want 1300403317", params.AppID)
	}
	if params.EngineModelType != "16k_zh" {
		t.Errorf("EngineModelType = %s, want 16k_zh", params.EngineModelType)
	}
	if params.VoiceID != "voice-001" {
		t.Errorf("VoiceID = %s, want voice-001", params.VoiceID)
	}
	if params.VoiceFormat != 1 {
		t.Errorf("VoiceFormat = %d, want 1", params.VoiceFormat)
	}
	if params.NeedVad != 1 {
		t.Errorf("NeedVad = %d, want 1", params.NeedVad)
	}
	if params.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
	if params.Expired <= params.Timestamp {
		t.Error("Expired should be greater than Timestamp")
	}
}

func TestBuildQueryString(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	qs := params.BuildQueryString()

	if qs == "" {
		t.Fatal("BuildQueryString returned empty string")
	}

	requiredKeys := []string{"secretid=", "timestamp=", "expired=", "nonce=", "engine_model_type=", "voice_id="}
	for _, key := range requiredKeys {
		if !strings.Contains(qs, key) {
			t.Errorf("BuildQueryString missing key: %s", key)
		}
	}

	// secretid should be the Tencent Cloud AppID
	if !strings.Contains(qs, "secretid=1300403317") {
		t.Error("secretid should equal AppID as string")
	}

	if strings.Contains(qs, "signature=") {
		t.Error("BuildQueryString should NOT contain signature")
	}
}

func TestBuildQueryStringWithSignature(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	userSig := "eJwtzDEOgCAQRdG9UBMH-test-user-sig"
	qs := params.BuildQueryStringWithSignature(userSig)

	if qs == "" {
		t.Fatal("BuildQueryStringWithSignature returned empty string")
	}

	if !strings.Contains(qs, "signature=") {
		t.Error("BuildQueryStringWithSignature missing signature parameter")
	}

	requiredKeys := []string{"secretid=", "timestamp=", "expired=", "nonce="}
	for _, key := range requiredKeys {
		if !strings.Contains(qs, key) {
			t.Errorf("BuildQueryStringWithSignature missing key: %s", key)
		}
	}
}

func TestSecretKeyNotInParams(t *testing.T) {
	params := NewSignatureParams(1300403317, "16k_zh", "voice-001")
	qs := params.BuildQueryStringWithSignature("some-user-sig")

	if strings.Contains(qs, "secret_key") || strings.Contains(qs, "secretkey") {
		t.Error("SecretKey should never appear in query string parameters")
	}
}

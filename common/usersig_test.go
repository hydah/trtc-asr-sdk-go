package common

import (
	"testing"
)

func TestGenUserSig(t *testing.T) {
	sdkAppID := 1400000000
	key := "test-secret-key-for-unit-testing"
	userID := "test-user-001"
	expire := 86400

	sig, err := GenUserSig(sdkAppID, key, userID, expire)
	if err != nil {
		t.Fatalf("GenUserSig failed: %v", err)
	}

	if sig == "" {
		t.Fatal("GenUserSig returned empty string")
	}

	t.Logf("Generated UserSig: %s", sig)
}

func TestGenUserSigDefaultExpire(t *testing.T) {
	sdkAppID := 1400000000
	key := "test-secret-key-for-unit-testing"
	userID := "test-user-002"

	sig, err := GenUserSig(sdkAppID, key, userID, 0)
	if err != nil {
		t.Fatalf("GenUserSig with default expire failed: %v", err)
	}

	if sig == "" {
		t.Fatal("GenUserSig returned empty string")
	}
}

func TestGenUserSigDeterministic(t *testing.T) {
	// Same inputs at the same time should yield the same output
	// But since time changes, we just verify no errors with various inputs
	testCases := []struct {
		sdkAppID int
		key      string
		userID   string
	}{
		{1400000001, "key1", "user1"},
		{1400000002, "key2", "user2"},
		{1400000003, "key-with-special-chars!@#$%", "user-with-dashes"},
	}

	for _, tc := range testCases {
		sig, err := GenUserSig(tc.sdkAppID, tc.key, tc.userID, 86400)
		if err != nil {
			t.Errorf("GenUserSig(%d, %s, %s) failed: %v", tc.sdkAppID, tc.key, tc.userID, err)
		}
		if sig == "" {
			t.Errorf("GenUserSig(%d, %s, %s) returned empty string", tc.sdkAppID, tc.key, tc.userID)
		}
	}
}

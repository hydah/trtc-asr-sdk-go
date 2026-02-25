// Package common provides shared types and utilities for TRTC-ASR SDK.
package common

import "fmt"

// Credential holds the authentication information for TRTC-ASR service.
//
// Three values are needed:
//   - AppID: Tencent Cloud account APPID, from https://console.cloud.tencent.com/cam/capi
//   - SdkAppID: TRTC application ID, from https://console.cloud.tencent.com/trtc/app
//   - SecretKey: TRTC SDK secret key, from TRTC console > Application Overview > SDK Key
type Credential struct {
	// AppID is the Tencent Cloud account APPID.
	// Used in the WebSocket URL path: wss://asr.cloud-rtc.com/asr/v2/<appid>
	// Obtain from: https://console.cloud.tencent.com/cam/capi
	AppID int

	// SdkAppID is the TRTC application ID.
	// Obtain from: https://console.cloud.tencent.com/trtc/app
	SdkAppID int

	// SecretKey is the TRTC SDK secret key for the application.
	// Used to generate UserSig. Never transmitted over the network.
	// Obtain from: TRTC console > Application Management > Application Overview > SDK Key
	SecretKey string

	// UserSig is the TRTC authentication signature (auto-generated if not set).
	UserSig string
}

// NewCredential creates a new Credential with the required authentication parameters.
//
// Parameters:
//   - appID: Tencent Cloud account APPID from CAM console
//   - sdkAppID: TRTC application ID from the TRTC console
//   - secretKey: SDK secret key from the TRTC application overview
func NewCredential(appID int, sdkAppID int, secretKey string) *Credential {
	return &Credential{
		AppID:     appID,
		SdkAppID:  sdkAppID,
		SecretKey: secretKey,
	}
}

// SetUserSig sets a pre-computed UserSig on the credential.
// If not set, the SDK will auto-generate it using SdkAppID and SecretKey.
func (c *Credential) SetUserSig(userSig string) {
	c.UserSig = userSig
}

// AppIDStr returns the AppID as a string.
func (c *Credential) AppIDStr() string {
	return fmt.Sprintf("%d", c.AppID)
}

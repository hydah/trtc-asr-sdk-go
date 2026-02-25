package common

import (
	"github.com/tencentyun/tls-sig-api-v2-golang/tencentyun"
)

const defaultExpire = 86400 * 180 // 180 days in seconds

// GenUserSig generates a TRTC UserSig using the official tls-sig-api-v2 library.
//
// Parameters:
//   - sdkAppID: TRTC application ID
//   - key: TRTC secret key (from console)
//   - userID: unique user identifier (maps to voice_id in ASR)
//   - expire: signature validity in seconds (0 uses default 180 days)
func GenUserSig(sdkAppID int, key, userID string, expire int) (string, error) {
	if expire <= 0 {
		expire = defaultExpire
	}
	return tencentyun.GenUserSig(sdkAppID, key, userID, expire)
}

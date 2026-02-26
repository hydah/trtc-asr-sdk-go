package common

import (
	"fmt"
	"math/rand"
	"net/url"
	"sort"
	"strings"
	"time"
)

// SignatureParams holds URL query parameters for the ASR WebSocket request.
// The "secretid" URL parameter is required by the protocol but internally
// populated with AppID — users do not need to provide a separate SecretID.
// The "signature" parameter is set to the UserSig value per protocol spec.
type SignatureParams struct {
	AppID           int
	Timestamp       int64
	Expired         int64
	Nonce           int
	EngineModelType string
	VoiceID         string
	VoiceFormat     int
	NeedVad         int

	// Optional parameters
	HotwordID       string
	CustomizationID string
	FilterDirty     int
	FilterModal     int
	FilterPunc      int
	ConvertNumMode  int
	WordInfo        int
	VadSilenceTime  int
	MaxSpeakTime    int
}

// NewSignatureParams creates SignatureParams with sensible defaults.
func NewSignatureParams(appID int, engineModelType, voiceID string) *SignatureParams {
	now := time.Now().Unix()
	return &SignatureParams{
		AppID:           appID,
		Timestamp:       now,
		Expired:         now + 86400,
		Nonce:           rand.Intn(9999999) + 1,
		EngineModelType: engineModelType,
		VoiceID:         voiceID,
		VoiceFormat:     1, // pcm
		NeedVad:         1,
		ConvertNumMode:  1,
	}
}

// BuildQueryString constructs the URL query string with all parameters (without signature).
func (p *SignatureParams) BuildQueryString() string {
	params := p.toMap()
	return encodeParams(params)
}

// BuildQueryStringWithSignature constructs the URL query string with signature set to the given userSig.
// Per protocol: "signature" value equals X-TRTC-UserSig.
func (p *SignatureParams) BuildQueryStringWithSignature(userSig string) string {
	params := p.toMap()
	params["signature"] = userSig
	return encodeParams(params)
}

func (p *SignatureParams) toMap() map[string]string {
	// "secretid" is required by protocol; internally use SdkAppID as its value.
	m := map[string]string{
		"secretid":          fmt.Sprintf("%d", p.AppID),
		"timestamp":         fmt.Sprintf("%d", p.Timestamp),
		"expired":           fmt.Sprintf("%d", p.Expired),
		"nonce":             fmt.Sprintf("%d", p.Nonce),
		"engine_model_type": p.EngineModelType,
		"voice_id":          p.VoiceID,
		"voice_format":      fmt.Sprintf("%d", p.VoiceFormat),
		"needvad":           fmt.Sprintf("%d", p.NeedVad),
	}

	if p.HotwordID != "" {
		m["hotword_id"] = p.HotwordID
	}
	if p.CustomizationID != "" {
		m["customization_id"] = p.CustomizationID
	}
	if p.FilterDirty != 0 {
		m["filter_dirty"] = fmt.Sprintf("%d", p.FilterDirty)
	}
	if p.FilterModal != 0 {
		m["filter_modal"] = fmt.Sprintf("%d", p.FilterModal)
	}
	if p.FilterPunc != 0 {
		m["filter_punc"] = fmt.Sprintf("%d", p.FilterPunc)
	}
	if p.ConvertNumMode != 0 {
		m["convert_num_mode"] = fmt.Sprintf("%d", p.ConvertNumMode)
	}
	if p.WordInfo != 0 {
		m["word_info"] = fmt.Sprintf("%d", p.WordInfo)
	}
	if p.VadSilenceTime != 0 {
		m["vad_silence_time"] = fmt.Sprintf("%d", p.VadSilenceTime)
	}
	if p.MaxSpeakTime != 0 {
		m["max_speak_time"] = fmt.Sprintf("%d", p.MaxSpeakTime)
	}

	return m
}

func encodeParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, url.QueryEscape(params[k])))
	}
	return strings.Join(parts, "&")
}

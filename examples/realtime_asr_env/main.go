// Package main demonstrates real-time speech recognition with environment variable configuration.
//
// Set the following environment variables:
//
//	export TRTC_APP_ID="1300403317"
//	export TRTC_SDK_APP_ID="1400188366"
//	export TRTC_SECRET_KEY="your-sdk-secret-key"
//
// Then run:
//
//	go run main.go -f audio.pcm
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/trtc-asr/trtc-asr-sdk-go/asr"
	"github.com/trtc-asr/trtc-asr-sdk-go/common"
)

const (
	envAppID     = "TRTC_APP_ID"
	envSdkAppID  = "TRTC_SDK_APP_ID"
	envSecretKey = "TRTC_SECRET_KEY"
)

var SliceSize = 6400

// ASRListener implements asr.SpeechRecognitionListener.
type ASRListener struct {
	ID int
}

func (l *ASRListener) OnRecognitionStart(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[Worker-%d] Recognition started | voice_id=%s", l.ID, resp.VoiceID)
}

func (l *ASRListener) OnSentenceBegin(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[Worker-%d] Sentence begin | index=%d", l.ID, resp.Result.Index)
}

func (l *ASRListener) OnRecognitionResultChange(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[Worker-%d] Intermediate result | index=%d text=\"%s\"",
		l.ID, resp.Result.Index, resp.Result.VoiceTextStr)
}

func (l *ASRListener) OnSentenceEnd(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[Worker-%d] Sentence end | index=%d text=\"%s\"",
		l.ID, resp.Result.Index, resp.Result.VoiceTextStr)
}

func (l *ASRListener) OnRecognitionComplete(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[Worker-%d] Recognition complete | voice_id=%s", l.ID, resp.VoiceID)
}

func (l *ASRListener) OnFail(resp *asr.SpeechRecognitionResponse, err error) {
	if resp != nil {
		log.Printf("[Worker-%d] ERROR | voice_id=%s code=%d msg=%s err=%v",
			l.ID, resp.VoiceID, resp.Code, resp.Message, err)
	} else {
		log.Printf("[Worker-%d] ERROR | err=%v", l.ID, err)
	}
}

func main() {
	filePath := flag.String("f", "test.pcm", "path to PCM audio file (16kHz, 16bit, mono)")
	engine := flag.String("e", "16k_zh", "engine model type (16k_zh, 8k_zh, 16k_zh_en)")
	concurrency := flag.Int("c", 1, "concurrent recognition sessions")
	loopMode := flag.Bool("l", false, "enable loop mode")
	flag.Parse()

	appIDStr := os.Getenv(envAppID)
	sdkAppIDStr := os.Getenv(envSdkAppID)
	secretKey := os.Getenv(envSecretKey)

	if appIDStr == "" || sdkAppIDStr == "" || secretKey == "" {
		fmt.Fprintf(os.Stderr, `Error: Missing required environment variables.

Please set the following environment variables:

  export %s="your-tencent-cloud-appid"
  export %s="your-trtc-sdk-app-id"
  export %s="your-sdk-secret-key"

How to obtain:
  1. Get APPID from CAM Console: https://console.cloud.tencent.com/cam/capi
  2. Open TRTC Console:  https://console.cloud.tencent.com/trtc/app
  3. Create or select an application
  4. Copy SDKAppID and SDK secret key from the application overview page

`, envAppID, envSdkAppID, envSecretKey)
		os.Exit(1)
	}

	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		log.Fatalf("Invalid %s: %s (must be integer)", envAppID, appIDStr)
	}

	sdkAppID, err := strconv.Atoi(sdkAppIDStr)
	if err != nil {
		log.Fatalf("Invalid %s: %s (must be integer)", envSdkAppID, sdkAppIDStr)
	}

	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("Audio file not found: %s\nPlease provide a valid PCM audio file (16kHz, 16bit, mono).", *filePath)
	}

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				process(id, *filePath, appID, sdkAppID, secretKey, *engine)
				if !*loopMode {
					break
				}
				time.Sleep(time.Second)
			}
		}(i)
	}
	wg.Wait()
}

func process(id int, filePath string, appID, sdkAppID int, secretKey, engine string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[Worker-%d] Failed to open file: %v", id, err)
		return
	}
	defer file.Close()

	cred := common.NewCredential(appID, sdkAppID, secretKey)
	listener := &ASRListener{ID: id}
	recognizer := asr.NewSpeechRecognizer(cred, engine, listener)

	log.Printf("[Worker-%d] Starting recognition...", id)
	if err := recognizer.Start(); err != nil {
		log.Printf("[Worker-%d] Start failed: %v", id, err)
		return
	}

	buf := make([]byte, SliceSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[Worker-%d] Read error: %v", id, err)
			break
		}

		if err := recognizer.Write(buf[:n]); err != nil {
			log.Printf("[Worker-%d] Write error: %v", id, err)
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	if err := recognizer.Stop(); err != nil {
		log.Printf("[Worker-%d] Stop error: %v", id, err)
	}

	log.Printf("[Worker-%d] Done.", id)
}

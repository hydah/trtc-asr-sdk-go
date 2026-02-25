// Package main demonstrates how to use the TRTC-ASR Go SDK for real-time speech recognition.
//
// Usage:
//
//	go run main.go -f test.pcm
//	go run main.go -f test.pcm -e 16k_zh -c 1
//
// Prerequisites:
//  1. Get Tencent Cloud APPID: https://console.cloud.tencent.com/cam/capi
//  2. Create a TRTC application: https://console.cloud.tencent.com/trtc/app
//  3. Get SDKAppID and SDK secret key from the application overview page
//  4. Prepare a PCM audio file (16kHz, 16bit, mono)
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/trtc-asr/trtc-asr-sdk-go/asr"
	"github.com/trtc-asr/trtc-asr-sdk-go/common"
)

// ===== Configuration =====
// Fill in your credentials before running.
var (
	AppID     = 0  // Tencent Cloud APPID (https://console.cloud.tencent.com/cam/capi)
	SdkAppID  = 0  // TRTC application ID (e.g., 1400188366)
	SecretKey = "" // TRTC SDK secret key
)

// ===== Default Settings =====
var (
	EngineModelType = "16k_zh"
	SliceSize       = 6400 // bytes per audio chunk (200ms for 16kHz 16bit mono PCM)
)

// MySpeechRecognitionListener implements the SpeechRecognitionListener interface.
type MySpeechRecognitionListener struct {
	ID int
}

func (l *MySpeechRecognitionListener) OnRecognitionStart(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[%d] Recognition started, voice_id: %s", l.ID, resp.VoiceID)
}

func (l *MySpeechRecognitionListener) OnSentenceBegin(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[%d] Sentence begin, index: %d", l.ID, resp.Result.Index)
}

func (l *MySpeechRecognitionListener) OnRecognitionResultChange(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[%d] Result change, index: %d, text: %s",
		l.ID, resp.Result.Index, resp.Result.VoiceTextStr)
}

func (l *MySpeechRecognitionListener) OnSentenceEnd(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[%d] Sentence end, index: %d, text: %s",
		l.ID, resp.Result.Index, resp.Result.VoiceTextStr)
}

func (l *MySpeechRecognitionListener) OnRecognitionComplete(resp *asr.SpeechRecognitionResponse) {
	log.Printf("[%d] Recognition complete, voice_id: %s", l.ID, resp.VoiceID)
}

func (l *MySpeechRecognitionListener) OnFail(resp *asr.SpeechRecognitionResponse, err error) {
	if resp != nil {
		log.Printf("[%d] Recognition failed, voice_id: %s, error: %v", l.ID, resp.VoiceID, err)
	} else {
		log.Printf("[%d] Recognition failed, error: %v", l.ID, err)
	}
}

func main() {
	concurrency := flag.Int("c", 1, "number of concurrent recognition sessions")
	loop := flag.Bool("l", false, "loop mode for stress testing")
	filePath := flag.String("f", "test.pcm", "path to audio file (PCM format)")
	engine := flag.String("e", EngineModelType, "engine model type (16k_zh, 8k_zh, 16k_zh_en)")
	flag.Parse()

	EngineModelType = *engine

	if AppID == 0 || SdkAppID == 0 || SecretKey == "" {
		log.Fatal("Error: Please set AppID, SdkAppID and SecretKey in the code.\n\n" +
			"Steps:\n" +
			"  1. Get APPID from CAM Console: https://console.cloud.tencent.com/cam/capi\n" +
			"  2. Open TRTC Console: https://console.cloud.tencent.com/trtc/app\n" +
			"  3. Create or select an application\n" +
			"  4. Copy SDKAppID and SDK secret key from the application overview\n" +
			"  5. Fill in the credentials at the top of this file.\n")
	}

	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("Error: Audio file not found: %s\n\nPlease provide a valid PCM audio file (16kHz, 16bit, mono).", *filePath)
	}

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if *loop {
				for {
					processAudio(id, *filePath)
					time.Sleep(time.Second)
				}
			} else {
				processAudio(id, *filePath)
			}
		}(i)
	}
	wg.Wait()
}

func processAudio(id int, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[%d] Failed to open file: %v", id, err)
		return
	}
	defer file.Close()

	credential := common.NewCredential(AppID, SdkAppID, SecretKey)
	listener := &MySpeechRecognitionListener{ID: id}
	recognizer := asr.NewSpeechRecognizer(credential, EngineModelType, listener)

	if err := recognizer.Start(); err != nil {
		log.Printf("[%d] Failed to start recognizer: %v", id, err)
		return
	}

	buf := make([]byte, SliceSize)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[%d] Failed to read audio file: %v", id, err)
			break
		}

		if writeErr := recognizer.Write(buf[:n]); writeErr != nil {
			log.Printf("[%d] Failed to write audio data: %v", id, writeErr)
			break
		}

		time.Sleep(200 * time.Millisecond)
	}

	if err := recognizer.Stop(); err != nil {
		log.Printf("[%d] Failed to stop recognizer: %v", id, err)
	}

	fmt.Printf("[%d] Processing complete.\n", id)
}

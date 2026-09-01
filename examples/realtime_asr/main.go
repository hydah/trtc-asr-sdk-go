// Package main demonstrates how to use the TRTC-ASR Go SDK for real-time speech recognition.
//
// Usage:
//
//	go run main.go -f test.pcm
//	go run main.go -f test.pcm -e 16k_zh_en -c 1
//
// Prerequisites:
//  1. Get Tencent Cloud APPID: https://console.cloud.tencent.com/cam/capi
//  2. Create a TRTC application: https://console.cloud.tencent.com/trtc/app
//  3. Get SDKAppID and SDK secret key from the application overview page
//  4. Prepare a PCM audio file (16kHz, 16bit, mono)
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hydah/trtc-asr-sdk-go/asr"
	"github.com/hydah/trtc-asr-sdk-go/common"
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
	EngineModelType = "16k_zh_en"
	SliceSize       = 6400 // bytes per audio chunk (200ms for 16kHz 16bit mono PCM)
)

const (
	envAppID     = "TRTC_ASR_APP_ID"
	envSdkAppID  = "TRTC_ASR_SDK_APP_ID"
	envSecretKey = "TRTC_ASR_SECRET_KEY"
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
	log.Printf("[%d] Sentence end, index: %d, lang: %s, text: %s",
		l.ID, resp.Result.Index, resp.Result.Language, resp.Result.VoiceTextStr)
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
	filePath := flag.String("f", "../test.pcm", "path to audio file (PCM or WAV)")
	engine := flag.String("e", EngineModelType, "engine model type (16k_zh, 8k_zh, 16k_zh_en, bigmodel)")
	lang := flag.String("lang", "", "language hint for bigmodel engine (e.g. ms, zh, auto)")
	envFile := flag.String("env", "", "path to .env file to load credentials from (e.g. ../.env.test)")
	flag.Parse()

	EngineModelType = *engine
	loadCredentialsFromEnv()
	if *envFile != "" {
		loadEnvFile(*envFile)
	}

	if AppID == 0 || SdkAppID == 0 || SecretKey == "" {
		log.Fatal("Error: Please set AppID, SdkAppID and SecretKey in the code or via environment variables.\n\n" +
			"Steps:\n" +
			"  1. Get APPID from CAM Console: https://console.cloud.tencent.com/cam/capi\n" +
			"  2. Open TRTC Console: https://console.cloud.tencent.com/trtc/app\n" +
			"  3. Create or select an application\n" +
			"  4. Copy SDKAppID and SDK secret key from the application overview\n" +
			"  5. Fill in the credentials at the top of this file or export TRTC_ASR_APP_ID, TRTC_ASR_SDK_APP_ID and TRTC_ASR_SECRET_KEY.\n")
	}

	if _, err := os.Stat(*filePath); os.IsNotExist(err) {
		log.Fatalf("Error: Audio file not found: %s\n\nPlease provide a valid PCM/WAV audio file.", *filePath)
	}

	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if *loop {
				for {
					processAudio(id, *filePath, *lang)
					time.Sleep(time.Second)
				}
			} else {
				processAudio(id, *filePath, *lang)
			}
		}(i)
	}
	wg.Wait()
}

func loadCredentialsFromEnv() {
	if AppID == 0 {
		AppID = parseEnvInt(envAppID)
	}
	if SdkAppID == 0 {
		SdkAppID = parseEnvInt(envSdkAppID)
	}
	if SecretKey == "" {
		SecretKey = os.Getenv(envSecretKey)
	}
}

func parseEnvInt(name string) int {
	value := os.Getenv(name)
	if value == "" {
		return 0
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Error: %s must be an integer: %v", name, err)
	}
	return number
}

func processAudio(id int, filePath string, lang string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[%d] Failed to open file: %v", id, err)
		return
	}
	defer file.Close()

	// If the file is a WAV, skip the header and stream raw PCM.
	sliceSize := SliceSize
	dataReader := io.Reader(file)
	if sampleRate, isWAV, _ := skipWAVHeader(file); isWAV {
		log.Printf("[%d] WAV detected (sample_rate=%d), header skipped", id, sampleRate)
		dataReader = file
		// 200ms slice = sampleRate * 2 bytes(16bit) * 0.2s
		sliceSize = sampleRate * 2 * 200 / 1000
		if sliceSize <= 0 {
			sliceSize = SliceSize
		}
	}

	credential := common.NewCredential(AppID, SdkAppID, SecretKey)
	listener := &MySpeechRecognitionListener{ID: id}
	recognizer := asr.NewSpeechRecognizer(credential, EngineModelType, listener)
	if lang != "" {
		recognizer.SetLanguage(lang)
	}

	if err := recognizer.Start(); err != nil {
		log.Printf("[%d] Failed to start recognizer: %v", id, err)
		return
	}

	buf := make([]byte, sliceSize)
	for {
		n, err := dataReader.Read(buf)
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

// skipWAVHeader reads the RIFF/WAVE header and positions the file cursor at the
// start of the PCM data chunk. Returns (sampleRate, true, nil) for a valid WAV.
func skipWAVHeader(f *os.File) (int, bool, error) {
	// RIFF header: "RIFF" + 4-byte size + "WAVE" = 12 bytes
	header := make([]byte, 12)
	if _, err := io.ReadFull(f, header); err != nil {
		return 0, false, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		// Not a WAV — rewind so the caller can read raw PCM from the start.
		_, _ = f.Seek(0, io.SeekStart)
		return 0, false, nil
	}

	sampleRate := 16000
	// Walk chunks until we find "data".
	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(f, chunkHeader); err != nil {
			return 0, false, err
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if chunkID == "fmt " {
			// fmt chunk: audioFormat(2) + numChannels(2) + sampleRate(4) + ...
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(f, fmtData); err != nil {
				return 0, false, err
			}
			if len(fmtData) >= 8 {
				sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			}
			continue
		}

		if chunkID == "data" {
			// File cursor now points at PCM samples.
			return sampleRate, true, nil
		}
		// Skip this chunk (chunkSize bytes, padded to even).
		skip := int64(chunkSize)
		if chunkSize%2 == 1 {
			skip++
		}
		if _, err := f.Seek(skip, io.SeekCurrent); err != nil {
			return 0, false, err
		}
	}
}

// loadEnvFile reads a simple KEY = VALUE / KEY = "VALUE" file and populates
// AppID, SdkAppID, SecretKey when the corresponding env vars are not already set.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error: cannot open env file %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "Key = Value" or "Key = \"Value\""
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"")

		switch strings.ToLower(key) {
		case "appid":
			if AppID == 0 {
				AppID = parseStrInt(val)
			}
		case "sdkappid":
			if SdkAppID == 0 {
				SdkAppID = parseStrInt(val)
			}
		case "secretkey":
			if SecretKey == "" {
				SecretKey = val
			}
		}
	}
}

func parseStrInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		log.Fatalf("Error: invalid integer %q: %v", s, err)
	}
	return n
}

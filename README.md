# TRTC-ASR Go SDK

基于 TRTC 鉴权体系的实时语音识别（ASR）Go SDK，通过 WebSocket 协议与 ASR 服务通信。

> 其他语言 SDK：[Python](https://github.com/hydah/trtc-asr-sdk-python) | [Node.js](https://github.com/hydah/trtc-asr-sdk-nodejs)

## 前提条件

使用本 SDK 前，您需要：

1. **获取腾讯云 APPID** — 在 [CAM API 密钥管理](https://console.cloud.tencent.com/cam/capi) 页面查看
2. **创建 TRTC 应用** — 在 [实时音视频控制台](https://console.cloud.tencent.com/trtc/app) 创建应用，获取 `SDKAppID`
3. **获取 SDK 密钥** — 在应用概览页点击「SDK密钥」查看密钥，即用于计算 UserSig 的加密密钥

## 协议说明

### WebSocket 连接

- **连接地址**：`wss://asr.cloud-rtc.com/asr/v2/<appid>?{请求参数}`

其中 `<appid>` 为腾讯云账号的 APPID，可通过 [API 密钥管理页面](https://console.cloud.tencent.com/cam/capi) 获取。

### 鉴权方式

在 WebSocket Header 中携带以下字段：

| Header | 说明 |
|--------|------|
| `X-TRTC-SdkAppId` | TRTC 应用 ID，从 [TRTC 控制台](https://console.cloud.tencent.com/trtc/app) 获取 |
| `X-TRTC-UserSig` | TRTC 签名，[计算文档](https://cloud.tencent.com/document/product/647/17275)，UserID 等于 URL 参数中的 `voice_id` |

### 请求参数

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `secretid` | 是 | String | SDK 内部自动用 APPID 填充 |
| `timestamp` | 是 | Integer | 当前 UNIX 时间戳（秒） |
| `expired` | 是 | Integer | 签名有效期截止时间戳，必须大于 timestamp |
| `nonce` | 是 | Integer | 随机正整数，最长10位 |
| `engine_model_type` | 是 | String | 引擎类型：`8k_zh`(中文电话)、`16k_zh`(中文通用)、`16k_zh_en`(中英文) |
| `voice_id` | 是 | String | 音频流全局唯一标识（推荐 UUID），最长128位 |
| `voice_format` | 否 | Integer | 语音编码：`1` PCM（默认） |
| `needvad` | 否 | Integer | `0` 关闭 VAD，`1` 开启（默认） |
| `hotword_id` | 否 | String | 热词表 ID |
| `customization_id` | 否 | String | 自学习模型 ID |
| `filter_dirty` | 否 | Integer | 过滤脏词：`0` 不过滤，`1` 过滤，`2` 替换为 * |
| `filter_modal` | 否 | Integer | 过滤语气词：`0` 不过滤，`1` 部分，`2` 严格 |
| `filter_punc` | 否 | Integer | 过滤句末句号：`0` 不过滤，`1` 过滤 |
| `convert_num_mode` | 否 | Integer | 数字转换：`0` 不转，`1` 智能转换（默认），`3` 数学转换 |
| `word_info` | 否 | Int | 显示词级时间：`0` 不显示，`1` 显示 |
| `vad_silence_time` | 否 | Integer | 静音断句阈值（ms），范围 240-1000，默认 1000 |
| `max_speak_time` | 否 | Integer | 强制断句时间（ms），范围 5000-90000，默认 60000 |
| `signature` | 是 | String | 接口签名参数，值与 X-TRTC-UserSig 一致 |

---

## 安装

```bash
go get github.com/trtc-asr/trtc-asr-sdk-go@latest
```

**要求**：Go 1.21+

## 快速开始

```go
package main

import (
    "io"
    "log"
    "os"
    "time"

    "github.com/trtc-asr/trtc-asr-sdk-go/asr"
    "github.com/trtc-asr/trtc-asr-sdk-go/common"
)

// 实现回调接口
type MyListener struct{}

func (l *MyListener) OnRecognitionStart(resp *asr.SpeechRecognitionResponse) {
    log.Printf("Recognition started, voice_id: %s", resp.VoiceID)
}
func (l *MyListener) OnSentenceBegin(resp *asr.SpeechRecognitionResponse) {
    log.Printf("Sentence begin, index: %d", resp.Result.Index)
}
func (l *MyListener) OnRecognitionResultChange(resp *asr.SpeechRecognitionResponse) {
    log.Printf("Result: %s", resp.Result.VoiceTextStr)
}
func (l *MyListener) OnSentenceEnd(resp *asr.SpeechRecognitionResponse) {
    log.Printf("Sentence end: %s", resp.Result.VoiceTextStr)
}
func (l *MyListener) OnRecognitionComplete(resp *asr.SpeechRecognitionResponse) {
    log.Printf("Complete, voice_id: %s", resp.VoiceID)
}
func (l *MyListener) OnFail(resp *asr.SpeechRecognitionResponse, err error) {
    log.Printf("Failed: %v", err)
}

func main() {
    // 1. 创建凭证
    credential := common.NewCredential(
        1300403317,                                              // 腾讯云 APPID
        1400188366,                                              // TRTC SDKAppID
        "5bd2850fff3ecb11d7c805251c51ee463a25727bddc2385f3fa8b", // SDK密钥
    )

    // 2. 创建识别器
    recognizer := asr.NewSpeechRecognizer(credential, "16k_zh", &MyListener{})

    // 3. 可选配置
    // recognizer.SetHotwordID("hotword-id")     // 设置热词
    // recognizer.SetVadSilenceTime(500)          // VAD 静音时间

    // 4. 启动识别
    if err := recognizer.Start(); err != nil {
        log.Fatal(err)
    }

    // 5. 发送音频数据
    file, _ := os.Open("audio.pcm")
    defer file.Close()

    buf := make([]byte, 6400) // 200ms of 16kHz 16bit mono PCM
    for {
        n, err := file.Read(buf)
        if err == io.EOF {
            break
        }
        recognizer.Write(buf[:n])
        time.Sleep(200 * time.Millisecond) // 模拟实时
    }

    // 6. 停止识别
    recognizer.Stop()
}
```

## 凭证获取

| 参数 | 来源 | 说明 |
|------|------|------|
| `AppID` | [CAM 密钥管理](https://console.cloud.tencent.com/cam/capi) | 腾讯云账号 APPID，用于 URL 路径 |
| `SDKAppID` | [TRTC 控制台](https://console.cloud.tencent.com/trtc/app) > 应用管理 | TRTC 应用 ID |
| `SecretKey` | [TRTC 控制台](https://console.cloud.tencent.com/trtc/app) > 应用概览 > SDK密钥 | 用于生成 UserSig，不会传输到网络 |

## 配置项

| 方法 | 说明 | 默认值 |
|------|------|--------|
| `SetVoiceFormat(f)` | 音频格式 | 1 (PCM) |
| `SetNeedVad(v)` | 是否开启 VAD | 1 (开启) |
| `SetConvertNumMode(m)` | 数字转换模式 | 1 (智能) |
| `SetHotwordID(id)` | 热词表 ID | - |
| `SetCustomizationID(id)` | 自学习模型 ID | - |
| `SetFilterDirty(m)` | 脏词过滤 | 0 (关闭) |
| `SetFilterModal(m)` | 语气词过滤 | 0 (关闭) |
| `SetFilterPunc(m)` | 句号过滤 | 0 (关闭) |
| `SetWordInfo(m)` | 词级时间 | 0 (关闭) |
| `SetVadSilenceTime(ms)` | VAD 静音阈值 | 1000ms |
| `SetMaxSpeakTime(ms)` | 强制断句时间 | 60000ms |
| `SetVoiceID(id)` | 自定义 voice_id | 自动 UUID |

## 引擎模型

| 类型 | 说明 |
|------|------|
| `8k_zh` | 中文通用，常用于电话场景 |
| `16k_zh` | 中文通用（推荐） |
| `16k_zh_en` | 中英文通用 |

## 示例

完整示例请参见 [`examples/`](./examples/) 目录：

- [`realtime_asr/`](./examples/realtime_asr/) — 硬编码凭证的基础示例
- [`realtime_asr_env/`](./examples/realtime_asr_env/) — 使用环境变量的推荐示例

运行示例：

```bash
# 使用环境变量（推荐）
export TRTC_APP_ID="1300403317"
export TRTC_SDK_APP_ID="1400188366"
export TRTC_SECRET_KEY="your-sdk-secret-key"

cd examples/realtime_asr_env
go run main.go -f ../../test.pcm

# 查看所有选项
go run main.go -h
```

## 项目结构

```
trtc-asr-sdk-go/
├── common/                     # 公共模块
│   ├── credential.go           # 凭证管理（APPID + SDKAppID + SDK密钥）
│   ├── usersig.go              # TRTC UserSig 生成
│   ├── usersig_test.go         # UserSig 单元测试
│   ├── signature.go            # URL 请求参数构建
│   ├── signature_test.go       # 参数构建单元测试
│   └── errors.go               # 错误定义
├── asr/                        # ASR 语音识别模块
│   └── speech_recognizer.go    # 实时语音识别器
├── examples/                   # 示例代码
│   ├── .env.example            # 环境变量模板
│   ├── realtime_asr/           # 基础示例
│   │   └── main.go
│   └── realtime_asr_env/       # 环境变量示例（推荐）
│       └── main.go
├── go.mod                      # Go module 配置
├── go.sum                      # 依赖校验
├── .gitignore
└── README.md
```

## 常见问题

### APPID 和 SDKAppID 有什么区别？

- **APPID**（如 `1300403317`）：腾讯云账号级别的 ID，从 [CAM 密钥管理](https://console.cloud.tencent.com/cam/capi) 获取，用于 WebSocket URL 路径
- **SDKAppID**（如 `1400188366`）：TRTC 应用级别的 ID，从 [TRTC 控制台](https://console.cloud.tencent.com/trtc/app) 获取，用于 Header 鉴权

### UserSig 是什么？

UserSig 是基于 SDKAppID 和 SDK 密钥计算的签名，用于 TRTC 服务鉴权。SDK 会自动生成，无需手动计算。详见[鉴权文档](https://cloud.tencent.com/document/product/647/17275)。

### signature 参数怎么计算？

根据协议，`signature` 的值与 `X-TRTC-UserSig` 一致，SDK 内部自动处理，用户无需关心。

### 支持哪些音频格式？

当前支持 PCM 格式（`voice_format=1`），建议使用 16kHz、16bit、单声道的 PCM 音频。

## License

MIT License

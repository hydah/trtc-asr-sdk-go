# TRTC-ASR Go SDK

基于 TRTC 鉴权体系的语音识别（ASR）Go SDK，支持实时语音识别（WebSocket）、一句话识别（HTTP）和录音文件识别（异步 HTTP）三种模式。

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

### 一句话识别接口

- **请求地址**：`https://asr.cloud-rtc.com/v1/SentenceRecognition?{请求参数}`
- **请求方法**：HTTP POST，Content-Type 为 `application/json; charset=utf-8`

#### URL 请求参数

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `AppId` | 是 | String | 腾讯云 APPID |
| `Secretid` | 是 | String | SDK 内部自动用 APPID 填充 |
| `RequestId` | 是 | String | 全局请求唯一 ID（UUID），用于生成 UserSig |
| `Timestamp` | 是 | Integer | 当前 UNIX 时间戳（秒） |

#### 请求体参数（JSON）

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `EngSerViceType` | 是 | String | 引擎类型：`16k_zh`(中文)、`16k_zh_en`(中英文) |
| `SourceType` | 是 | Integer | `0` URL 上传、`1` 本地数据（base64） |
| `VoiceFormat` | 是 | String | 音频格式：`wav`、`pcm`、`ogg-opus`、`mp3`、`m4a` |
| `Data` | 条件 | String | base64 编码的音频数据（SourceType=1 时必填） |
| `DataLen` | 条件 | Integer | 音频数据原始长度（SourceType=1 时必填） |
| `Url` | 条件 | String | 音频 URL（SourceType=0 时必填） |
| `WordInfo` | 否 | Integer | 词级时间：`0` 不显示、`1` 显示、`2` 含标点 |
| `FilterDirty` | 否 | Integer | 脏词过滤：`0` 不过滤、`1` 过滤、`2` 替换 |
| `FilterModal` | 否 | Integer | 语气词过滤：`0` 不过滤、`1` 部分、`2` 严格 |
| `FilterPunc` | 否 | Integer | 标点过滤：`0` 不过滤、`2` 过滤全部 |
| `ConvertNumMode` | 否 | Integer | 数字转换：`0` 不转、`1` 智能转换（默认） |
| `HotwordId` | 否 | String | 热词表 ID |
| `HotwordList` | 否 | String | 临时热词列表 |
| `InputSampleRate` | 否 | Integer | PCM 输入采样率（仅 PCM 格式，支持 8000） |

**限制**：音频时长 ≤ 60s，文件大小 ≤ 3MB，单账号并发 ≤ 30次/秒

### 录音文件识别接口

录音文件识别是异步接口，适用于较长音频（≤12h）。工作流程为：提交任务 → 轮询结果。

#### 创建任务：CreateRecTask

- **请求地址**：`https://asr.cloud-rtc.com/v1/CreateRecTask?{请求参数}`
- **请求方法**：HTTP POST，Content-Type 为 `application/json; charset=utf-8`
- **并发限制**：默认 20次/秒

URL 请求参数与一句话识别相同（AppId、Secretid、RequestId、Timestamp）。

##### 请求体参数（JSON）

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `EngineModelType` | 是 | String | 引擎类型：`16k_zh`(中文)、`16k_zh_en`(中英文) |
| `ChannelNum` | 是 | Integer | 声道数，目前仅支持 `1` |
| `ResTextFormat` | 是 | Integer | 结果格式：`0` 基础、`1` 含词级时间、`2` 含标点时间 |
| `SourceType` | 是 | Integer | `0` URL 上传、`1` 本地数据（base64） |
| `Url` | 条件 | String | 音频 URL（SourceType=0，时长≤12h，大小≤1GB） |
| `Data` | 条件 | String | base64 编码音频数据（SourceType=1，大小≤5MB） |
| `DataLen` | 条件 | Integer | 音频数据原始长度（SourceType=1） |
| `CallbackUrl` | 否 | String | 回调 URL，任务完成后 POST 结果 |
| `FilterDirty` | 否 | Integer | 脏词过滤 |
| `FilterModal` | 否 | Integer | 语气词过滤 |
| `FilterPunc` | 否 | Integer | 标点过滤 |
| `ConvertNumMode` | 否 | Integer | 数字转换 |
| `HotwordId` | 否 | String | 热词表 ID |
| `HotwordList` | 否 | String | 临时热词列表 |

##### 响应

返回 `RecTaskId`（任务 ID），用于后续查询。任务有效期 24 小时。

#### 查询结果：DescribeTaskStatus

- **请求地址**：`https://asr.cloud-rtc.com/v1/DescribeTaskStatus?{请求参数}`
- **请求方法**：HTTP POST
- **并发限制**：默认 50次/秒

##### 请求体参数（JSON）

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `RecTaskId` | 是 | String | CreateRecTask 返回的任务 ID |

##### 响应（TaskStatus）

| 字段 | 类型 | 说明 |
|------|------|------|
| `RecTaskId` | String | 任务 ID |
| `Status` | Integer | `0` 等待、`1` 执行中、`2` 成功、`3` 失败 |
| `StatusStr` | String | waiting / doing / success / failed |
| `Result` | String | 识别结果文本 |
| `ErrorMsg` | String | 失败原因 |
| `ResultDetail` | Array | 句级详细结果（含词级时间偏移） |
| `AudioDuration` | Float | 音频时长（秒） |

---

## 安装

```bash
go get github.com/hydah/trtc-asr-sdk-go@latest
```

**要求**：Go 1.21+

## 快速开始

### 实时语音识别

```go
package main

import (
    "io"
    "log"
    "os"
    "time"

    "github.com/hydah/trtc-asr-sdk-go/asr"
    "github.com/hydah/trtc-asr-sdk-go/common"
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
        0,                       // 腾讯云 APPID
        0,                       // TRTC SDKAppID
        "your-sdk-secret-key",   // SDK密钥
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

### 一句话识别

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/hydah/trtc-asr-sdk-go/asr"
    "github.com/hydah/trtc-asr-sdk-go/common"
)

func main() {
    // 1. 创建凭证
    credential := common.NewCredential(
        0,                       // 腾讯云 APPID
        0,                       // TRTC SDKAppID
        "your-sdk-secret-key",   // SDK密钥
    )

    // 2. 创建一句话识别器
    recognizer := asr.NewSentenceRecognizer(credential)

    // 3. 从本地文件识别（自动 base64 编码）
    data, _ := os.ReadFile("audio.pcm")
    result, err := recognizer.RecognizeData(data, "pcm", "16k_zh_en")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("识别结果: %s\n", result.Result)
    fmt.Printf("音频时长: %d ms\n", result.AudioDuration)

    // 或者从 URL 识别
    // result, err := recognizer.RecognizeURL("https://example.com/audio.wav", "wav", "16k_zh_en")
}
```

### 录音文件识别

```go
package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/hydah/trtc-asr-sdk-go/asr"
    "github.com/hydah/trtc-asr-sdk-go/common"
)

func main() {
    // 1. 创建凭证
    credential := common.NewCredential(
        0,                       // 腾讯云 APPID
        0,                       // TRTC SDKAppID
        "your-sdk-secret-key",   // SDK密钥
    )

    // 2. 创建录音文件识别器
    recognizer := asr.NewFileRecognizer(credential)

    // 3. 提交识别任务（本地文件）
    data, _ := os.ReadFile("audio.pcm")
    taskID, err := recognizer.CreateTaskFromData(data, "pcm", "16k_zh_en")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("任务已提交: %s\n", taskID)

    // 4. 轮询等待结果（默认 1 秒间隔，10 分钟超时）
    status, err := recognizer.WaitForResult(taskID)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("识别结果: %s\n", status.Result)
    fmt.Printf("音频时长: %.2f s\n", status.AudioDuration)

    // 或者从 URL 提交（支持更大文件，≤1GB / ≤12h）
    // taskID, err := recognizer.CreateTaskFromURL("https://example.com/audio.wav", "16k_zh_en")

    // 或者自定义轮询间隔
    // status, err := recognizer.WaitForResultWithInterval(taskID, 2*time.Second, 30*time.Minute)
    _ = time.Second // suppress unused import
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

完整示例请参见：

- **实时语音识别**：[`examples/realtime_asr/`](./examples/realtime_asr/) — WebSocket 流式识别
- **一句话识别**：[`examples/sentence_asr/`](./examples/sentence_asr/) — HTTP 短音频识别（≤60s）
- **录音文件识别**：[`examples/file_asr/`](./examples/file_asr/) — 异步长音频识别

运行示例：

```bash
# 实时语音识别
cd examples/realtime_asr
go run main.go -f ../test.pcm

# 一句话识别
cd examples/sentence_asr
go run main.go -f ../test.pcm -fmt pcm

# 录音文件识别
cd examples/file_asr
go run main.go -f ../test.pcm

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
│   ├── speech_recognizer.go    # 实时语音识别器（WebSocket）
│   ├── speech_recognizer_test.go # 生命周期与并发健壮性测试
│   ├── sentence_recognizer.go  # 一句话识别器（HTTP）
│   ├── sentence_recognizer_test.go # 一句话识别单元测试
│   ├── file_recognizer.go      # 录音文件识别器（异步 HTTP）
│   └── file_recognizer_test.go # 录音文件识别单元测试
├── examples/                   # 示例代码
│   ├── test.pcm                # 测试音频文件（16kHz 16bit 单声道 PCM）
│   ├── realtime_asr/           # 实时语音识别示例
│   │   └── main.go
│   ├── sentence_asr/           # 一句话识别示例
│   │   └── main.go
│   └── file_asr/               # 录音文件识别示例
│       └── main.go
├── go.mod                      # Go module 配置
├── go.sum                      # 依赖校验
├── .gitignore
└── README.md
```

## 常见问题

### APPID 和 SDKAppID 有什么区别？

- **APPID**（如 `13xxxxxxxx`）：腾讯云账号级别的 ID，从 [CAM 密钥管理](https://console.cloud.tencent.com/cam/capi) 获取，用于 WebSocket URL 路径
- **SDKAppID**（如 `14xxxxxxxx`）：TRTC 应用级别的 ID，从 [TRTC 控制台](https://console.cloud.tencent.com/trtc/app) 获取，用于 Header 鉴权

### UserSig 是什么？

UserSig 是基于 SDKAppID 和 SDK 密钥计算的签名，用于 TRTC 服务鉴权。SDK 会自动生成，无需手动计算。详见[鉴权文档](https://cloud.tencent.com/document/product/647/17275)。

### signature 参数怎么计算？

根据协议，`signature` 的值与 `X-TRTC-UserSig` 一致，SDK 内部自动处理，用户无需关心。

### 支持哪些音频格式？

- **实时语音识别**：支持 PCM 格式（`voice_format=1`），建议 16kHz、16bit、单声道
- **一句话识别**：支持 wav、pcm、ogg-opus、mp3、m4a，音频时长 ≤ 60s，文件 ≤ 3MB
- **录音文件识别**：支持 wav、ogg-opus、mp3、m4a，本地文件 ≤ 5MB，URL ≤ 1GB / ≤ 12h

## License

MIT License

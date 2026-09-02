# 更新日志

本文件记录 TRTC-ASR Go SDK 的所有重要变更。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循[语义化版本](https://semver.org/lang/zh-CN/)。

## [未发布]

首个正式版本尚未打 tag。发布时把本节标题改为 `## [0.1.0] - YYYY-MM-DD`，
并同步更新 `common/sdkinfo.go` 中的 `SDKVersion`。

### 新增

- Credential 可通过 `SetSite` 选择国内站（默认，`asr.cloud-rtc.com`）或国际站（`asr-intl.cloud-rtc.com`），三个识别器共用
- 实时语音识别（WebSocket），支持流式写入与优雅停止
- 一句话识别（HTTP）
- 录音文件识别（异步 HTTP，CreateRecTask + DescribeTaskStatus）
- 说话人分离：匿名聚类与声纹角色认证两种模式
- VAD 调优、热词、自定义语言模型、脏词/语气词/标点过滤等识别参数
- 所有请求上报 SDK 自身标识（`platform` / `sdk_lang` / `sdk_type` / `version`），
  便于服务端按语言、版本、平台定位客户问题
- MIT LICENSE
- GitHub Actions CI：Go 1.21/1.22/1.23 × Linux/macOS，含 `go vet`、gofmt 检查与
  `-race` 测试

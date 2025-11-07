# MicroVibe-Go 技术文档

> MicroVibe-Go 是一个基于 AI 推荐算法的多端短视频平台后端系统，对标抖音的核心功能。

## 📚 文档导航

### 🏗️ 架构设计 (Architecture)

- **[系统架构](architecture/system-architecture.md)** - 整体架构设计、技术选型和模块划分
- **[推荐算法](architecture/recommendation-algorithm.md)** - 自研推荐算法引擎（召回、排序、过滤）
- **[事件驱动架构](architecture/event-driven-architecture.md)** - 事件总线、事件处理器和异步架构
- **[流媒体架构](architecture/streaming-architecture.md)** - WebRTC + SFU 混合架构设计

### 🔌 系统集成 (Integration)

- **[Ion SFU 集成](integration/ion-sfu.md)** - Ion SFU 服务器集成和 WebRTC 信令
- **[Ion SFU 部署](integration/ion-sfu-deployment.md)** - SFU 服务器部署和配置指南
- **[Ion SDK 集成](integration/ion-sdk.md)** - Go SDK 客户端集成
- **[Authentik SSO](integration/authentik-sso.md)** - 单点登录和 OAuth 2.0 集成

### 💻 开发指南 (Development)

- **[快速开始](development/quick-start.md)** - 项目启动和环境配置
- **[Pre-commit 使用指南](development/pre-commit-guide.md)** - Git 提交规范和代码质量检查
- **[缓存框架](development/cache-framework.md)** - 多级缓存系统使用指南（类似 Spring Cache）
- **[错误处理](development/error-handling.md)** - 统一错误处理机制
- **[直播开发指南](development/live-streaming-guide.md)** - 直播功能开发完整指南
- **[OBS 推流指南](development/obs-streaming.md)** - OBS Studio 推流配置
- **[SFU 快速开始](development/sfu-quickstart.md)** - WebRTC 直播快速上手

### 📦 归档文档 (Archived)

历史版本和临时文档，仅供参考：
- [归档目录](archived/) - 已过时或被替代的文档

## 🚀 快速导航

### 新手入门
1. 阅读 [README.md](../README.md) - 项目概述
2. 阅读 [CLAUDE.md](../CLAUDE.md) - Claude Code 开发指南
3. 阅读 [快速开始](development/quick-start.md) - 环境配置
4. 查看 [PROGRESS.md](../PROGRESS.md) - 功能进度

### 核心功能开发
- **推荐系统**: [推荐算法](architecture/recommendation-algorithm.md)
- **缓存系统**: [缓存框架](development/cache-framework.md)
- **直播系统**: [直播开发指南](development/live-streaming-guide.md) + [Ion SFU 集成](integration/ion-sfu.md)
- **事件系统**: [事件驱动架构](architecture/event-driven-architecture.md)

### 部署运维
- [Ion SFU 部署](integration/ion-sfu-deployment.md) - 流媒体服务器部署
- [Authentik SSO](integration/authentik-sso.md) - 单点登录部署

## 📖 文档约定

### 命名规范
- 文件名使用小写字母和连字符（kebab-case）：`system-architecture.md`
- 目录名使用小写字母：`architecture/`, `integration/`, `development/`
- 避免使用大写、下划线或空格

### 文档结构
```
docs/
├── README.md                    # 本文件 - 文档导航中心
├── architecture/                # 架构设计文档
│   ├── system-architecture.md
│   ├── recommendation-algorithm.md
│   ├── event-driven-architecture.md
│   └── streaming-architecture.md
├── integration/                 # 第三方集成文档
│   ├── ion-sfu.md
│   ├── ion-sfu-deployment.md
│   ├── ion-sdk.md
│   └── authentik-sso.md
├── development/                 # 开发指南文档
│   ├── quick-start.md
│   ├── pre-commit-guide.md
│   ├── cache-framework.md
│   ├── error-handling.md
│   ├── live-streaming-guide.md
│   ├── obs-streaming.md
│   └── sfu-quickstart.md
└── archived/                    # 归档文档（已过时）
    └── ...
```

### 文档维护
- **更新频率**: 每次重大功能开发完成后更新相关文档
- **版本控制**: 通过 Git 管理文档版本
- **过时文档**: 移动到 `archived/` 目录，不直接删除
- **文档审查**: 每月审查一次文档准确性

## 🔗 相关链接

- **项目首页**: [README.md](../README.md)
- **开发指南**: [CLAUDE.md](../CLAUDE.md)
- **功能进度**: [PROGRESS.md](../PROGRESS.md)
- **示例代码**: [examples/](../examples/)
- **API 文档**: 启动服务器后访问 `/swagger/index.html`

## 📝 文档贡献

欢迎贡献文档！请遵循以下原则：

1. **清晰性**: 使用简洁明了的语言
2. **完整性**: 包含必要的示例代码和配置
3. **准确性**: 确保内容与代码实现一致
4. **时效性**: 及时更新过时内容

提交文档时请确保：
- 使用 Markdown 格式
- 代码块指定语言（```go、```bash 等）
- 添加目录索引（对于长文档）
- 更新本导航文件的链接

---

最后更新: 2025-11-04

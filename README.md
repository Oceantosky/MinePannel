# CloudSync 开源套装 (Server & Client)

欢迎使用 CloudSync 项目！这是一套由 **CloudSync Server**（独立服务端）和 **CloudSync Client**（基于 PCL CE 的二次开发客户端）组成的完整的 Minecraft 游戏实例云端高速同步解决方案。

为了保证本套装能够健康、合法、安全地在整个 Minecraft 开源社区中传播，请在二次开发和分发时遵守以下开源协议：

## 🔒 整体开源协议与版权声明

本项目由于包含自主研发的服务端架构以及对第三方开源启动器的修改，采用了**混合开源许可协议（Dual License）**。

### 1. 服务端 (Server 目录)
Server 端完全由 CloudSync 独立开发（包括 Go 后端程序、Vue 管理面板等）。
**开源协议**：[Apache License 2.0](Server/LICENSE)

- 您可以自由地使用、修改、商用或分发 Server 端的代码。
- 若您修改了代码，请在源代码文件中加入显著的修改声明。
- 任何专利诉讼将导致授权终止。

### 2. 客户端 (Client 目录)
Client 端基于开源启动器 [PCL CE](https://github.com/PCL-Community/PCL-CE) 和官方 PCL 进行了二次创作。为了尊重原作者（龙腾猫跃）及 PCL CE 社区的劳动成果并满足授权条款（“重度使用”限制），Client 端继续继承原有的混合授权协议：

- **核心界面与主模块** (`Client/Plain Craft Launcher 2/` 目录)：
  强制遵守原作者的 **[自定义授权指南](Client/Plain%20Craft%20Launcher%202/LICENCE)**。
  **注意**：基于此部分进行再创作时，您**必须**继续保留原作者的赞助链接、名称前缀要求，并禁止破解任何原本设定的高级解锁功能。

- **底层组件与新增同步代码** (如 `Client/PCL.Core` 以及所有由 CloudSync 编写的逻辑)：
  遵守 **[Apache License 2.0](Client/LICENSE)**。我们（CloudSync Contributors）拥有这部分修改代码的附加版权声明。

## 🚀 立即开始
如果您想部署您的云端同步网络，请分别查看对应目录的使用指南。

- **服务端**：前往 `Server/` 目录。我们提供了纯净的 Go 后端代码、Vue 面板源码，您可以根据需要自由编译、运行并在云服务器中部署。
- **客户端**：前往 `Client/` 目录。我们在此为您提供了整合 CloudSync 协议的 PCL 客户端分支。您可以采用 `.NET 8` SDK 直接编译生成独立的发布版本程序。

---
*Copyright (c) 2026 CloudSync Contributors. All Rights Reserved where applicable.*
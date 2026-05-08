# 服装 AI 打版工具实施计划

> 当前项目决策：AI 层运行时从原始文档指定的 Python 3.10.13 调整为 Python 3.11，后续依赖版本与镜像配置按 Python 3.11 兼容性适配。除该项外，其余架构边界和业务约束保持不变。

## 1. 项目目标

基于架构文档要求，分阶段实现一个可运行、可部署、可逐步替换为真实 AI 与工业版型能力的服装 AI 打版系统。整个实施过程中必须满足以下原则：

- 严格遵守文档指定技术栈、接口路径、字段命名和数据格式。
- AI 只负责结构识别和参数预填，绝不生成或预测绝对尺寸。
- 3D 预览必须基于 GarmentCode 原生输出和 Google ModelViewer。
- 工业版型输出必须遵循 GarmentCode -> 双向转换器 -> Seamly2D -> DXF 的双引擎链路。
- 所有模块从第一阶段开始就保持可配置、可容器化、可持续集成。

## 2. 实施原则

1. 先验证高风险依赖，再建设完整工程。
2. 先固定接口契约，再填充具体实现。
3. 先交付可运行 MVP，再逐步接入真实模型与工业能力。
4. 所有阶段都保持接口兼容，避免后续大规模重构。
5. 所有配置必须外置，支持环境变量注入。

## 3. 风险优先级

以下依赖和链路是整个项目的主要风险点，应优先验证：

1. pygarment 0.3.0 是否可稳定生成 2D/3D 输出。
2. trimesh 是否可稳定完成 OBJ 到 GLB 的转换。
3. seamly2d-cli 2024.1.0 是否支持文档要求的无头命令参数。
4. YOLOv8-seg、YOLOv8-pose、MediaPipe、CLIP、Milvus 的组合链路是否可运行。
5. Electron 29.0.0 与 React 18.2.0 的桌面端集成是否稳定。
6. MinIO、Redis、PostgreSQL、Milvus 在 docker-compose 下的协同是否顺畅。

## 4. 分阶段实施计划

### 阶段 0：关键依赖验证

目标：确认所有高风险第三方组件在当前环境中的最小可行性。

交付内容：

- pygarment 最小示例脚本。
- OBJ 导出与 GLB 转换验证脚本。
- seamly2d-cli 最小 DXF 生成验证脚本。
- MinIO 上传下载验证脚本。
- 通义千问 V3 API 连通性验证脚本。
- YOLO / MediaPipe / CLIP / Milvus 最小调用验证脚本。

验收标准：

- 每项关键依赖至少有一个可独立运行的验证脚本。
- 所有验证结果写入 docs 目录，明确成功条件、失败原因和替代处理方案。

### 阶段 1：工程骨架与基础设施

目标：建立完整仓库结构、依赖管理、基础配置和本地编排环境。

目录结构：

```text
garment-ai/
├── frontend/
├── backend/
├── ai/
├── engine/
├── deploy/
├── scripts/
└── docs/
```

交付内容：

- 前端 package.json、TypeScript 配置、Electron 主进程基础结构。
- 后端 Go workspace / go.mod、服务拆分目录、基础配置加载。
- AI 服务 requirements.txt、FastAPI 应用骨架、配置加载模块。
- 引擎服务 requirements.txt、pygarment 与转换器骨架。
- docker-compose.yml。
- 各服务 Dockerfile。
- README 初稿。
- .env.example。
- GitHub Actions 初版流水线。

验收标准：

- docker-compose 可以启动 PostgreSQL、Redis、MinIO、Milvus 和各应用空服务。
- 所有模块具备健康检查接口。

### 阶段 2：接口契约冻结

目标：在不依赖完整业务实现的前提下先固定前后端与服务间契约。

交付内容：

- REST API 路由定义。
- 统一响应结构。
- 错误码定义。
- Protobuf 文件。
- gRPC 客户端与服务端桩代码。
- OpenAPI 文档或等价接口说明文档。

需要优先固定的接口：

- POST /api/v1/upload/photos
- POST /api/v1/ai/preprocess
- POST /api/v1/ai/keypoints
- POST /api/v1/ai/structure
- POST /api/v1/params/generate
- POST /api/v1/3d/generate
- POST /api/v1/pattern/generate
- GET /api/v1/files/download/dxf/{file_id}
- POST /api/v1/engine/garmentcode/3d

验收标准：

- 前端、后端、AI、引擎模块均可基于同一份契约对接。
- mock 实现与后续真实实现不需要改动接口定义。

### 阶段 3：MVP 主链路打通

目标：实现一个可演示的闭环版本，覆盖上传、参数预填、参数编辑、3D 预览。

交付内容：

- Electron + React 前端基础界面。
- 拍照上传页面与本地文件选择。
- 前端照片质量检测逻辑。
- MinIO 文件上传。
- 参数模板服务。
- 动态参数表单。
- 参数联动逻辑。
- 3D 预览页面。
- 统一错误提示与加载状态。

本阶段允许：

- AI 预处理、关键点识别、结构识别先使用 mock 数据。
- 工业 DXF 导出先保留接口和任务骨架。

验收标准：

- 用户可以从上传照片进入参数页。
- 前端可以获取完整参数集并修改。
- 点击“更新 3D 预览”后能看到可加载的 GLB 模型或默认回退模型。

### 阶段 4：真实 AI 能力接入

目标：将阶段 3 的 mock AI 替换为文档要求的真实能力链路。

交付内容：

- OpenCV 图像预处理服务。
- YOLOv8-seg 实例分割。
- YOLOv8-pose / MediaPipe 关键点检测与融合。
- CLIP 特征提取。
- Milvus 相似模板检索。
- 通义千问 V3 结构解析。
- 结构融合逻辑。
- AI 结果缓存与错误处理。

验收标准：

- 输入照片后，可稳定输出预处理图片、关键点数据、比例参数和结构识别结果。
- AI 结果严格不包含绝对尺寸推断。

### 阶段 5：GarmentCode 真实 3D 预览

目标：完成 GarmentCode 到前端 3D 预览的真实链路。

交付内容：

- GarmentCode 参数映射器。
- 模板参数合并逻辑。
- pygarment 生成 2D 裁片。
- mesh 生成与 OBJ 导出。
- trimesh 转 GLB。
- GLB 上传 MinIO。
- Redis 缓存参数哈希与 GLB URL。
- 前端 Google ModelViewer 集成。

验收标准：

- 前端可加载真实 GLB 文件。
- 支持旋转、缩放、平移和重置视角。
- 3D 生成失败时可回退到默认 3D 模型。

### 阶段 6：工业版型与 DXF 导出

目标：完成最关键的工业输出链路。

交付内容：

- GarmentCode JSON 裁片输出。
- 双向转换器：GarmentCode JSON -> Seamly2D XML。
- Seamly2D CLI 调用封装。
- 缝份、剪口、钻眼、布纹线、放码参数接入。
- DXF 上传 MinIO。
- DXF 下载接口。

验收标准：

- 用户确认参数后可以生成 ASTM DXF 2000 文件。
- DXF 文件可下载并具备基础生产所需工艺信息。

### 阶段 7：生产化部署与运维

目标：将系统提升为可部署、可观测、可运维的生产级环境。

交付内容：

- Kong / Envoy 网关配置。
- Kubernetes 部署清单。
- Prometheus / Grafana 监控。
- ELK 日志收集。
- GitHub Actions 持续集成与部署流水线。
- 配置中心、密钥注入、环境区分。
- 灰度发布与失败回滚策略。

验收标准：

- 测试环境可一键部署。
- 关键服务具备健康检查、日志、指标和告警。

## 5. 推荐开发顺序

建议按以下顺序推进：

1. 先完成阶段 0，验证技术可行性。
2. 立即完成阶段 1 和阶段 2，固定工程骨架与接口契约。
3. 以阶段 3 为第一里程碑，交付可演示 MVP。
4. 并行推进阶段 4 和阶段 5，分别接入真实 AI 与真实 3D。
5. 单独攻坚阶段 6，完成工业 DXF 输出。
6. 最后做阶段 7，补齐部署、监控和 CI/CD。

## 6. 首期 MVP 范围

首期建议只覆盖以下范围：

1. 桌面端照片上传与基础质量检测。
2. 后端上传文件到 MinIO。
3. AI 接口按最终协议返回 mock 结构参数和比例参数。
4. 参数服务基于模板生成完整 GarmentCode 参数集。
5. 前端动态表单支持编辑和联动。
6. 3D 服务生成真实或回退 GLB 预览。
7. 工业版型导出先保留接口与任务骨架，不在首期强行完成。

这样可以尽快形成一个可演示、可扩展、接口稳定的版本。

## 7. 第一阶段详细任务清单

如果从现在开始实施，建议第一批任务如下：

1. 创建 garment-ai 目录结构。
2. 初始化 frontend、backend、ai、engine 四个模块的依赖管理文件。
3. 创建统一配置文件和环境变量模板。
4. 编写 docker-compose.yml，拉起 PostgreSQL、Redis、MinIO、Milvus。
5. 定义 protobuf 与 REST 接口契约。
6. 建立前端基础壳、后端 API 壳、AI 服务壳、引擎服务壳。
7. 完成健康检查、日志、错误响应封装。
8. 编写阶段 0 的依赖验证脚本。
9. 编写 README 与 docs/architecture、docs/api、docs/setup 文档。

## 8. 当前建议

下一步最合适的动作不是直接写全量业务代码，而是先进入“阶段 0 + 阶段 1”。

具体建议：

1. 先验证 pygarment、trimesh、seamly2d-cli、Milvus、通义千问 API 的最小运行能力。
2. 验证通过后，立即生成完整项目骨架与接口契约。
3. 在保证接口不变的情况下，以 MVP 方式逐步替换 mock 为真实能力。
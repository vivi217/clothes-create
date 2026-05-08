# Garment AI 技术栈说明

本文档基于仓库内 `README.md`、`docker-compose.yml`、各模块依赖清单与 CI 配置整理，描述 **garment-ai** 子项目当前采用的技术栈与运行拓扑。

## 1. 项目定位

面向服装 AI 制版系统的多服务架构：**桌面/Web 前端** + **Go 业务网关** + **Python 视觉与编排服务**，配合 **PostgreSQL、Redis、MinIO、Milvus** 等基础设施，通过 **Docker Compose** 本地编排。

当前阶段以 API 契约、健康检查、Mock/流水线骨架为主；完整领域流程见 `README.md` 与各阶段 `docs/` 说明。

## 2. 仓库与模块划分

| 目录 | 职责 |
|------|------|
| `frontend/` | React + TypeScript 界面；Vite 开发；Electron 桌面壳 |
| `backend/` | Go HTTP API（Gin）、对象存储与下游服务编排 |
| `ai/` | Python FastAPI：预处理、关键点、结构分析、向量检索与视觉模型 |
| `engine/` | Python FastAPI：GarmentCode / Seamly2D 等 3D 与制版编排（骨架） |
| `deploy/` | 容器相关资产（若存在） |
| `docs/` | 阶段说明与 API 契约 |
| `scripts/` | 校验、`protoc` 生成等脚本 |

仓库根目录另有 `implementation-plan.md`、`scripts/stage0/` 等环境与依赖检查脚本，与 **Seamly2D、Qwen、PyGarment** 等生态对齐，不等同于 `garment-ai` 运行时镜像内容。

## 3. 前端（`frontend/`）

| 类别 | 技术 | 版本（以 `package.json` 为准） |
|------|------|-------------------------------|
| 语言 | TypeScript | 5.4.5 |
| UI 框架 | React | 18.2.0 |
| 构建/开发服务器 | Vite | 5.2.8 |
| React 插件 | `@vitejs/plugin-react` | 4.2.1 |
| 样式 | Tailwind CSS、PostCSS、Autoprefixer | 3.4.x / 8.4.x / 10.4.x |
| 桌面壳 | Electron | 29.0.0 |
| 画布 | Fabric.js | 5.3.0 |
| 类型定义 | `@types/react`、`@types/node` 等 | 见 `package.json` |

**环境变量（构建期）**：`VITE_BACKEND_BASE_URL`、`VITE_AI_BASE_URL`、`VITE_ENGINE_BASE_URL`（Compose 中指向本机各服务端口）。

**容器**：`node:22-bookworm-slim`，默认 `npm run dev`（开发模式，非生产静态资源）。

## 4. 后端（`backend/`）

| 类别 | 技术 | 说明 |
|------|------|------|
| 语言 | Go | 1.22.1（`go.mod`、CI、Dockerfile 一致） |
| Web 框架 | Gin | `github.com/gin-gonic/gin` v1.9.1 |
| CORS | `gin-contrib/cors` | v1.7.2 |
| 对象存储客户端 | minio-go v7 | v7.0.76 |
| 序列化 | Protobuf（runtime 间接依赖） | `google.golang.org/protobuf` v1.34.0 |

**契约**：`backend/proto/garment.proto`；生成脚本 `scripts/generate-proto.sh`（`protoc` + `--go_out` / `--go-grpc_out`）。

**容器**：多阶段构建，`golang:1.22.1-bookworm` 编译，`distroless/base-debian12` 运行，默认监听 **8080**。

## 5. AI 服务（`ai/`）

| 类别 | 技术 | 版本（运行时以 `requirements_new.txt` 为准） |
|------|------|-----------------------------------------------|
| 语言 | Python | 3.11（Dockerfile / CI） |
| Web 框架 | FastAPI | 0.110.0 |
| ASGI 服务器 | Uvicorn | 0.29.0 |
| 对象存储 | MinIO Python SDK | 7.2.5 |
| HTTP 客户端 | httpx | 0.27.0 |
| 数值计算 | NumPy | 1.26.4 |
| 图像 | OpenCV（headless） | 4.10.0.84 |
| 分割/检测 | Ultralytics（YOLOv8 等） | 8.1.0 |
| 向量库 | PyMilvus | 2.3.7 |
| 多模态/嵌入 | Hugging Face Transformers（CLIP 等） | 4.41.2 |
| 校验/模式 | Marshmallow | 3.21.3 |

**基础镜像**：`Dockerfile` 基于自定义 `ai_base:v1.2`（由 `Dockerfile_base` 从 `python:3.11-slim-bookworm` 构建，含 `libgl1` 等系统库）；镜像内再安装 `requirements_new.txt`。

**典型环境变量**：Milvus 地址、MinIO、DashScope / Qwen 相关（`QWEN_MODEL`、`QWEN_BASE_URL`、`DASHSCOPE_API_KEY`）、CLIP/YOLO 模型路径与缓存目录等（详见 `docker-compose.yml`）。

## 6. Engine 服务（`engine/`）

| 类别 | 技术 | 版本 |
|------|------|------|
| 语言 | Python | 3.11 |
| Web | FastAPI + Uvicorn | 与 `ai/` 同主版本线 |
| 对象存储 | MinIO SDK | 7.2.5 |
| 3D 网格 | Trimesh | 4.4.0 |

默认端口 **8010**。

## 7. 基础设施（Docker Compose）

| 服务 | 镜像/版本 | 用途 |
|------|-----------|------|
| PostgreSQL | `postgres:15.5` | 关系数据 |
| Redis | `redis:7.2.4` | 缓存/队列（按配置使用） |
| MinIO | `quay.io/minio/minio:latest` | S3 兼容对象存储 |
| etcd | `quay.io/coreos/etcd:v3.5.12` | Milvus 元数据依赖 |
| Milvus | `milvusdb/milvus:v2.3.3` | 向量检索（standalone） |

应用服务 **`backend`**、**`ai`**、**`engine`**、**`frontend`** 均由本地 Dockerfile 构建并接入上述依赖。

## 8. DevOps 与质量

| 项目 | 说明 |
|------|------|
| 编排 | Docker Compose v2（`docker compose`） |
| 本地命令 | `Makefile`：`build`、`up`、`down`、`health`、`validate`、`proto`、`clean` |
| CI | GitHub Actions（`.github/workflows/ci.yml`）：Node 22、Go 1.22.1、Python 3.11；前端 `npm run typecheck`、后端 `go test ./...`、Python `py_compile`、`docker compose config` |
| 代理（构建） | 后端 Dockerfile 使用 `GOPROXY=https://goproxy.cn,direct`；部分基础镜像来自 DaoCloud 镜像前缀以加速拉取 |

## 9. 服务端口速查（默认）

| 服务 | 端口 |
|------|------|
| frontend (Vite) | 3000 |
| backend | 8080 |
| ai | 8000 |
| engine | 8010 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| MinIO API / Console | 9000 / 9001 |
| etcd | 2379 |
| Milvus gRPC / HTTP 健康 | 19530 / 9091 |

端口均可通过 `.env` 中变量覆盖（见 `docker-compose.yml` 与 `.env.example`）。

## 10. 小结

- **语言**：TypeScript（前端）、Go（网关与业务 API）、Python（AI 与 Engine）。
- **核心框架**：React 18、Vite 5、Electron 29、Gin、FastAPI、Uvicorn。
- **数据与存储**：PostgreSQL、Redis、MinIO、Milvus（+ etcd）。
- **视觉与 AI**：OpenCV、Ultralytics、Transformers、PyMilvus；可选云端 Qwen/DashScope。
- **交付**：多容器 Docker Compose；CI 覆盖类型检查、Go 测试、Python 语法与 Compose 校验。

文档版本与代码同步维护；若依赖升级，请以各模块 `package.json`、`go.mod`、`requirements*.txt` 及 Compose 镜像标签为准。

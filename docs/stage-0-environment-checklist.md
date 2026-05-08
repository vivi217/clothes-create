# 阶段 0 环境验证清单

## 目标

本阶段用于验证服装 AI 打版工具的高风险依赖、运行时环境和外部服务接入条件，避免在正式开发阶段因底层环境不可用而大规模返工。

## 当前版本决策

项目原始架构文档指定 AI 层使用 Python 3.10.13。本仓库当前已确认改为 Python 3.11 基线推进，后续 AI 服务相关依赖版本和 Docker 基础镜像均以 Python 3.11 兼容性为准。

## 使用方式

1. 先执行 `scripts/stage0/run_all_checks.sh`。
2. 按输出结果修复缺失依赖。
3. 对需要凭据或外部服务的检查项，先配置环境变量后重试。
4. 将每次验证结果记录到本文件末尾的“验证记录”章节。

## 验证项

### 1. 基础工具链

验证内容：

- Git
- Docker
- Docker Compose
- Python 3.11.x
- pip
- Node.js
- npm
- Go 1.22.x
- protoc
- kubectl

脚本：`scripts/stage0/check_system.sh`

通过标准：

- 所有命令都可执行。
- Python 主版本为 3.11。
- Go 主版本为 1.22。

### 2. GarmentCode / pygarment

验证内容：

- `pygarment` 可导入。
- 存在 `Garment` 类型。
- 存在 `meshgen.GarmentMeshGenerator`。

脚本：`scripts/stage0/check_pygarment.py`

通过标准：

- 模块导入成功。
- 关键类型存在。

### 3. OBJ -> GLB 转换链路

验证内容：

- `trimesh` 可导入。
- 可创建最小三角网格。
- 可导出 OBJ。
- 可导出 GLB。

脚本：`scripts/stage0/check_obj_to_glb.py`

通过标准：

- 在临时目录中成功生成 OBJ 和 GLB 文件。

### 4. Seamly2D CLI

验证内容：

- `seamly2d-cli` 命令存在。
- `--help` 可执行。
- 能输出版本或帮助信息。

脚本：`scripts/stage0/check_seamly2d.sh`

通过标准：

- 命令存在且能正常返回帮助信息。

### 5. MinIO SDK / 连接能力

验证内容：

- Python `minio` SDK 可导入。
- 配置凭据后可连通 MinIO。
- 可检查目标桶是否存在。

脚本：`scripts/stage0/check_minio.py`

环境变量：

- `MINIO_ENDPOINT`
- `MINIO_ACCESS_KEY`
- `MINIO_SECRET_KEY`
- `MINIO_SECURE`
- `MINIO_BUCKET`

通过标准：

- SDK 导入成功。
- 凭据完整时可以连通实例。

### 6. 通义千问 V3 API

验证内容：

- API Key 已配置。
- 能成功请求模型列表或最小聊天请求。

脚本：`scripts/stage0/check_qwen_api.py`

环境变量：

- `DASHSCOPE_API_KEY`
- `QWEN_BASE_URL`，可选，默认 `https://dashscope.aliyuncs.com/compatible-mode/v1`
- `QWEN_MODEL`，可选，默认 `qwen-max`

通过标准：

- 返回 HTTP 200。
- 返回内容可解析。

### 7. 视觉与检索能力栈

验证内容：

- `cv2`
- `ultralytics`
- `mediapipe`
- `clip` 或兼容实现
- `pymilvus`

脚本：`scripts/stage0/check_vision_stack.py`

通过标准：

- 关键模块可导入。
- `pymilvus` 安装后可读取客户端类型。

## 推荐执行顺序

1. 先跑基础工具链。
2. 再跑 Python 相关依赖。
3. 然后验证外部服务和 API。
4. 最后汇总所有失败项，决定是否进入阶段 1。

## 进入阶段 1 的门槛

- 基础工具链全部通过。
- `pygarment`、`trimesh`、`seamly2d-cli` 至少具备基础可用性。
- MinIO 和通义千问至少能完成一次连通验证。
- 视觉栈核心依赖可导入，即使模型权重尚未下载完整，也要确认框架版本兼容。

## 验证记录

| 日期 | 执行人 | 结果 | 说明 |
| --- | --- | --- | --- |
| 待执行 |  |  |  |
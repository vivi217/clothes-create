服装 AI 打版工具 生产级系统架构设计文档
一、给 AI 编程工具的明确指令
基于本文档生成完整的前后端代码、Dockerfile、docker-compose.yml、K8s 部署配置、CI/CD 脚本。
所有工具必须使用本文档指定的版本和调用方式，禁止私自替换版本。
所有接口路径、参数名称、数据结构必须与本文档完全一致，不得私自增减字段、改路由。
严格遵循 AI 边界：AI 只做结构识别、参数预填，绝对不能生成、预测任何服装绝对尺寸，尺寸必须人工确认修改。
3D 可视化部分必须使用 GarmentCode 原生 3D 输出 + Google ModelViewer，禁止编写任何自定义 3D 渲染代码。
双引擎调用必须严格按固定流程：
GarmentCode 生成裁片 → 自研双向转换器转 Seamly2D 格式 → Seamly2D 无头 CLI 生成工业 DXF。
生成代码要求：可直接本地 / 容器运行、自带完整依赖管理、配置文件、启动脚本、注释齐全。
二、系统整体架构概述
2.1 核心技术栈（版本锁定，不得修改）
表格
层级	技术选型（精确固定版本）
前端	React 18.2.0 + TypeScript 5.4.0 + Electron 29.0.0 + Fabric.js 5.3.0 + Tailwind CSS 3.4.0 + Google ModelViewer 3.5.0
网关	Kong 3.6.0 + Envoy 1.29.0 + gRPC 1.62.0
后端微服务	Go 1.22.1 + gRPC 1.62.0 + Protobuf 3.25.3 + etcd 3.5.12 + Gin 1.9.1
AI 层	Python 3.10.13 + FastAPI 0.110.0 + LangChain 0.1.13 + 通义千问 V3 API + YOLOv8-seg 8.1.0 + MediaPipe 0.10.9 + CLIP + Milvus 2.3.3
双引擎核心	GarmentCode 0.3.0（pygarment）、Seamly2D 2024.1.0（seamly2d-cli 无头模式）、自研双向转换器
数据库 / 存储	PostgreSQL 15.5、Milvus 2.3.3、MinIO RELEASE.2024-03-15T01-35-08Z、Redis 7.2.4
基础设施	Docker 25.0.3 + Kubernetes 1.29.2 + Prometheus 2.49.1 + Grafana 10.4.0 + ELK 8.13.0 + GitHub Actions
2.2 核心数据流转总览
plaintext
前端拍照上传 
→ OpenCV+YOLOv8图像预处理 
→ YOLOv8-seg实例分割 
→ MediaPipe关键点检测 
→ 通义千问V3结构解析 
→ 生成GarmentCode参数集 
→ 预填前端表单 
→ 点击更新3D预览 
→ 调用GarmentCode generate_pattern() 
→ GarmentCode生成OBJ 3D模型 
→ trimesh转GLB 
→ MinIO存储 
→ Google ModelViewer加载展示 
→ 人工确认参数 
→ GarmentCode生成JSON裁片 
→ 双向转换器转Seamly2D XML 
→ seamly2d-cli无头调用添加工艺+放码 
→ 输出ASTM DXF 2000工业文件 
→ 前端下载导出
三、核心业务完整流转流程（11 步闭环）
步骤 1：打版师前端拍照上传（前端交互层）
输入
打版师电脑摄像头拍摄 / 本地上传三类照片：
正面全身照（必填）：样衣平铺 / 标准人台、纯白背景、光线均匀
侧面照（必填）：展示厚度廓形、同拍摄距离
局部细节照（可选最多 3 张）：领型、袖型、口袋细节
处理逻辑
前端调用 navigator.mediaDevices.getUserMedia() 调取摄像头实时流；
叠加水平 / 垂直参考线：对齐肩线、腰线、服装中心线；
拍照后前端 OpenCV.js 初步裁剪、扶正；
前端质量检测规则：
模糊度：Laplacian 方差＜100 → 提示「照片模糊，请重拍」
亮度：平均亮度＜50 或 ＞200 → 提示「光线不足 / 过曝，请重拍」
服装占比：画面占比＜60% → 提示「请将样衣置于画面中央」
统一压缩分辨率 1920×1080，单文件≤2MB，JPG 格式。
依赖工具
React 18 + Electron + OpenCV.js 4.8.0
前端接口
POST /api/v1/upload/photos 上传至 MinIO
输出
3 张标准 JPG 照片 + MinIO 访问 URL
步骤 2：AI 图像预处理（AI 能力层）
输入
MinIO 返回的 3 张原图 URL
处理逻辑
OpenCV 4.8.0 读取图像；
加载 YOLOv8-seg 服装专用模型 做实例分割，生成像素掩码；
依据掩码抠图、去除背景，保留纯服装主体；
白平衡校正、对比度增强；
正 / 侧 / 局部多视角图像坐标对齐。
依赖工具
Python 3.10 + OpenCV 4.8.0 + YOLOv8-seg 8.1.0
AI 接口
POST /api/v1/ai/preprocess
输出
3 张透明背景 PNG 预处理图 + MinIO URL
步骤 3：多视角服装关键点检测（AI 能力层）
输入
预处理后 3 张照片 URL
处理逻辑
加载 yolov8-pose-garment.pt 服装专用关键点模型，共 128 个服装结构关键点；
正 / 侧 / 局部分别检测关键点；
多视角关键点融合，生成全局坐标集；
自动计算结构相对比例：肩宽 / 胸围、衣长 / 身高、袖长 / 臂长、领宽 / 肩宽等。
依赖工具
YOLOv8-pose 8.1.0 + MediaPipe 0.10.9
AI 接口
POST /api/v1/ai/keypoints
输出示例 JSON
json
{
  "keypoints": {
    "left_shoulder": [x1, y1],
    "right_shoulder": [x2, y2],
    "neck_point": [x3, y3],
    "chest_point": [x4, y4],
    "waist_point": [x5, y5],
    "hip_point": [x6, y6],
    "left_cuff": [x7, y7],
    "right_cuff": [x8, y8],
    "collar_left": [x9, y9],
    "collar_right": [x10, y10]
  },
  "ratios": {
    "shoulder_to_chest": 0.65,
    "length_to_height": 0.45,
    "sleeve_to_arm": 0.85,
    "collar_to_shoulder": 0.35
  }
}
步骤 4：服装结构特征提取与款式分类（AI 能力层）
输入
预处理照片 URL + 关键点坐标集
处理逻辑
调用 CLIP 生成服装特征向量；
调用 Milvus 2.3.3 向量库，检索相似度最高前 5 个款式模板；
调用通义千问 V3，传入图像 + 关键点，固定 Prompt 解析服装结构；
融合向量检索结果 + 大模型识别结果，取最高置信度作为最终结构参数。
固定 Prompt
text
你是专业的服装结构分析师，请根据这张样衣照片和关键点坐标，识别以下信息：
1. 服装品类：上衣/裤子/裙子/外套/连衣裙
2. 廓形：H型/X型/A型/O型/T型
3. 领型：圆领/V领/翻领/立领/西装领/娃娃领
4. 袖型：长袖/短袖/无袖/泡泡袖/喇叭袖/插肩袖
5. 门襟类型：单排扣/双排扣/拉链/套头/暗扣
6. 口袋类型：贴袋/插袋/挖袋/无口袋
7. 口袋数量：0/1/2/更多
8. 省道位置：前腰/后腰/胸省/肩省/无省道
9. 省道数量：0/1/2/更多
10. 是否有腰带：是/否
11. 是否有拉链：是/否
12. 缝份类型：平缝/包缝/锁边缝
输出严格为JSON格式，不要任何额外文字。
依赖工具
CLIP + Milvus 2.3.3 + 通义千问 V3 API
AI 接口
POST /api/v1/ai/structure
输出示例 JSON
json
{
  "garment_type": "shirt",
  "silhouette": "H型",
  "collar_type": "翻领",
  "sleeve_type": "长袖",
  "placket_type": "单排扣",
  "pocket_type": "贴袋",
  "pocket_count": 2,
  "dart_position": "前腰",
  "dart_count": 2,
  "has_belt": false,
  "has_zipper": false,
  "seam_type": "平缝",
  "confidence": 0.92
}
步骤 5：生成 GarmentCode 全量参数集并预填前端表单（业务服务层）
输入
服装结构参数 + 关键点相对比例参数
处理逻辑
Go 服务做参数映射，将 AI 识别字段转为 GarmentCode 标准字段；
从 PostgreSQL templates 表加载对应款式默认模板参数；
合并 AI 识别参数、比例参数、模板默认参数，生成完整 GarmentCode 参数集；
参数分三类：
基础尺寸参数：仅给默认值，必须人工手动修改
结构组件参数：AI 自动预填，可人工微调
工艺细节参数：系统默认，可人工修改
返回参数集给前端，自动填充动态表单。
依赖工具
Go 1.22 + PostgreSQL 15.5
业务接口
POST /api/v1/params/generate
输出
完整 GarmentCode 标准参数 JSON
步骤 6：打版师手动改参数 + 更新 3D 预览（前端交互层）
输入
AI 预填完整 GarmentCode 参数集
处理逻辑
前端三级分类动态表单：
一级：基础尺寸（衣长、胸围、肩宽、领围、袖长、腰围、臀围等）
二级：结构组件（领型、袖型、门襟、口袋、省道）
三级：工艺细节（缝份宽度、剪口、布纹线、放码规则）
每个参数展示默认值、取值范围、单位 cm；
支持参数联动：改胸围自动联动袖窿、改衣长联动侧缝；
人工微调确认后，点击「更新 3D 预览」提交参数。
依赖工具
React 18 + TypeScript + Tailwind CSS
前端接口
POST /api/v1/3d/generate
输出
人工确认后的完整 GarmentCode 参数集
步骤 7：调用 GarmentCode 生成 3D 模型（双引擎核心层）
输入
人工确认完整 GarmentCode 参数集
处理逻辑
调度服务传入参数至 pygarment；
Garment() 构造实例 → generate_pattern() 生成 2D 裁片；
调用 GarmentMeshGenerator 生成基础 3D 几何模型（MVP 禁用物理模拟）；
导出 OBJ → trimesh 加载转 GLB 格式（ModelViewer 原生支持）。
依赖工具
Python 3.10 + pygarment 0.3.0 + trimesh 4.4.0
双引擎接口
POST /api/v1/engine/garmentcode/3d
输出
GLB 模型文件：≤1MB、顶点数≤2000
步骤 8：3D 模型上传 MinIO 并返回 URL（业务服务层）
输入
GLB 3D 模型文件
处理逻辑
上传至 MinIO garment-3d-models 存储桶；
生产环境生成预签名 URL，测试环境公开 URL；
参数哈希 + URL 写入 Redis 缓存，有效期 1 小时；
返回可访问 URL 给前端。
依赖工具
Go 1.22 + MinIO + Redis 7.2.4
业务接口
POST /api/v1/files/upload/3d
输出
GLB 可访问 URL
步骤 9：Google ModelViewer 3D 预览渲染（前端交互层）
输入
GLB 文件 URL
处理逻辑
CDN 引入 ModelViewer 3.5.0；
绑定 src 自动加载渲染；
内置交互：拖拽旋转、滚轮缩放、右键平移、双击重置、自动旋转；
支持左右 / 上下分屏、全屏预览模式。
前端固定代码
html
预览
<script type="module" src="https://ajax.googleapis.com/ajax/libs/model-viewer/3.5.0/model-viewer.min.js"></script>
<model-viewer
  id="garmentViewer"
  src=""
  alt="服装3D预览"
  camera-controls
  auto-rotate
  rotation-per-second="30deg"
  shadow-intensity="1"
  exposure="1.5"
  style="width: 100%; height: 100%;"
></model-viewer>
输出
前端实时 3D 服装模型可视化
步骤 10：打版师确认版型 → 生成工业版型（前端交互层）
输入
3D 预览确认后的最终 GarmentCode 参数
处理逻辑
人工旋转缩放检查版型，可退回步骤 6 反复微调；
确认无误点击「生成工业版型」；
前端提交最终参数至版型生成服务。
前端接口
POST /api/v1/pattern/generate
输出
最终定稿 GarmentCode 参数集
步骤 11：双引擎生成工业 DXF 并导出（双引擎核心层）
输入
最终确认 GarmentCode 参数集
处理逻辑
GarmentCode 生成标准 2D 裁片 JSON；
自研双向转换器：GarmentCode JSON → Seamly2D XML，精度 ±0.01cm；
调用 seamly2d-cli 无头模式执行命令，自动：
添加 1cm 标准缝份
生成剪口、钻眼、布纹线
国标多号型自动放码
输出 ASTM DXF 2000 工业格式
DXF 上传 MinIO，前端生成下载链接，可直接交付工厂裁剪生产。
固定 CLI 调用命令
bash
运行
seamly2d-cli --input pattern.xml --output pattern.dxf \
--add-seam-allowance 1cm --add-notches --add-drill-holes --grade \
--size-set "国标女160/84A,165/88A,170/92A" --format "ASTM DXF 2000"
业务接口
GET /api/v1/files/download/dxf/{file_id}
输出
标准工业 ASTM DXF 2000 版型文件
四、关键技术强制规范
4.1 接口规范
所有 REST 接口统一 JSON；
固定统一响应结构：
json
{
  "code": 0,
  "message": "success",
  "data": {}
}
微服务内部通信统一 gRPC + Protobuf；
路由固定格式：/api/v1/[服务名]/[功能]。
4.2 命名规范
变量 / 函数：小驼峰 camelCase
类名：大驼峰 PascalCase
常量：全大写下划线 UPPER_SNAKE_CASE
数据库表名：小写下划线 snake_case
接口路径：小写连字符 kebab-case
4.3 数据格式规范
所有尺寸单位：厘米 cm，保留 2 位小数；
时间统一 ISO 8601 格式；
文件编码统一 UTF-8；
DXF 固定版本：ASTM DXF 2000。
4.4 错误处理规范
所有接口必须返回明确错误码 + 可读提示；
前端友好弹窗提示；
后端完整日志记录；
3D 生成失败自动 fallback 同款默认 3D 模型；
Seamly2D 调用失败返回明确文案，引导人工核对参数。
附录 A：GarmentCode 完整参数示例
json
{
  "garment_type": "shirt",
  "silhouette": "H型",
  "length": 65.0,
  "chest": 96.0,
  "shoulder": 42.0,
  "neck": 38.0,
  "sleeve_length": 60.0,
  "cuff": 22.0,
  "waist": 84.0,
  "hip": 98.0,
  "collar_type": "翻领",
  "collar_width": 8.0,
  "collar_depth": 5.0,
  "sleeve_type": "长袖",
  "sleeve_cuff_type": "单扣",
  "placket_type": "单排扣",
  "placket_length": 20.0,
  "button_count": 6,
  "pocket_type": 贴袋,
  "pocket_count": 2,
  "pocket_width": 12.0,
  "pocket_height": 14.0,
  "pocket_position_x": 15.0,
  "pocket_position_y": 25.0,
  "dart_position": "前腰",
  "dart_count": 2,
  "dart_length": 10.0,
  "dart_width": 2.0,
  "seam_allowance": 1.0,
  "notch_type": "标准剪口",
  "grain_line": "经向",
  "grade_rule": "国标女"
}
附录 B：核心 Protobuf 定义
protobuf
syntax = "proto3";

package garment;

service GarmentService {
  rpc GeneratePattern(GeneratePatternRequest) returns (GeneratePatternResponse);
  rpc Generate3DModel(Generate3DRequest) returns (Generate3DResponse);
}

message GeneratePatternRequest {
  string params_json = 1;
}

message GeneratePatternResponse {
  int32 code = 1;
  string message = 2;
  string pattern_json = 3;
  string dxf_url = 4;
}

message Generate3DRequest {
  string params_json = 1;
}

message Generate3DResponse {
  int32 code = 1;
  string message = 2;
  string glb_url = 3;
}
项目固定目录结构（AI 生成必须严格遵守）
plaintext
garment-ai/
├── frontend/          # React+Electron前端
├── backend/           # Go微服务后端
├── ai/                # Python FastAPI AI服务
├── engine/            # GarmentCode+Seamly2D双引擎、双向转换器
├── deploy/            # Docker、docker-compose、K8s yaml
├── scripts/           # GitHub Actions CI/CD、部署脚本
└── docs/              # 项目文档、接口文档、架构图
给 AI 编程工具最终强制执行要求
按以上目录结构生成全量代码；
每个子模块自带完整依赖文件：package.json、go.mod、requirements.txt；
生成可一键启动的 docker-compose.yml；
生成完整 K8s 部署 YAML、命名空间、配置映射、存储卷、服务、部署；
生成 GitHub Actions CI/CD 流水线脚本；
所有配置抽离为环境变量 / 配置文件，支持容器与 K8s 注入；
关键业务逻辑、模型调用、引擎调用加详细注释；
生成根目录 README.md，包含环境依赖、部署步骤、接口说明、启动命令。

服装 AI 打版工具 生产级系统架构设计文档（AI 代码生成专用版）
文档用途：专供 GitHub 专业 AI 编程工具生成全套可运行代码使用
核心要求：严格按照本文档指定的工具、接口、数据格式和流程生成代码，不得自行修改技术选型或业务逻辑
技术栈严格约束：完全复用指定开源组件，禁止自研任何服装结构逻辑、3D 渲染引擎或 AI 底层模型
一、给 AI 编程工具的明确指令
基于本文档生成完整的前后端代码、Dockerfile、docker-compose.yml、K8s 部署配置、CI/CD 脚本
所有工具必须使用本文档指定的版本和调用方式，禁止替换
所有接口路径、参数名称、数据结构必须与本文档完全一致
严格遵循 AI 边界：AI 只做结构识别和参数预填，绝对不能生成或预测任何绝对尺寸
3D 可视化部分必须使用 GarmentCode 原生 3D 输出 + Google ModelViewer，禁止编写任何自定义 3D 渲染代码
双引擎调用必须严格按照本文档的流程：GarmentCode 生成裁片→双向转换器转 Seamly2D 格式→Seamly2D 无头调用生成 DXF
生成的代码必须可直接运行，包含完整的依赖管理、配置文件和启动脚本
二、系统整体架构概述
2.1 核心技术栈（严格执行，不得修改）
表格
层级	技术选型（精确到版本）
前端	React 18.2.0 + TypeScript 5.4.0 + Electron 29.0.0 + Fabric.js 5.3.0 + Tailwind CSS 3.4.0 + Google ModelViewer 3.5.0
网关	Kong 3.6.0 + Envoy 1.29.0 + gRPC 1.62.0
后端微服务	Go 1.22.1 + gRPC 1.62.0 + Protobuf 3.25.3 + etcd 3.5.12 + Gin 1.9.1
AI 层	Python 3.10.13 + FastAPI 0.110.0 + LangChain 0.1.13 + 通义千问 V3 API + YOLOv8-seg 8.1.0 + MediaPipe 0.10.9 + CLIP + Milvus 2.3.3
双引擎核心	GarmentCode 0.3.0（pygarment）、Seamly2D 2024.1.0（seamly2d-cli 无头模式）、自研双向转换器
数据库	PostgreSQL 15.5、Milvus 2.3.3、MinIO RELEASE.2024-03-15T01-35-08Z、Redis 7.2.4
基础设施	Docker 25.0.3 + Kubernetes 1.29.2 + Prometheus 2.49.1 + Grafana 10.4.0 + ELK 8.13.0 + GitHub Actions
2.2 核心数据流转总览
plaintext
前端拍照上传 → OpenCV+YOLOv8图像预处理 → YOLOv8-seg实例分割 → MediaPipe关键点检测 → 通义千问V3结构解析 → 生成GarmentCode参数集 → 预填前端表单 → 点击"更新3D预览" → 调用GarmentCode generate_pattern() → GarmentCode生成OBJ 3D模型 → trimesh转换为GLB → MinIO存储 → Google ModelViewer加载展示 → 确认参数 → GarmentCode生成JSON裁片 → 双向转换器转Seamly2D XML → seamly2d-cli无头调用添加工艺+放码 → 输出ASTM DXF 2000文件 → 前端下载导出
三、核心业务完整流转流程（超详细 11 步闭环，精确到工具调用）
步骤 1：打版师前端拍照上传（前端交互层）
输入：打版师通过电脑摄像头拍摄或本地选择 3 类照片
正面全身照（必填）：样衣平铺或穿在标准人台上，背景为白色，光线均匀
侧面照（必填）：展示服装厚度与廓形，与正面照同一拍摄距离
局部细节照（可选，最多 3 张）：分别拍摄领型、袖型、口袋等关键结构
处理逻辑：
前端调用navigator.mediaDevices.getUserMedia()获取摄像头流
显示实时参考线：水平参考线对齐肩线和腰线，垂直参考线对齐中心线
拍照后自动调用OpenCV.js进行初步裁剪和扶正
调用前端拍照质量检测函数：
检测模糊度：使用 Laplacian 方差，方差 < 100 提示 "照片模糊，请重拍"
检测亮度：平均亮度 <50 或> 200 提示 "光线不足 / 过曝，请重拍"
检测服装占比：服装占比 < 60% 提示 "请将样衣放在画面中央"
调用工具：React 18 + Electron + OpenCV.js 4.8.0
输出：3 张 JPG 格式照片，分辨率统一压缩为 1920×1080，文件大小≤2MB
前端接口：POST /api/v1/upload/photos（上传到 MinIO）
步骤 2：AI 图像预处理（AI 能力层）
输入：MinIO 返回的 3 张照片 URL
处理逻辑：
调用OpenCV 4.8.0读取照片
调用YOLOv8-seg.pt（服装专用模型）进行服装实例分割，生成掩码
根据掩码移除背景，只保留服装主体
对照片进行白平衡校正和对比度增强
将正面、侧面、局部照片对齐到统一坐标系
调用工具：Python 3.10 + OpenCV 4.8.0 + YOLOv8-seg 8.1.0
输出：3 张预处理后的 PNG 格式照片（透明背景）
AI 层接口：POST /api/v1/ai/preprocess
步骤 3：多视角服装关键点检测（AI 能力层）
输入：预处理后的 3 张照片 URL
处理逻辑：
加载预训练的yolov8-pose-garment.pt模型（标注 128 个服装关键点）
分别对正面、侧面、局部照片进行关键点检测
融合多视角关键点数据，生成完整的服装关键点坐标集
计算关键点之间的相对比例：
肩宽 / 胸围比例
衣长 / 身高比例
袖长 / 臂长比例
领宽 / 肩宽比例
调用工具：YOLOv8-pose 8.1.0 + MediaPipe 0.10.9
输出：JSON 格式的关键点坐标集和相对比例参数
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
AI 层接口：POST /api/v1/ai/keypoints
步骤 4：服装结构特征提取与款式分类（AI 能力层）
输入：预处理后的照片 URL + 关键点坐标集
处理逻辑：
调用CLIP模型生成照片的特征向量
调用Milvus 2.3.3向量数据库，检索最相似的前 5 个款式模板
调用通义千问V3 API，传入照片和关键点信息，执行以下 Prompt：
plaintext
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
融合 CLIP 向量检索结果和通义千问识别结果，取置信度最高的结果
调用工具：CLIP + Milvus 2.3.3 + 通义千问 V3 API
输出：JSON 格式的服装结构参数
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
AI 层接口：POST /api/v1/ai/structure
步骤 5：生成 GarmentCode 全量参数集并预填前端表单（业务服务层）
输入：服装结构参数 + 相对比例参数
处理逻辑：
调用GarmentCode参数映射服务，将 AI 识别的参数转换为 GarmentCode 可识别的标准参数
从PostgreSQL的templates表中加载对应款式的默认参数模板
将 AI 识别的结构参数和相对比例参数填充到模板中
生成完整的 GarmentCode 参数集，包含以下三类参数：
基础尺寸参数（默认值，必须由打版师手动修改确认）
结构组件参数（AI 自动预填，可手动修改）
工艺细节参数（系统默认值，可手动修改）
将参数集返回给前端，自动填充到动态表单中
调用工具：Go 1.22 + PostgreSQL 15.5
输出：JSON 格式的 GarmentCode 全量参数集（完整示例见附录 A）
业务服务接口：POST /api/v1/params/generate
步骤 6：打版师手动修改参数并点击 "更新 3D 预览"（前端交互层）
输入：打版师修改后的参数
处理逻辑：
前端显示三级分类的动态表单：
第一级：基础尺寸（衣长、胸围、肩宽、领围、袖长、袖口围、腰围、臀围）
第二级：结构组件（领型、袖型、门襟、口袋、省道）
第三级：工艺细节（缝份宽度、剪口类型、布纹线方向、放码规则）
每个参数显示默认值、取值范围和单位（cm）
支持参数联动：修改胸围自动同步调整袖窿宽，修改衣长自动调整侧缝长度
打版师修改完成后，点击 "更新 3D 预览" 按钮
前端将最新参数提交给 3D 模型调度服务
调用工具：React 18 + TypeScript + Tailwind CSS
输出：完整的 GarmentCode 参数集
前端接口：POST /api/v1/3d/generate
步骤 7：调用 GarmentCode 生成 3D 模型（双引擎核心层）
输入：完整的 GarmentCode 参数集
处理逻辑：
3D 模型调度服务将参数传递给 GarmentCode 引擎
调用pygarment.Garment()构造函数，传入参数
调用garment.generate_pattern()生成 2D 裁片
调用pygarment.meshgen.GarmentMeshGenerator()生成基础 3D 几何模型（MVP 阶段禁用物理模拟）
调用base_mesh.export()将 3D 模型保存为 OBJ 格式
调用trimesh.load()读取 OBJ 文件
调用mesh.export()将 OBJ 转换为 GLB 格式（Google ModelViewer 原生支持）
调用工具：Python 3.10 + pygarment 0.3.0 + trimesh 4.4.0
输出：GLB 格式的 3D 模型文件，文件大小≤1MB，顶点数≤2000
双引擎接口：POST /api/v1/engine/garmentcode/3d
步骤 8：3D 模型上传到 MinIO 并返回 URL（业务服务层）
输入：GLB 格式的 3D 模型文件
处理逻辑：
文件服务将 GLB 文件上传到 MinIO 的garment-3d-models存储桶
生成文件的公开访问 URL（生产环境使用预签名 URL）
将 URL 和参数的哈希值存入 Redis 缓存，缓存时间 1 小时
将 URL 返回给前端
调用工具：Go 1.22 + MinIO + Redis 7.2.4
输出：GLB 文件的可访问 URL
业务服务接口：POST /api/v1/files/upload/3d
步骤 9：Google ModelViewer 加载并展示 3D 模型（前端交互层）
输入：GLB 文件 URL
处理逻辑：
前端获取 GLB URL 后，将其赋值给<model-viewer>组件的src属性
ModelViewer 自动加载并渲染 3D 模型
显示加载动画，加载完成后自动旋转展示
支持以下交互操作：
鼠标拖拽：360° 旋转模型
滚轮：缩放模型
鼠标右键拖拽：平移模型
双击：重置视角
支持切换布局模式：左右分屏、上下分屏、全屏
调用工具：Google ModelViewer 3.5.0（通过 CDN 引入）
前端代码：
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
输出：前端右侧实时显示 3D 服装模型
步骤 10：打版师确认版型并点击 "生成工业版型"（前端交互层）
输入：打版师最终确认的参数
处理逻辑：
打版师旋转、缩放 3D 模型，检查版型是否符合预期
如有问题，重复步骤 6-9，继续微调参数
确认无误后，点击 "生成工业版型" 按钮
前端将最终参数提交给版型服务
调用工具：React 18 + TypeScript
输出：最终确认的 GarmentCode 参数集
前端接口：POST /api/v1/pattern/generate
步骤 11：双引擎生成工业级 DXF 文件并导出（双引擎核心层）
输入：最终确认的 GarmentCode 参数集
处理逻辑：
版型服务调用 GarmentCode 引擎，生成高精度 2D 裁片
python
运行
from pygarment import Garment
garment = Garment(final_params)
pattern = garment.generate_pattern()
pattern_json = pattern.to_json()
调用自研双向转换器，将 GarmentCode JSON 裁片转换为 Seamly2D XML 格式
转换精度控制在 ±0.01cm
保留所有裁片的形状、尺寸、位置、布纹线信息
将 Seamly2D XML 文件保存到临时目录
调用seamly2d-cli无头模式，执行以下命令添加工艺和放码：
bash
运行
seamly2d-cli --input pattern.xml --output pattern.dxf --add-seam-allowance 1cm --add-notches --add-drill-holes --grade --size-set "国标女160/84A,165/88A,170/92A" --format "ASTM DXF 2000"
Seamly2D 自动完成以下操作：
添加 1cm 标准缝份
添加剪口和钻眼
添加布纹线
按照国标号型进行全号型放码
输出标准 ASTM DXF 2000 格式文件
文件服务将 DXF 文件上传到 MinIO
前端显示下载链接，打版师点击下载 DXF 文件，直接给到工厂生产
调用工具：pygarment 0.3.0 + 自研双向转换器 + Seamly2D 2024.1.0（seamly2d-cli）
输出：标准 ASTM DXF 2000 格式的工业版型文件
业务服务接口：GET /api/v1/files/download/dxf/{file_id}
四、关键技术规范（AI 代码生成必须严格遵守）
4.1 接口规范
所有 REST 接口统一使用 JSON 格式
响应格式统一为：
json
{
  "code": 0, // 0成功，非0失败
  "message": "success",
  "data": {} // 响应数据
}
所有微服务之间的通信使用 gRPC，Protobuf 定义见附录 B
接口路径统一使用/api/v1/[服务名]/[功能]格式
4.2 命名规范
变量名：小驼峰（camelCase）
函数名：小驼峰（camelCase）
类名：大驼峰（PascalCase）
常量名：全大写 + 下划线（UPPER_SNAKE_CASE）
数据库表名：小写 + 下划线（snake_case）
接口路径：小写 + 连字符（kebab-case）
4.3 数据格式规范
所有尺寸单位统一为厘米（cm），保留两位小数
日期时间统一使用 ISO 8601 格式：2024-03-20T12:00:00Z
JSON 文件编码统一为 UTF-8
DXF 文件版本统一为 ASTM DXF 2000
4.4 错误处理规范
所有错误必须返回明确的错误码和错误信息
前端必须显示友好的错误提示
后端必须记录详细的错误日志
3D 生成失败时，自动返回对应款式的默认 3D 模型
Seamly2D 调用失败时，返回错误信息并提示用户检查参数
附录 A：GarmentCode 完整参数集示例
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
  "pocket_type": "贴袋",
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
给 AI 编程工具的最终要求
按照本文档生成完整的项目代码，目录结构如下：
plaintext
garment-ai/
├── frontend/          # React前端
├── backend/           # Go微服务
├── ai/                # Python AI服务
├── engine/            # 双引擎核心
├── deploy/            # Docker和K8s配置
├── scripts/           # CI/CD脚本
└── docs/              # 文档
每个模块必须包含完整的依赖管理文件（package.json、go.mod、requirements.txt）
生成完整的 docker-compose.yml，可一键启动所有服务
生成完整的 K8s 部署配置文件
生成 GitHub Actions CI/CD 脚本
所有配置项必须抽离到配置文件中，支持环境变量注入
代码必须包含详细的注释，特别是关键业务逻辑和工具调用部分
生成 README.md 文件，包含详细的部署和运行说明

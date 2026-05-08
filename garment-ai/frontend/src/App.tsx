import { ChangeEvent, startTransition, useState } from 'react';

type ViewKey = 'front' | 'side';
type StepKey = 'upload' | 'params' | 'preview';
type QualityStatus = 'pending' | 'pass' | 'warn' | 'fail';
type Primitive = boolean | number | string;
type ParamMap = Record<string, Primitive>;
type GenericMap = Record<string, unknown>;

type ApiResponse<T> = {
  code: number;
  message: string;
  data: T;
};

type QualityReport = {
  status: QualityStatus;
  score: number;
  width?: number;
  height?: number;
  sizeKb?: number;
  issues: string[];
};

type SelectedPhoto = {
  view: ViewKey;
  fileName: string;
  contentType: string;
  previewUrl: string;
  base64Data: string;
  quality: QualityReport;
};

type UploadPhotoResult = {
  fileName: string;
  objectKey: string;
  url: string;
  view: ViewKey;
};

type UploadResponse = {
  bucket: string;
  sessionId: string;
  photos: UploadPhotoResult[];
};

type PreprocessResponse = {
  photoUrls: string[];
  processedPhotoUrls: string[];
};

type KeypointsResponse = {
  photoUrls: string[];
  keypoints: Record<string, number[]>;
  ratios: Record<string, number>;
};

type StructureResponse = {
  photoUrls: string[];
  keypoints: Record<string, number[]>;
  ratios: Record<string, number>;
  structure: GenericMap;
};

type ParamsResponse = {
  templateId: string;
  structure: GenericMap;
  ratios: Record<string, number>;
  params: ParamMap;
};

type PreviewResponse = {
  taskId: string;
  params: ParamMap;
  glbUrl: string;
  cacheHit?: boolean;
};

type PatternJsonResponse = {
  version?: string;
  units?: string;
  garmentType?: string;
  silhouette?: string;
  metadata?: {
    pieceCount?: number;
    notchType?: string;
    gradeRule?: string;
    seamAllowanceMm?: number;
  };
  pieces?: Array<{
    id: string;
    name: string;
    category: string;
    seamAllowanceMm?: number;
  }>;
};

type PatternResponse = {
  taskId: string;
  params: ParamMap;
  patternJson: PatternJsonResponse;
  fileId: string;
  dxfUrl: string;
};

type StageHighlightProps = {
  eyebrow: string;
  title: string;
  description: string;
};

type StepChipProps = {
  label: string;
  active: boolean;
  complete: boolean;
};

const FALLBACK_MODEL_URL = 'https://modelviewer.dev/shared-assets/models/Astronaut.glb';

const SELECT_OPTIONS: Record<string, string[]> = {
  garment_type: ['shirt', 'dress', 'jacket', 'vest'],
  silhouette: ['H型', 'A型', 'X型', '宽松'],
  collar_type: ['翻领', '立领', '圆领', 'V领'],
  sleeve_type: ['长袖', '短袖', '无袖'],
  sleeve_cuff_type: ['单扣', '双扣', '罗纹'],
  placket_type: ['单排扣', '拉链', '暗门襟'],
  pocket_type: ['贴袋', '挖袋', '无'],
  notch_type: ['标准剪口', '双剪口', '无'],
  grain_line: ['经向', '纬向', '斜纹'],
  grade_rule: ['国标女', '国标男', '童装'],
  dart_position: ['前腰', '侧腰', '胸省', '无'],
};

const VIEW_LABELS: Record<ViewKey, string> = {
  front: '正面照',
  side: '侧面照',
};

const emptyQualityReport = (): QualityReport => ({
  status: 'pending',
  score: 0,
  issues: [],
});

async function requestJson<T>(url: string, init: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  const payload = (await response.json()) as ApiResponse<T>;

  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || '请求失败');
  }

  return payload.data;
}

async function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ''));
    reader.onerror = () => reject(new Error('文件读取失败'));
    reader.readAsDataURL(file);
  });
}

async function loadImageMetrics(dataUrl: string): Promise<{ width: number; height: number }> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve({ width: image.width, height: image.height });
    image.onerror = () => reject(new Error('图片解析失败'));
    image.src = dataUrl;
  });
}

function analyzePhotoQuality(width: number, height: number, sizeKb: number): QualityReport {
  const issues: string[] = [];
  let penalty = 0;
  const aspectRatio = width / Math.max(height, 1);

  if (width < 960 || height < 960) {
    issues.push('建议使用不低于 960 x 960 的原始照片。');
    penalty += 20;
  }

  if (width < 640 || height < 640) {
    issues.push('分辨率过低，可能无法稳定提取关键点。');
    penalty += 25;
  }

  if (sizeKb < 150) {
    issues.push('文件体积较小，可能存在压缩痕迹。');
    penalty += 15;
  }

  if (aspectRatio < 0.6 || aspectRatio > 1.4) {
    issues.push('建议使用居中、完整半身构图。');
    penalty += 10;
  }

  const score = Math.max(10, 100 - penalty);
  const status: QualityStatus = width < 640 || height < 640 ? 'fail' : issues.length > 0 ? 'warn' : 'pass';

  return {
    status,
    score,
    width,
    height,
    sizeKb,
    issues,
  };
}

function parseInputValue(rawValue: string, currentValue: Primitive): Primitive {
  if (typeof currentValue === 'number') {
    const parsed = Number(rawValue);
    return Number.isNaN(parsed) ? currentValue : parsed;
  }

  if (typeof currentValue === 'boolean') {
    return rawValue === 'true';
  }

  return rawValue;
}

function applyParamLinkages(params: ParamMap, key: string): ParamMap {
  const next = { ...params };

  if (key === 'sleeve_type') {
    if (next.sleeve_type === '短袖') {
      next.sleeve_length = typeof next.sleeve_length === 'number' ? Math.min(next.sleeve_length, 28) : 28;
    }

    if (next.sleeve_type === '长袖') {
      next.sleeve_length = typeof next.sleeve_length === 'number' ? Math.max(next.sleeve_length, 55) : 55;
    }

    if (next.sleeve_type === '无袖') {
      next.sleeve_length = 0;
      next.sleeve_cuff_type = '无';
    }
  }

  if (key === 'pocket_count' && typeof next.pocket_count === 'number' && next.pocket_count === 0) {
    next.pocket_type = '无';
    next.pocket_width = 0;
    next.pocket_height = 0;
    next.pocket_position_x = 0;
    next.pocket_position_y = 0;
  }

  if (key === 'length' && typeof next.length === 'number') {
    const recommendedPlacket = Math.max(12, Math.round(next.length * 0.35));
    if (typeof next.placket_length === 'number') {
      next.placket_length = Math.min(next.placket_length, recommendedPlacket);
    }
  }

  if (key === 'shoulder' && typeof next.shoulder === 'number') {
    const recommendedChest = Math.round(next.shoulder * 2.25);
    if (typeof next.chest === 'number') {
      next.chest = Math.max(next.chest, recommendedChest);
    }
  }

  if (key === 'chest' && typeof next.chest === 'number') {
    if (typeof next.waist === 'number') {
      next.waist = Math.min(next.waist, Math.max(next.chest - 12, 0));
    }
    if (typeof next.hip === 'number') {
      next.hip = Math.max(next.hip, next.chest + 2);
    }
  }

  if (key === 'garment_type' && next.garment_type === 'vest') {
    next.sleeve_type = '无袖';
    next.sleeve_length = 0;
  }

  return next;
}

function createSessionId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `session-${crypto.randomUUID()}`;
  }

  return `session-${Date.now().toString(36)}`;
}

function StageHighlight({ eyebrow, title, description }: StageHighlightProps) {
  return (
    <div className="rounded-3xl border border-black/10 bg-white/80 p-5 shadow-sm backdrop-blur">
      <div className="text-[11px] uppercase tracking-[0.3em] text-accent">{eyebrow}</div>
      <div className="mt-3 text-lg font-medium text-ink">{title}</div>
      <p className="mt-2 text-sm leading-6 text-black/70">{description}</p>
    </div>
  );
}

function StepChip({ label, active, complete }: StepChipProps) {
  const stateClass = complete
    ? 'border-emerald-300 bg-emerald-50 text-emerald-700'
    : active
      ? 'border-accent/30 bg-white text-accent'
      : 'border-black/10 bg-black/5 text-black/45';

  return <div className={`rounded-full border px-4 py-2 text-xs tracking-[0.24em] ${stateClass}`}>{label}</div>;
}

function QualityBadge({ quality }: { quality: QualityReport }) {
  const className = {
    pending: 'bg-black/5 text-black/50',
    pass: 'bg-emerald-100 text-emerald-700',
    warn: 'bg-amber-100 text-amber-700',
    fail: 'bg-red-100 text-red-700',
  }[quality.status];

  const label = {
    pending: '待检测',
    pass: '质量通过',
    warn: '建议复拍',
    fail: '不建议上传',
  }[quality.status];

  return <span className={`rounded-full px-3 py-1 text-xs font-medium ${className}`}>{label}</span>;
}

export default function App() {
  const backendBaseUrl = import.meta.env.VITE_BACKEND_BASE_URL ?? 'http://localhost:8080';
  const aiBaseUrl = import.meta.env.VITE_AI_BASE_URL ?? 'http://localhost:8000';

  const [sessionId] = useState(() => createSessionId());
  const [step, setStep] = useState<StepKey>('upload');
  const [loading, setLoading] = useState(false);
  const [statusText, setStatusText] = useState('请选择正面照与侧面照，系统会先做本地质量检查。');
  const [errorText, setErrorText] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [patternError, setPatternError] = useState<string | null>(null);
  const [modelSource, setModelSource] = useState(FALLBACK_MODEL_URL);
  const [modelLabel, setModelLabel] = useState('默认回退模型');
  const [templateId] = useState('shirt-basic-v1');
  const [uploadedPhotos, setUploadedPhotos] = useState<UploadPhotoResult[]>([]);
  const [ratios, setRatios] = useState<Record<string, number>>({});
  const [structure, setStructure] = useState<GenericMap>({});
  const [editableParams, setEditableParams] = useState<ParamMap>({});
  const [previewTask, setPreviewTask] = useState<PreviewResponse | null>(null);
  const [patternTask, setPatternTask] = useState<PatternResponse | null>(null);
  const [photoSelections, setPhotoSelections] = useState<Record<ViewKey, SelectedPhoto | null>>({
    front: null,
    side: null,
  });

  const selectedPhotos = Object.values(photoSelections).filter((item): item is SelectedPhoto => Boolean(item));
  const canGenerateParams = selectedPhotos.length > 0 && selectedPhotos.every((item) => item.quality.status !== 'fail');
  const stepReady = {
    upload: selectedPhotos.length > 0,
    params: Object.keys(editableParams).length > 0,
    preview: Boolean(previewTask),
  };

  async function handlePhotoSelection(view: ViewKey, event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];

    if (!file) {
      return;
    }

    setErrorText(null);

    try {
      const dataUrl = await fileToDataUrl(file);
      const metrics = await loadImageMetrics(dataUrl);
      const quality = analyzePhotoQuality(metrics.width, metrics.height, Math.round(file.size / 1024));

      setPhotoSelections((current) => ({
        ...current,
        [view]: {
          view,
          fileName: file.name,
          contentType: file.type || 'image/jpeg',
          previewUrl: dataUrl,
          base64Data: dataUrl.split(',')[1] ?? '',
          quality,
        },
      }));

      setStatusText(`${VIEW_LABELS[view]}已加载，本地质检分数 ${quality.score}。`);
    } catch (error) {
      setErrorText(error instanceof Error ? error.message : '照片读取失败');
    } finally {
      event.target.value = '';
    }
  }

  async function handleGenerateParams() {
    if (!canGenerateParams) {
      setErrorText('请先选择至少一张质量合格的照片。');
      return;
    }

    setLoading(true);
    setErrorText(null);

    try {
      setStatusText('正在上传照片到后端文件服务...');
      const uploadPayload = {
        sessionId,
        photos: selectedPhotos.map((photo) => ({
          view: photo.view,
          fileName: photo.fileName,
          contentType: photo.contentType,
          base64Data: photo.base64Data,
        })),
      };
      const uploadData = await requestJson<UploadResponse>(`${backendBaseUrl}/api/v1/upload/photos`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(uploadPayload),
      });
      setUploadedPhotos(uploadData.photos);

      setStatusText('正在请求 AI 预处理...');
      const preprocessData = await requestJson<PreprocessResponse>(`${aiBaseUrl}/api/v1/ai/preprocess`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ photoUrls: uploadData.photos.map((photo) => photo.url) }),
      });

      setStatusText('正在请求关键点与比例参数...');
      const keypointData = await requestJson<KeypointsResponse>(`${aiBaseUrl}/api/v1/ai/keypoints`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ photoUrls: preprocessData.processedPhotoUrls }),
      });

      setStatusText('正在解析服装结构...');
      const structureData = await requestJson<StructureResponse>(`${aiBaseUrl}/api/v1/ai/structure`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          photoUrls: preprocessData.processedPhotoUrls,
          keypoints: keypointData.keypoints,
          ratios: keypointData.ratios,
        }),
      });

      setStatusText('正在生成参数模板...');
      const paramsData = await requestJson<ParamsResponse>(`${backendBaseUrl}/api/v1/params/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          templateId,
          structure: structureData.structure,
          ratios: keypointData.ratios,
        }),
      });

      setRatios(keypointData.ratios);
      setStructure(structureData.structure);
      setEditableParams(paramsData.params);
      setPreviewTask(null);
      setPatternTask(null);
      setPreviewError(null);
      setPatternError(null);
      setStatusText('参数预填完成，可以继续人工微调。');
      startTransition(() => setStep('params'));
    } catch (error) {
      setErrorText(error instanceof Error ? error.message : '参数生成失败');
    } finally {
      setLoading(false);
    }
  }

  function handleParamChange(key: string, rawValue: string) {
    setEditableParams((current) => {
      const currentValue = current[key];
      const nextValue = parseInputValue(rawValue, currentValue);
      const linked = applyParamLinkages({ ...current, [key]: nextValue }, key);
      return linked;
    });
  }

  async function handleGeneratePreview() {
    if (Object.keys(editableParams).length === 0) {
      setErrorText('请先生成并确认参数。');
      return;
    }

    setLoading(true);
    setErrorText(null);
    setPreviewError(null);

    try {
      setStatusText('正在请求 3D 预览模型...');
      const previewData = await requestJson<PreviewResponse>(`${backendBaseUrl}/api/v1/3d/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ params: editableParams }),
      });

      setPreviewTask(previewData);
      setModelSource(previewData.glbUrl);
      setModelLabel(previewData.cacheHit ? '已命中缓存预览' : '实时生成预览');
      setStatusText('3D 预览已更新，可以继续调整参数后重新生成。');
      startTransition(() => setStep('preview'));
    } catch (error) {
      setPreviewError('3D 生成失败，已回退到默认展示模型。');
      setModelSource(FALLBACK_MODEL_URL);
      setModelLabel('默认回退模型');
      setErrorText(error instanceof Error ? error.message : '3D 预览生成失败');
    } finally {
      setLoading(false);
    }
  }

  async function handleGeneratePattern() {
    if (Object.keys(editableParams).length === 0) {
      setErrorText('请先生成并确认参数。');
      return;
    }

    setLoading(true);
    setErrorText(null);
    setPatternError(null);

    try {
      setStatusText('正在生成工业版型与 DXF...');
      const patternData = await requestJson<PatternResponse>(`${backendBaseUrl}/api/v1/pattern/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ params: editableParams }),
      });

      setPatternTask(patternData);
      setStatusText('工业版型已生成，可以直接下载 DXF 文件。');
    } catch (error) {
      setPatternError('DXF 导出失败，请稍后重试。');
      setErrorText(error instanceof Error ? error.message : '工业版型导出失败');
    } finally {
      setLoading(false);
    }
  }

  function renderParamField(key: string, value: Primitive) {
    if (typeof value === 'boolean') {
      return (
        <select
          className="w-full rounded-2xl border border-black/10 bg-white px-3 py-3 text-sm text-black/75 outline-none transition focus:border-accent/40"
          value={String(value)}
          onChange={(event) => handleParamChange(key, event.target.value)}
        >
          <option value="true">是</option>
          <option value="false">否</option>
        </select>
      );
    }

    if (SELECT_OPTIONS[key]) {
      return (
        <select
          className="w-full rounded-2xl border border-black/10 bg-white px-3 py-3 text-sm text-black/75 outline-none transition focus:border-accent/40"
          value={String(value)}
          onChange={(event) => handleParamChange(key, event.target.value)}
        >
          {SELECT_OPTIONS[key].map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      );
    }

    if (typeof value === 'number') {
      return (
        <input
          className="w-full rounded-2xl border border-black/10 bg-white px-3 py-3 text-sm text-black/75 outline-none transition focus:border-accent/40"
          type="number"
          step="0.1"
          value={String(value)}
          onChange={(event) => handleParamChange(key, event.target.value)}
        />
      );
    }

    return (
      <input
        className="w-full rounded-2xl border border-black/10 bg-white px-3 py-3 text-sm text-black/75 outline-none transition focus:border-accent/40"
        type="text"
        value={String(value)}
        onChange={(event) => handleParamChange(key, event.target.value)}
      />
    );
  }

  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(217,166,121,0.22),_transparent_36%),linear-gradient(135deg,_#efe6d8_0%,_#f8f5ef_44%,_#e9ddd0_100%)] px-3 py-4 md:px-5 md:py-6 xl:px-6">
      <section className="w-full">
        <div className="min-h-[calc(100vh-2rem)] rounded-[2rem] border border-white/70 bg-white/55 p-4 shadow-2xl shadow-black/5 backdrop-blur md:p-6 xl:p-8">
          <div className="grid gap-8 xl:grid-cols-[minmax(0,1.18fr)_minmax(460px,0.82fr)] 2xl:grid-cols-[minmax(0,1.28fr)_minmax(520px,0.72fr)]">
            <div>
              <div className="inline-flex rounded-full border border-accent/20 bg-white/70 px-4 py-2 text-xs uppercase tracking-[0.35em] text-accent">
                Stage 5 Preview
              </div>
              <h1 className="mt-5 max-w-5xl text-4xl font-semibold leading-tight text-ink md:text-6xl xl:text-[4.4rem]">
                从服装照片到参数编辑与真实 3D 预览的主链路
              </h1>
              <p className="mt-5 max-w-4xl text-base leading-8 text-black/70 md:text-lg">
                当前版本已经串起本地选图、照片质量检查、真实 AI 预处理与结构识别、参数微调，以及阶段 5 的实时 GLB 预览生成。
              </p>

              <div className="mt-8 flex flex-wrap gap-3">
                <StepChip label="01 上传照片" active={step === 'upload'} complete={step !== 'upload' || stepReady.upload} />
                <StepChip label="02 参数预填" active={step === 'params'} complete={step !== 'upload' && stepReady.params} />
                <StepChip label="03 3D 预览" active={step === 'preview'} complete={stepReady.preview} />
              </div>

              <div className="mt-8 grid gap-4 md:grid-cols-3">
                <StageHighlight eyebrow="Local QA" title="本地照片质检" description="选图后立即检查分辨率、体积和构图，先过滤明显不适合提取关键点的照片。" />
                <StageHighlight eyebrow="AI Runtime" title="真实参数预填" description="已接入照片预处理、关键点比例计算与服装结构识别，参数页展示真实识别结果。" />
                <StageHighlight eyebrow="Stage 6" title="工业 DXF 导出" description="参数确认后可生成版型 JSON 与 DXF 文件，下载链路统一由后端代理输出。" />
              </div>

              <div className="mt-8 rounded-[2rem] border border-black/10 bg-[#fffaf4] p-5 shadow-sm">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div className="text-xs uppercase tracking-[0.28em] text-accent">工作台状态</div>
                    <div className="mt-2 text-lg font-medium text-ink">{statusText}</div>
                  </div>
                  <div className="rounded-2xl bg-black/5 px-3 py-2 font-mono text-xs text-black/60">{sessionId}</div>
                </div>

                {errorText ? <div className="mt-4 rounded-2xl bg-red-50 px-4 py-3 text-sm text-red-700">{errorText}</div> : null}

                <div className="mt-5 grid gap-5 md:grid-cols-2">
                  {(['front', 'side'] as ViewKey[]).map((view) => {
                    const selected = photoSelections[view];

                    return (
                      <div key={view} className="rounded-[1.5rem] border border-black/10 bg-white p-4">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="text-sm font-medium text-ink">{VIEW_LABELS[view]}</div>
                            <div className="mt-1 text-xs text-black/45">支持桌面本地文件，也兼容摄像头输入。</div>
                          </div>
                          <QualityBadge quality={selected?.quality ?? emptyQualityReport()} />
                        </div>

                        <label className="mt-4 flex h-52 cursor-pointer items-center justify-center rounded-[1.25rem] border border-dashed border-black/15 bg-[#f7f1e8] transition hover:border-accent/40 hover:bg-[#f8efe1]">
                          {selected ? (
                            <img alt={`${VIEW_LABELS[view]}预览`} className="h-full w-full rounded-[1.25rem] object-cover" src={selected.previewUrl} />
                          ) : (
                            <div className="px-6 text-center text-sm leading-7 text-black/50">
                              点击选择 {VIEW_LABELS[view]}
                              <div className="text-xs">推荐完整上半身、背景简洁、光照均匀</div>
                            </div>
                          )}
                          <input accept="image/*" capture="environment" className="hidden" type="file" onChange={(event) => handlePhotoSelection(view, event)} />
                        </label>

                        <div className="mt-4 rounded-2xl bg-black/5 px-3 py-3 text-xs text-black/60">
                          {selected ? (
                            <>
                              <div>
                                分辨率 {selected.quality.width} x {selected.quality.height} · 文件约 {selected.quality.sizeKb} KB · 分数 {selected.quality.score}
                              </div>
                              {selected.quality.issues.length > 0 ? (
                                <ul className="mt-2 space-y-1 text-[11px] leading-5 text-black/55">
                                  {selected.quality.issues.map((issue) => (
                                    <li key={issue}>{issue}</li>
                                  ))}
                                </ul>
                              ) : (
                                <div className="mt-2 text-[11px] text-emerald-700">本地质检通过，可进入参数预填。</div>
                              )}
                            </>
                          ) : (
                            '尚未选择图片。'
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>

                <div className="mt-5 flex flex-wrap gap-3">
                  <button
                    className="rounded-full bg-ink px-6 py-3 text-sm font-medium text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading || !canGenerateParams}
                    onClick={() => {
                      void handleGenerateParams();
                    }}
                    type="button"
                  >
                    {loading && step === 'upload' ? '处理中...' : '进入参数页'}
                  </button>
                  <button
                    className="rounded-full border border-black/10 bg-white px-6 py-3 text-sm font-medium text-black/70 transition hover:border-accent/30 hover:text-accent"
                    onClick={() => setStep('upload')}
                    type="button"
                  >
                    回到上传
                  </button>
                </div>
              </div>
            </div>

            <div className="space-y-6">
              <div className="overflow-hidden rounded-[2rem] border border-black/10 bg-[#fbf7f1] p-4 shadow-xl shadow-black/5">
                <div className="mb-4 flex flex-wrap items-center justify-between gap-3 px-2">
                  <div>
                    <div className="text-xs uppercase tracking-[0.3em] text-accent">3D Preview</div>
                    <div className="mt-1 text-sm text-black/60">Google ModelViewer 3.5.0 · {modelLabel}</div>
                  </div>
                  <button
                    className="rounded-full bg-accent px-5 py-2 text-sm font-medium text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading || Object.keys(editableParams).length === 0}
                    onClick={() => {
                      void handleGeneratePreview();
                    }}
                    type="button"
                  >
                    {loading && step !== 'upload' ? '生成中...' : '更新 3D 预览'}
                  </button>
                </div>
                <div className="relative h-[420px] rounded-[1.5rem] border border-dashed border-black/15 bg-white/70 p-3">
                  <model-viewer
                    alt="服装 3D 预览"
                    ar
                    auto-rotate
                    camera-controls
                    exposure="1.4"
                    shadow-intensity="1"
                    src={modelSource}
                    style={{ width: '100%', height: '100%', backgroundColor: '#f8f2ea', borderRadius: '1rem' }}
                    onError={() => {
                      setPreviewError('模型加载失败，当前展示默认回退模型。');
                      setModelSource(FALLBACK_MODEL_URL);
                      setModelLabel('默认回退模型');
                    }}
                  />
                  <div className="pointer-events-none absolute left-5 top-5 rounded-full bg-white/85 px-3 py-1 text-xs tracking-[0.24em] text-black/50">
                    {previewTask?.taskId ?? 'preview-pending'}
                  </div>
                </div>
                {previewError ? <div className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-700">{previewError}</div> : null}
                <div className="mt-4 rounded-2xl bg-black/5 px-4 py-3 text-sm leading-6 text-black/60">
                  {previewTask ? (
                    <>
                      当前 GLB 来源：<span className="font-mono text-xs">{previewTask.glbUrl}</span>
                    </>
                  ) : (
                    '参数尚未提交，当前展示默认回退模型。'
                  )}
                </div>
              </div>

              <div className="rounded-[2rem] border border-black/10 bg-white/75 p-5 shadow-sm">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div className="text-xs uppercase tracking-[0.28em] text-accent">参数编辑</div>
                    <div className="mt-2 text-lg font-medium text-ink">模板 {templateId}</div>
                  </div>
                  <button
                    className="rounded-full border border-black/10 bg-white px-5 py-2 text-sm text-black/70 transition hover:border-accent/30 hover:text-accent disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={!stepReady.params}
                    onClick={() => setStep('params')}
                    type="button"
                  >
                    查看参数页
                  </button>
                </div>

                {patternError ? <div className="mt-4 rounded-2xl bg-amber-50 px-4 py-3 text-sm text-amber-700">{patternError}</div> : null}

                <div className="mt-4 grid gap-3 md:grid-cols-2">
                  {Object.keys(structure).length > 0 ? (
                    Object.entries(structure).map(([key, value]) => (
                      <div key={key} className="rounded-2xl bg-[#f7f2ea] px-3 py-3 text-sm text-black/65">
                        <div className="text-[11px] uppercase tracking-[0.18em] text-black/40">{key}</div>
                        <div className="mt-1 font-medium text-ink">{String(value)}</div>
                      </div>
                    ))
                  ) : (
                    <div className="rounded-2xl bg-black/5 px-4 py-4 text-sm text-black/50">上传照片后，这里会展示结构识别结果。</div>
                  )}
                </div>

                <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  {Object.entries(editableParams).map(([key, value]) => (
                    <label key={key} className="block rounded-[1.5rem] border border-black/10 bg-[#fffdf9] p-4">
                      <div className="text-[11px] uppercase tracking-[0.18em] text-black/40">{key}</div>
                      <div className="mt-3">{renderParamField(key, value)}</div>
                    </label>
                  ))}
                </div>

                <div className="mt-5 flex flex-wrap items-center gap-3">
                  <button
                    className="rounded-full bg-accent px-6 py-3 text-sm font-medium text-white transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
                    disabled={loading || Object.keys(editableParams).length === 0}
                    onClick={() => {
                      void handleGeneratePattern();
                    }}
                    type="button"
                  >
                    {loading ? '处理中...' : '生成 DXF 版型'}
                  </button>
                  {patternTask ? (
                    <a
                      className="rounded-full border border-black/10 bg-white px-6 py-3 text-sm font-medium text-black/70 transition hover:border-accent/30 hover:text-accent"
                      href={patternTask.dxfUrl}
                      rel="noreferrer"
                      target="_blank"
                    >
                      下载 DXF
                    </a>
                  ) : null}
                </div>

                <div className="mt-5 rounded-[1.5rem] border border-black/10 bg-[#f7f2ea] p-4 text-sm text-black/65">
                  {patternTask ? (
                    <>
                      <div className="text-[11px] uppercase tracking-[0.18em] text-black/40">工业版型结果</div>
                      <div className="mt-2 text-base font-medium text-ink">文件 {patternTask.fileId.slice(0, 12)}... · {patternTask.patternJson.metadata?.pieceCount ?? patternTask.patternJson.pieces?.length ?? 0} 个裁片</div>
                      <div className="mt-2 leading-6">
                        缝份 {patternTask.patternJson.metadata?.seamAllowanceMm ?? '-'} mm · 剪口 {patternTask.patternJson.metadata?.notchType ?? '-'} · 放码 {patternTask.patternJson.metadata?.gradeRule ?? '-'}
                      </div>
                      <div className="mt-3 font-mono text-xs text-black/55 break-all">{patternTask.dxfUrl}</div>
                    </>
                  ) : (
                    '参数确认后，这里会展示阶段 6 的工业版型结果，并提供 DXF 下载入口。'
                  )}
                </div>
              </div>

              <div className="rounded-[2rem] border border-black/10 bg-[#1f2329] p-5 text-white shadow-sm">
                <div className="text-xs uppercase tracking-[0.28em] text-[#f0c38a]">AI 摘要</div>
                <div className="mt-4 grid gap-3 md:grid-cols-2">
                  {Object.keys(ratios).length > 0 ? (
                    Object.entries(ratios).map(([key, value]) => (
                      <div key={key} className="rounded-2xl bg-white/10 px-4 py-3 text-sm text-white/85">
                        <div className="text-[11px] uppercase tracking-[0.18em] text-white/45">{key}</div>
                        <div className="mt-1 text-lg font-medium">{value}</div>
                      </div>
                    ))
                  ) : (
                    <div className="rounded-2xl bg-white/10 px-4 py-4 text-sm text-white/65">参数预填完成后，这里会展示本次识别得到的人体比例数据。</div>
                  )}
                </div>
                <div className="mt-4 rounded-2xl bg-white/10 px-4 py-3 text-sm leading-6 text-white/70">
                  已上传 {uploadedPhotos.length} 张照片。当前页面展示的是阶段 6 主链路：本地质检、AI 参数预填、人工微调、真实 3D 预览与工业 DXF 导出。
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}
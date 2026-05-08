from datetime import datetime, timezone

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.config import load_config
from app.pipeline import PreviewPipeline
from app.schemas import ApiResponse, Generate3DRequest

config = load_config()
pipeline = PreviewPipeline(config)
app = FastAPI(title="Garment Engine Service", version="0.1.0")
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:3000", "http://127.0.0.1:3000"],
    allow_credentials=False,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/healthz")
def healthz() -> dict:
    return {
        "code": 0,
        "message": "success",
        "data": {
            "service": "engine",
            "status": "ok",
            "environment": config.app_env,
            "timestamp": datetime.now(timezone.utc).isoformat(),
        },
    }


@app.get("/api/v1/engine/health")
def engine_health() -> dict:
    return {
        "code": 0,
        "message": "success",
        "data": {
            "minioEndpoint": config.minio_endpoint,
            "minioPublicEndpoint": config.minio_public_endpoint,
            "minioBucket": config.minio_bucket,
        },
    }


@app.post("/api/v1/engine/garmentcode/3d", response_model=ApiResponse)
def generate_3d(request: Generate3DRequest) -> ApiResponse:
    try:
        return ApiResponse(data=pipeline.generate_preview(request.params))
    except ValueError as error:
        return ApiResponse(code=1001, message=str(error), data={})
    except Exception:
        return ApiResponse(code=2002, message="3d preview generation failed", data={})
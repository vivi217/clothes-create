from dataclasses import dataclass
import os


@dataclass(slots=True)
class Config:
    app_env: str
    port: int
    minio_endpoint: str
    minio_public_endpoint: str
    minio_access_key: str
    minio_secret_key: str
    minio_bucket: str


def load_config() -> Config:
    return Config(
        app_env=os.getenv("APP_ENV", "development"),
        port=int(os.getenv("PORT", "8010")),
        minio_endpoint=os.getenv("MINIO_ENDPOINT", "localhost:9000"),
        minio_public_endpoint=os.getenv("MINIO_PUBLIC_ENDPOINT", "localhost:9000"),
        minio_access_key=os.getenv("MINIO_ACCESS_KEY", "minioadmin"),
        minio_secret_key=os.getenv("MINIO_SECRET_KEY", "minioadmin123"),
        minio_bucket=os.getenv("MINIO_BUCKET", "garment-3d-models"),
    )
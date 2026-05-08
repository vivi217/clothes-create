#!/usr/bin/env python3

import os
import sys


def fail(message: str) -> int:
    print(f"[ERROR] {message}", file=sys.stderr)
    return 1


def main() -> int:
    try:
        from minio import Minio
    except Exception as exc:
        return fail(f"Unable to import minio SDK: {exc}")

    endpoint = os.getenv("MINIO_ENDPOINT")
    access_key = os.getenv("MINIO_ACCESS_KEY")
    secret_key = os.getenv("MINIO_SECRET_KEY")
    secure = os.getenv("MINIO_SECURE", "false").lower() == "true"
    bucket = os.getenv("MINIO_BUCKET", "garment-3d-models")

    print("[OK] MinIO SDK imported successfully.")

    if not endpoint or not access_key or not secret_key:
      print("[WARN] MinIO environment variables are incomplete. Import check passed, connection check skipped.")
      return 0

    try:
        client = Minio(
            endpoint,
            access_key=access_key,
            secret_key=secret_key,
            secure=secure,
        )
        exists = client.bucket_exists(bucket)
    except Exception as exc:
        return fail(f"Unable to connect to MinIO: {exc}")

    print(f"[OK] Connected to MinIO endpoint: {endpoint}")
    print(f"[OK] Bucket '{bucket}' exists: {exists}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
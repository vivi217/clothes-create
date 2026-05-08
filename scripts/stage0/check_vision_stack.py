#!/usr/bin/env python3

import importlib
import sys


REQUIRED_MODULES = {
    "cv2": "OpenCV",
    "ultralytics": "YOLOv8 / ultralytics",
    "mediapipe": "MediaPipe",
    "pymilvus": "Milvus Python SDK",
}

OPTIONAL_MODULES = {
    "clip": "OpenAI CLIP",
    "open_clip": "OpenCLIP fallback",
}


def fail(message: str) -> int:
    print(f"[ERROR] {message}", file=sys.stderr)
    return 1


def import_module(module_name: str):
    return importlib.import_module(module_name)


def main() -> int:
    failed = False

    for module_name, label in REQUIRED_MODULES.items():
        try:
            module = import_module(module_name)
            version = getattr(module, "__version__", "unknown")
            print(f"[OK] {label} imported. module={module_name} version={version}")
        except Exception as exc:
            print(f"[ERROR] {label} import failed. module={module_name} error={exc}", file=sys.stderr)
            failed = True

    clip_loaded = False
    for module_name, label in OPTIONAL_MODULES.items():
        try:
            module = import_module(module_name)
            version = getattr(module, "__version__", "unknown")
            print(f"[OK] {label} imported. module={module_name} version={version}")
            clip_loaded = True
            break
        except Exception:
            continue

    if not clip_loaded:
        print("[WARN] Neither clip nor open_clip is installed.")

    if failed:
        return fail("Vision stack validation failed.")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
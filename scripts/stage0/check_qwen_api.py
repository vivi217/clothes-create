#!/usr/bin/env python3

import json
import os
import sys
from urllib import error, request


def fail(message: str) -> int:
    print(f"[ERROR] {message}", file=sys.stderr)
    return 1


def main() -> int:
    api_key = os.getenv("DASHSCOPE_API_KEY")
    if not api_key:
        print("[WARN] DASHSCOPE_API_KEY is not set. API check skipped.")
        return 0

    base_url = os.getenv(
        "QWEN_BASE_URL",
        "https://dashscope.aliyuncs.com/compatible-mode/v1",
    ).rstrip("/")
    model = os.getenv("QWEN_MODEL", "qwen-max")
    url = f"{base_url}/chat/completions"

    payload = {
        "model": model,
        "messages": [
            {"role": "system", "content": "You are a validation assistant."},
            {"role": "user", "content": "Reply with JSON only: {\"status\":\"ok\"}"},
        ],
        "temperature": 0,
    }

    data = json.dumps(payload).encode("utf-8")
    req = request.Request(
        url,
        data=data,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        method="POST",
    )

    try:
        with request.urlopen(req, timeout=20) as response:
            body = response.read().decode("utf-8")
            print(f"[OK] Qwen API responded with HTTP {response.status}")
            print(body[:800])
            return 0
    except error.HTTPError as exc:
        details = exc.read().decode("utf-8", errors="replace")
        return fail(f"Qwen API returned HTTP {exc.code}: {details}")
    except Exception as exc:
        return fail(f"Unable to reach Qwen API: {exc}")


if __name__ == "__main__":
    raise SystemExit(main())
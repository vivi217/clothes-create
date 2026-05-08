#!/usr/bin/env python3

import importlib
import sys


def fail(message: str) -> int:
    print(f"[ERROR] {message}", file=sys.stderr)
    return 1


def main() -> int:
    try:
        pygarment = importlib.import_module("pygarment")
    except Exception as exc:
        return fail(f"Unable to import pygarment: {exc}")

    garment_type = getattr(pygarment, "Garment", None)
    if garment_type is None:
        return fail("pygarment.Garment is missing.")

    try:
        meshgen = importlib.import_module("pygarment.meshgen")
    except Exception as exc:
        return fail(f"Unable to import pygarment.meshgen: {exc}")

    mesh_generator = getattr(meshgen, "GarmentMeshGenerator", None)
    if mesh_generator is None:
        return fail("pygarment.meshgen.GarmentMeshGenerator is missing.")

    version = getattr(pygarment, "__version__", "unknown")
    print(f"[OK] pygarment imported successfully. version={version}")
    print("[OK] Found pygarment.Garment")
    print("[OK] Found pygarment.meshgen.GarmentMeshGenerator")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
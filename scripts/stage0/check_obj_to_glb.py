#!/usr/bin/env python3

import sys
import tempfile
from pathlib import Path


def fail(message: str) -> int:
    print(f"[ERROR] {message}", file=sys.stderr)
    return 1


def main() -> int:
    try:
        import trimesh
    except Exception as exc:
        return fail(f"Unable to import trimesh: {exc}")

    try:
        mesh = trimesh.Trimesh(
            vertices=[
                [0.0, 0.0, 0.0],
                [1.0, 0.0, 0.0],
                [0.0, 1.0, 0.0],
            ],
            faces=[[0, 1, 2]],
            process=False,
        )
    except Exception as exc:
        return fail(f"Unable to create sample trimesh mesh: {exc}")

    with tempfile.TemporaryDirectory(prefix="garment-ai-stage0-") as temp_dir:
        temp_path = Path(temp_dir)
        obj_path = temp_path / "sample.obj"
        glb_path = temp_path / "sample.glb"

        try:
            mesh.export(obj_path)
            mesh.export(glb_path)
        except Exception as exc:
            return fail(f"Unable to export OBJ/GLB: {exc}")

        if not obj_path.exists() or obj_path.stat().st_size == 0:
            return fail("OBJ export did not create a valid file.")
        if not glb_path.exists() or glb_path.stat().st_size == 0:
            return fail("GLB export did not create a valid file.")

        print(f"[OK] OBJ export succeeded: {obj_path}")
        print(f"[OK] GLB export succeeded: {glb_path}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
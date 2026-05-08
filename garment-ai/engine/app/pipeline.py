from __future__ import annotations

import hashlib
import io
import json
import math
from datetime import timedelta
from typing import Any
from urllib.parse import urlparse, urlunparse

import trimesh
from minio import Minio
from minio.error import S3Error

from app.config import Config


class PreviewPipeline:
    def __init__(self, config: Config):
        self.config = config
        self.storage = Minio(
            config.minio_endpoint,
            access_key=config.minio_access_key,
            secret_key=config.minio_secret_key,
            secure=False,
        )

    def generate_preview(self, params: dict[str, Any]) -> dict[str, Any]:
        normalized = self._normalize_params(params)
        digest = hashlib.sha256(json.dumps(normalized, sort_keys=True, ensure_ascii=False).encode("utf-8")).hexdigest()
        asset_key = f"previews/generated/{digest}.glb"
        metadata_key = f"previews/generated/{digest}.json"

        self._ensure_bucket()
        cached = self._read_cached_metadata(metadata_key)
        if cached is not None and self._object_exists(asset_key):
            return {
                "params": params,
                "glbUrl": self._presigned_public_url(asset_key),
                "vertexCount": int(cached["vertexCount"]),
                "fileSizeKb": int(cached["fileSizeKb"]),
                "cacheHit": True,
                "assetKey": asset_key,
            }

        mesh = self._build_garment_mesh(normalized)
        scene = trimesh.Scene()
        scene.add_geometry(mesh)
        glb_bytes = scene.export(file_type="glb")
        if isinstance(glb_bytes, str):
            glb_bytes = glb_bytes.encode("utf-8")

        self.storage.put_object(
            self.config.minio_bucket,
            asset_key,
            io.BytesIO(glb_bytes),
            len(glb_bytes),
            content_type="model/gltf-binary",
        )

        metadata = {
            "vertexCount": int(len(mesh.vertices)),
            "fileSizeKb": max(1, math.ceil(len(glb_bytes) / 1024)),
            "assetKey": asset_key,
        }
        metadata_bytes = json.dumps(metadata, ensure_ascii=False).encode("utf-8")
        self.storage.put_object(
            self.config.minio_bucket,
            metadata_key,
            io.BytesIO(metadata_bytes),
            len(metadata_bytes),
            content_type="application/json",
        )

        return {
            "params": params,
            "glbUrl": self._presigned_public_url(asset_key),
            "vertexCount": metadata["vertexCount"],
            "fileSizeKb": metadata["fileSizeKb"],
            "cacheHit": False,
            "assetKey": asset_key,
        }

    def _normalize_params(self, params: dict[str, Any]) -> dict[str, Any]:
        def number(name: str, default: float) -> float:
            value = params.get(name, default)
            try:
                return float(value)
            except (TypeError, ValueError):
                return default

        def text(name: str, default: str) -> str:
            value = params.get(name, default)
            return str(value) if value is not None else default

        normalized = {
            "garment_type": text("garment_type", "shirt"),
            "silhouette": text("silhouette", "H型"),
            "collar_type": text("collar_type", "翻领"),
            "sleeve_type": text("sleeve_type", "长袖"),
            "placket_type": text("placket_type", "单排扣"),
            "pocket_type": text("pocket_type", "贴袋"),
            "length": number("length", 65.0),
            "chest": number("chest", 96.0),
            "waist": number("waist", 84.0),
            "hip": number("hip", 98.0),
            "shoulder": number("shoulder", 42.0),
            "neck": number("neck", 38.0),
            "sleeve_length": number("sleeve_length", 60.0),
            "pocket_count": int(number("pocket_count", 2)),
            "pocket_width": number("pocket_width", 12.0),
            "pocket_height": number("pocket_height", 14.0),
            "pocket_position_x": number("pocket_position_x", 15.0),
            "pocket_position_y": number("pocket_position_y", 25.0),
        }
        return normalized

    def _build_garment_mesh(self, params: dict[str, Any]) -> trimesh.Trimesh:
        chest_width = max(params["chest"] / 180.0, 0.38)
        waist_width = max(params["waist"] / 190.0, 0.32)
        hip_width = max(params["hip"] / 180.0, 0.36)
        shoulder_width = max(params["shoulder"] / 55.0, chest_width * 0.92)
        length = max(params["length"] / 100.0, 0.55)
        depth = max(chest_width * 0.42, 0.18)

        if params["silhouette"] == "A型":
            hip_width *= 1.12
        elif params["silhouette"] == "X型":
            waist_width *= 0.82
            shoulder_width *= 1.04
        elif params["silhouette"] == "宽松":
            chest_width *= 1.1
            hip_width *= 1.06

        torso = self._torso_mesh(shoulder_width, chest_width, waist_width, hip_width, depth, length)
        components = [torso]

        sleeve_type = params["sleeve_type"]
        if sleeve_type != "无袖":
            components.extend(self._sleeve_meshes(shoulder_width, depth, params["sleeve_length"], sleeve_type))

        components.extend(self._collar_meshes(params["collar_type"], shoulder_width, depth, params["neck"]))
        components.extend(self._placket_meshes(params["placket_type"], chest_width, depth, length))
        components.extend(self._pocket_meshes(params))

        mesh = trimesh.util.concatenate(components)
        mesh.remove_duplicate_faces()
        mesh.remove_unreferenced_vertices()
        mesh.merge_vertices()
        return mesh

    def _torso_mesh(self, shoulder: float, chest: float, waist: float, hip: float, depth: float, length: float) -> trimesh.Trimesh:
        y_top = length / 2.0
        y_chest = y_top - length * 0.18
        y_waist = y_top - length * 0.56
        y_bottom = -length / 2.0
        half_depth = depth / 2.0
        rings = [
            (shoulder / 2.0, y_top),
            (chest / 2.0, y_chest),
            (waist / 2.0, y_waist),
            (hip / 2.0, y_bottom),
        ]
        vertices = []
        for half_width, y_pos in rings:
            vertices.extend(
                [
                    (-half_width, y_pos, half_depth),
                    (half_width, y_pos, half_depth),
                    (half_width, y_pos, -half_depth),
                    (-half_width, y_pos, -half_depth),
                ]
            )

        faces = []
        for ring_index in range(len(rings) - 1):
            start = ring_index * 4
            next_start = start + 4
            for edge in range(4):
                a = start + edge
                b = start + ((edge + 1) % 4)
                c = next_start + ((edge + 1) % 4)
                d = next_start + edge
                faces.extend([[a, b, c], [a, c, d]])

        faces.extend([[0, 1, 2], [0, 2, 3]])
        bottom = (len(rings) - 1) * 4
        faces.extend([[bottom, bottom + 2, bottom + 1], [bottom, bottom + 3, bottom + 2]])

        return trimesh.Trimesh(vertices=vertices, faces=faces, process=False)

    def _sleeve_meshes(self, shoulder_width: float, depth: float, sleeve_length: float, sleeve_type: str) -> list[trimesh.Trimesh]:
        if sleeve_type == "短袖":
            length = max(sleeve_length / 220.0, 0.18)
        else:
            length = max(sleeve_length / 120.0, 0.45)
        radius = max(depth * 0.34, 0.06)
        y_pos = length * 0.55 / 2.0
        x_pos = shoulder_width / 2.0 + radius * 0.65
        z_rot = math.radians(18)
        meshes = []
        for direction in (-1.0, 1.0):
            sleeve = trimesh.creation.cylinder(radius=radius, height=length, sections=24)
            sleeve.apply_transform(trimesh.transformations.rotation_matrix(math.pi / 2.0, [0, 0, 1]))
            sleeve.apply_transform(trimesh.transformations.rotation_matrix(direction * z_rot, [0, 0, 1]))
            sleeve.apply_translation([direction * x_pos, y_pos, 0])
            meshes.append(sleeve)
        return meshes

    def _collar_meshes(self, collar_type: str, shoulder_width: float, depth: float, neck: float) -> list[trimesh.Trimesh]:
        collar_width = max(neck / 180.0, shoulder_width * 0.18)
        collar_depth = max(depth * 0.6, 0.05)
        collar_height = 0.05 if collar_type == "立领" else 0.03
        if collar_type == "圆领":
            collar = trimesh.creation.annulus(r_min=max(collar_width * 0.18, 0.05), r_max=max(collar_width * 0.3, 0.08), height=0.02)
            collar.apply_transform(trimesh.transformations.rotation_matrix(math.pi / 2.0, [1, 0, 0]))
            collar.apply_translation([0, 0.3, 0])
        else:
            collar = trimesh.creation.box(extents=[collar_width, collar_height, collar_depth])
            collar.apply_translation([0, 0.31, 0])
        return [collar]

    def _placket_meshes(self, placket_type: str, chest_width: float, depth: float, length: float) -> list[trimesh.Trimesh]:
        if placket_type == "暗门襟":
            return []
        width = 0.03 if placket_type == "单排扣" else 0.04
        placket = trimesh.creation.box(extents=[width, length * 0.78, max(depth * 0.15, 0.015)])
        placket.apply_translation([0, 0.0, depth / 2.0 + 0.012])
        return [placket]

    def _pocket_meshes(self, params: dict[str, Any]) -> list[trimesh.Trimesh]:
        if params["pocket_type"] == "无" or params["pocket_count"] <= 0:
            return []
        pocket_width = max(params["pocket_width"] / 120.0, 0.08)
        pocket_height = max(params["pocket_height"] / 120.0, 0.1)
        x_offset = max(params["pocket_position_x"] / 120.0, 0.12)
        y_offset = params["pocket_position_y"] / 140.0
        pockets = []
        if params["pocket_count"] == 1:
            offsets = [0.0]
        else:
            offsets = [-x_offset, x_offset]
        for x_pos in offsets[: max(params["pocket_count"], 1)]:
            pocket = trimesh.creation.box(extents=[pocket_width, pocket_height, 0.018])
            pocket.apply_translation([x_pos, -y_offset, 0.13])
            pockets.append(pocket)
        return pockets

    def _ensure_bucket(self) -> None:
        if self.storage.bucket_exists(self.config.minio_bucket):
            return
        self.storage.make_bucket(self.config.minio_bucket)

    def _read_cached_metadata(self, object_key: str) -> dict[str, Any] | None:
        try:
            response = self.storage.get_object(self.config.minio_bucket, object_key)
        except S3Error:
            return None
        try:
            return json.loads(response.read().decode("utf-8"))
        finally:
            response.close()
            response.release_conn()

    def _object_exists(self, object_key: str) -> bool:
        try:
            self.storage.stat_object(self.config.minio_bucket, object_key)
            return True
        except S3Error:
            return False

    def _presigned_public_url(self, object_key: str) -> str:
        internal = self.storage.presigned_get_object(
            self.config.minio_bucket,
            object_key,
            expires=timedelta(minutes=30),
        )
        parsed = urlparse(internal)
        return urlunparse((parsed.scheme, self.config.minio_public_endpoint, parsed.path, parsed.params, parsed.query, parsed.fragment))
# Garment AI Stage 1 Skeleton

This directory contains the stage 1 project skeleton for the garment AI pattern-making system. The goal of this stage is to establish the module layout, base dependencies, configuration flow, container orchestration, and health endpoints without implementing the full business workflow yet.

## Included Modules

- `frontend`: React 18 + TypeScript 5.4 + Electron 29 desktop shell with a development web surface.
- `backend`: Go 1.22 + Gin service skeleton with configuration loading and health routes.
- `ai`: Python 3.11 + FastAPI service skeleton for future AI preprocessing and structure analysis.
- `engine`: Python 3.11 + FastAPI service skeleton for future GarmentCode and Seamly2D orchestration.
- `deploy`: container-related assets.
- `docs`: project notes for stage 1.

## Quick Start

1. Copy `.env.example` to `.env`.
2. Start the local stack:

```bash
make build
make up
```

3. Verify service health:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8000/healthz
curl http://localhost:8010/healthz
```

Common local commands:

```bash
make ps
make logs
make health
make down
```

For stage 1 compatibility, the local MinIO credentials are aligned to Milvus standalone defaults:

- access key: `minioadmin`
- secret key: `minioadmin`

## Stage 1 Scope

This stage intentionally includes only:

- project layout
- dependency manifests
- base config loading
- Dockerfiles
- docker-compose stack
- health endpoints
- CI workflow skeleton

The domain APIs described in the architecture document will be added in stage 2.

## Stage 2 Progress

The repository now includes a first stage 2 contract layer:

- backend mock REST routes for upload, parameter generation, preview generation, pattern generation, and DXF download
- AI mock REST routes for preprocess, keypoints, and structure analysis
- engine mock REST route for GarmentCode 3D generation
- protobuf source file and generation script
- API contract document for frontend and service integration
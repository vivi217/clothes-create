# Stage 2 API Contracts

This document freezes the stage 2 REST and gRPC contracts used for cross-service integration and frontend development. Current implementations are mock responses, but field names and route paths should remain stable.

## Unified Response Envelope

All REST responses use:

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

## Error Codes

- `0`: success
- `1001`: invalid request
- `1002`: upload failed
- `2001`: params generation failed
- `2002`: 3D preview generation failed
- `2003`: industrial pattern generation failed
- `3001`: file not found

## REST Endpoints

### POST /api/v1/upload/photos

Request:

```json
{
  "sessionId": "session-001",
  "photos": [
    {
      "view": "front",
      "fileName": "front.jpg",
      "contentType": "image/jpeg",
      "base64Data": "..."
    },
    {
      "view": "side",
      "fileName": "side.jpg",
      "contentType": "image/jpeg",
      "base64Data": "..."
    }
  ]
}
```

### POST /api/v1/ai/preprocess

Request:

```json
{
  "photoUrls": [
    "http://minio:9000/garment-3d-models/uploads/session-001/front.jpg",
    "http://minio:9000/garment-3d-models/uploads/session-001/side.jpg"
  ]
}
```

### POST /api/v1/ai/keypoints

Request:

```json
{
  "photoUrls": [
    "http://minio:9000/garment-3d-models/preprocessed/front.png",
    "http://minio:9000/garment-3d-models/preprocessed/side.png"
  ]
}
```

### POST /api/v1/ai/structure

Request:

```json
{
  "photoUrls": [
    "http://minio:9000/garment-3d-models/preprocessed/front.png"
  ],
  "keypoints": {
    "left_shoulder": [120.0, 160.0]
  },
  "ratios": {
    "shoulder_to_chest": 0.65
  }
}
```

### POST /api/v1/params/generate

Request:

```json
{
  "templateId": "shirt-basic-v1",
  "structure": {
    "garment_type": "shirt"
  },
  "ratios": {
    "shoulder_to_chest": 0.65
  }
}
```

### POST /api/v1/3d/generate

Request:

```json
{
  "params": {
    "garment_type": "shirt",
    "length": 65.0
  }
}
```

### POST /api/v1/pattern/generate

Request:

```json
{
  "params": {
    "garment_type": "shirt",
    "length": 65.0
  }
}
```

### GET /api/v1/files/download/dxf/{file_id}

Response `data` includes:

```json
{
  "fileId": "mock-dxf-001",
  "fileName": "mock-dxf-001.dxf",
  "format": "ASTM DXF 2000",
  "downloadUrl": "http://localhost:8080/api/v1/files/download/dxf/mock-dxf-001"
}
```

### POST /api/v1/engine/garmentcode/3d

Request:

```json
{
  "params": {
    "garment_type": "shirt",
    "length": 65.0
  }
}
```

## gRPC Contract

Source of truth: [backend/proto/garment.proto](../backend/proto/garment.proto)

Generation command:

```bash
bash scripts/generate-proto.sh
```
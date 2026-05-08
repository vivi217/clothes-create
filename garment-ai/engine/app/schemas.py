from pydantic import BaseModel, Field


class ApiResponse(BaseModel):
    code: int = 0
    message: str = "success"
    data: dict


class Generate3DRequest(BaseModel):
    params: dict = Field(default_factory=dict)


class Preview3DResponse(BaseModel):
    params: dict
    glbUrl: str
    vertexCount: int
    fileSizeKb: int
    cacheHit: bool = False
    assetKey: str
"""
Domain models/entities
"""
from dataclasses import dataclass
from typing import List, Optional
import numpy as np


@dataclass
class BoundingBox:
    """Face bounding box (normalized 0-1)"""
    x: float
    y: float
    width: float
    height: float


@dataclass
class DetectedFace:
    """Detected face with embedding"""
    bbox: BoundingBox
    embedding: List[float]  # 512-dimensional embedding
    confidence: float
    landmarks: Optional[List[List[float]]] = None  # 5-point landmarks

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization"""
        return {
            "bbox_x": self.bbox.x,
            "bbox_y": self.bbox.y,
            "bbox_width": self.bbox.width,
            "bbox_height": self.bbox.height,
            "embedding": self.embedding,
            "confidence": self.confidence,
        }


@dataclass
class ExtractionResult:
    """Result of face extraction"""
    success: bool
    faces: List[DetectedFace]
    error: Optional[str] = None
    processing_time_ms: int = 0

    def to_dict(self) -> dict:
        """Convert to dictionary for JSON serialization"""
        return {
            "success": self.success,
            "faces": [f.to_dict() for f in self.faces],
            "error": self.error,
            "processing_time_ms": self.processing_time_ms,
        }


@dataclass
class HealthStatus:
    """Service health status"""
    status: str
    model: str
    version: str

    def to_dict(self) -> dict:
        return {
            "status": self.status,
            "model": self.model,
            "version": self.version,
        }

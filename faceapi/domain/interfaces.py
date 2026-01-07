"""
Domain interfaces (ports)
"""
from abc import ABC, abstractmethod
from typing import Optional
import numpy as np

from .models import ExtractionResult, HealthStatus


class FaceDetectorInterface(ABC):
    """Interface for face detection service"""

    @abstractmethod
    def extract_faces(self, image: np.ndarray) -> ExtractionResult:
        """Extract faces from an image array"""
        pass

    @abstractmethod
    def get_health(self) -> HealthStatus:
        """Get service health status"""
        pass

    @abstractmethod
    def is_ready(self) -> bool:
        """Check if the detector is ready"""
        pass


class ImageLoaderInterface(ABC):
    """Interface for image loading"""

    @abstractmethod
    def load_from_url(self, url: str) -> Optional[np.ndarray]:
        """Load image from URL"""
        pass

    @abstractmethod
    def load_from_bytes(self, data: bytes) -> Optional[np.ndarray]:
        """Load image from bytes"""
        pass

"""
Face extraction service - application layer
"""
import logging

from domain.interfaces import FaceDetectorInterface, ImageLoaderInterface
from domain.models import ExtractionResult, HealthStatus

logger = logging.getLogger(__name__)


class FaceExtractionService:
    """Service for extracting faces from images"""

    def __init__(
        self,
        detector: FaceDetectorInterface,
        image_loader: ImageLoaderInterface,
    ):
        self.detector = detector
        self.image_loader = image_loader

    def extract_from_url(self, image_url: str) -> ExtractionResult:
        """Extract faces from an image URL"""
        # Load image
        image = self.image_loader.load_from_url(image_url)
        if image is None:
            return ExtractionResult(
                success=False,
                faces=[],
                error="Failed to load image from URL",
            )

        # Extract faces
        return self.detector.extract_faces(image)

    def extract_from_bytes(self, image_data: bytes) -> ExtractionResult:
        """Extract faces from image bytes"""
        # Load image
        image = self.image_loader.load_from_bytes(image_data)
        if image is None:
            return ExtractionResult(
                success=False,
                faces=[],
                error="Failed to decode image",
            )

        # Extract faces
        return self.detector.extract_faces(image)

    def get_health(self) -> HealthStatus:
        """Get service health status"""
        return self.detector.get_health()

    def is_ready(self) -> bool:
        """Check if service is ready"""
        return self.detector.is_ready()

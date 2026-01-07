"""
InsightFace implementation of face detector
"""
import time
import logging
from typing import List
import numpy as np

from insightface.app import FaceAnalysis

from domain.interfaces import FaceDetectorInterface
from domain.models import DetectedFace, BoundingBox, ExtractionResult, HealthStatus
from config import get_config

logger = logging.getLogger(__name__)


class InsightFaceDetector(FaceDetectorInterface):
    """Face detector using InsightFace library"""

    def __init__(self):
        self.config = get_config()
        self.model: FaceAnalysis = None
        self.is_initialized = False
        self.model_name = self.config.MODEL_NAME
        self._initialize()

    def _initialize(self):
        """Initialize the InsightFace model"""
        try:
            logger.info(f"Initializing InsightFace model: {self.model_name}")

            # Determine providers based on GPU setting
            if self.config.USE_GPU:
                providers = [
                    ('CUDAExecutionProvider', {'device_id': self.config.GPU_ID}),
                    'CPUExecutionProvider'
                ]
            else:
                providers = ['CPUExecutionProvider']

            # Initialize FaceAnalysis
            self.model = FaceAnalysis(
                name=self.model_name,
                providers=providers,
            )

            # Prepare with detection size
            self.model.prepare(
                ctx_id=self.config.GPU_ID if self.config.USE_GPU else -1,
                det_size=(self.config.DET_SIZE, self.config.DET_SIZE),
            )

            self.is_initialized = True
            logger.info(f"InsightFace model initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize InsightFace: {e}")
            self.is_initialized = False
            raise

    def extract_faces(self, image: np.ndarray) -> ExtractionResult:
        """Extract faces from an image"""
        if not self.is_initialized:
            return ExtractionResult(
                success=False,
                faces=[],
                error="Model not initialized",
            )

        start_time = time.time()

        try:
            # Get image dimensions for normalization
            height, width = image.shape[:2]

            # Detect faces
            faces = self.model.get(image, max_num=self.config.MAX_FACES)

            detected_faces: List[DetectedFace] = []

            for face in faces:
                # Get bounding box (x1, y1, x2, y2)
                bbox = face.bbox.astype(float)
                x1, y1, x2, y2 = bbox

                # Normalize bounding box to 0-1 range
                norm_x = max(0, x1) / width
                norm_y = max(0, y1) / height
                norm_width = min(x2 - x1, width - x1) / width
                norm_height = min(y2 - y1, height - y1) / height

                # Get embedding (512 dimensions)
                embedding = face.embedding.tolist() if face.embedding is not None else []

                # Get confidence score
                confidence = float(face.det_score) if hasattr(face, 'det_score') else 0.0

                # Filter by minimum confidence
                if confidence < self.config.MIN_CONFIDENCE:
                    continue

                detected_faces.append(DetectedFace(
                    bbox=BoundingBox(
                        x=norm_x,
                        y=norm_y,
                        width=norm_width,
                        height=norm_height,
                    ),
                    embedding=embedding,
                    confidence=confidence,
                ))

            processing_time = int((time.time() - start_time) * 1000)

            return ExtractionResult(
                success=True,
                faces=detected_faces,
                processing_time_ms=processing_time,
            )

        except Exception as e:
            logger.error(f"Face extraction failed: {e}")
            processing_time = int((time.time() - start_time) * 1000)
            return ExtractionResult(
                success=False,
                faces=[],
                error=str(e),
                processing_time_ms=processing_time,
            )

    def get_health(self) -> HealthStatus:
        """Get health status"""
        return HealthStatus(
            status="ok" if self.is_initialized else "error",
            model=self.model_name,
            version="0.7.3",
        )

    def is_ready(self) -> bool:
        """Check if detector is ready"""
        return self.is_initialized

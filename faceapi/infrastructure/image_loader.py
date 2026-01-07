"""
Image loader implementation
"""
import logging
from typing import Optional
from io import BytesIO

import numpy as np
import cv2
import requests
from PIL import Image

from domain.interfaces import ImageLoaderInterface
from config import get_config

logger = logging.getLogger(__name__)


class ImageLoader(ImageLoaderInterface):
    """Image loader for various sources"""

    def __init__(self):
        self.config = get_config()
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'FaceAPI/1.0'
        })

    def load_from_url(self, url: str) -> Optional[np.ndarray]:
        """Load image from URL"""
        try:
            logger.info(f"Loading image from URL: {url[:100]}...")

            response = self.session.get(
                url,
                timeout=30,
                stream=True,
            )
            response.raise_for_status()

            # Check content type
            content_type = response.headers.get('Content-Type', '')
            if not any(mt in content_type for mt in ['image/', 'octet-stream']):
                logger.warning(f"Unexpected content type: {content_type}")

            # Check size
            content_length = int(response.headers.get('Content-Length', 0))
            if content_length > self.config.MAX_IMAGE_SIZE:
                logger.error(f"Image too large: {content_length} bytes")
                return None

            # Read and decode image
            image_data = response.content
            return self._decode_image(image_data)

        except requests.RequestException as e:
            logger.error(f"Failed to download image: {e}")
            return None
        except Exception as e:
            logger.error(f"Failed to load image from URL: {e}")
            return None

    def load_from_bytes(self, data: bytes) -> Optional[np.ndarray]:
        """Load image from bytes"""
        try:
            if len(data) > self.config.MAX_IMAGE_SIZE:
                logger.error(f"Image too large: {len(data)} bytes")
                return None

            return self._decode_image(data)

        except Exception as e:
            logger.error(f"Failed to load image from bytes: {e}")
            return None

    def _decode_image(self, data: bytes) -> Optional[np.ndarray]:
        """Decode image bytes to numpy array"""
        try:
            # Try PIL first (better format support)
            pil_image = Image.open(BytesIO(data))

            # Convert to RGB if needed
            if pil_image.mode != 'RGB':
                pil_image = pil_image.convert('RGB')

            # Convert to numpy array (RGB format)
            image = np.array(pil_image)

            # Convert RGB to BGR for OpenCV/InsightFace
            image = cv2.cvtColor(image, cv2.COLOR_RGB2BGR)

            return image

        except Exception as e:
            logger.warning(f"PIL failed, trying OpenCV: {e}")

            # Fallback to OpenCV
            try:
                nparr = np.frombuffer(data, np.uint8)
                image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

                if image is None:
                    logger.error("OpenCV failed to decode image")
                    return None

                return image

            except Exception as e2:
                logger.error(f"OpenCV also failed: {e2}")
                return None

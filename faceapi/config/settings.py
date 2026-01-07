"""
Application configuration settings
"""
import os
from dotenv import load_dotenv

load_dotenv()


class Config:
    """Base configuration"""
    # Flask
    DEBUG = os.getenv("DEBUG", "false").lower() == "true"
    HOST = os.getenv("HOST", "0.0.0.0")
    PORT = int(os.getenv("PORT", 5000))

    # InsightFace model settings
    MODEL_NAME = os.getenv("INSIGHTFACE_MODEL", "buffalo_l")  # buffalo_l, buffalo_s, buffalo_sc
    DET_SIZE = int(os.getenv("DET_SIZE", 640))  # Detection size

    # Processing settings
    MAX_FACES = int(os.getenv("MAX_FACES", 50))  # Max faces to detect per image
    MIN_CONFIDENCE = float(os.getenv("MIN_CONFIDENCE", 0.5))  # Min detection confidence

    # Image settings
    MAX_IMAGE_SIZE = int(os.getenv("MAX_IMAGE_SIZE", 10 * 1024 * 1024))  # 10MB
    ALLOWED_MIME_TYPES = ["image/jpeg", "image/png", "image/webp", "image/gif"]

    # GPU settings
    USE_GPU = os.getenv("USE_GPU", "true").lower() == "true"
    GPU_ID = int(os.getenv("GPU_ID", 0))


class DevelopmentConfig(Config):
    """Development configuration"""
    DEBUG = True


class ProductionConfig(Config):
    """Production configuration"""
    DEBUG = False


def get_config():
    """Get configuration based on environment"""
    env = os.getenv("FLASK_ENV", "development")
    if env == "production":
        return ProductionConfig()
    return DevelopmentConfig()

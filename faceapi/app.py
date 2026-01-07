"""
Face API - InsightFace service
Main application entry point
"""
import logging
import sys

from flask import Flask
from flask_cors import CORS

from config import get_config
from infrastructure.insightface_detector import InsightFaceDetector
from infrastructure.image_loader import ImageLoader
from application.face_service import FaceExtractionService
from api.routes import api, init_routes

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
    ]
)
logger = logging.getLogger(__name__)


def create_app() -> Flask:
    """Application factory"""
    config = get_config()

    # Create Flask app
    app = Flask(__name__)
    app.config['DEBUG'] = config.DEBUG

    # Enable CORS
    CORS(app)

    # Initialize infrastructure
    logger.info("Initializing face detector...")
    try:
        detector = InsightFaceDetector()
        image_loader = ImageLoader()
    except Exception as e:
        logger.error(f"Failed to initialize detector: {e}")
        raise

    # Initialize application service
    face_service = FaceExtractionService(
        detector=detector,
        image_loader=image_loader,
    )

    # Initialize routes with service
    init_routes(face_service)

    # Register blueprint
    app.register_blueprint(api)

    logger.info("Application initialized successfully")
    return app


def main():
    """Main entry point"""
    config = get_config()

    logger.info(f"Starting Face API on {config.HOST}:{config.PORT}")
    logger.info(f"Model: {config.MODEL_NAME}")
    logger.info(f"GPU enabled: {config.USE_GPU}")

    app = create_app()
    app.run(
        host=config.HOST,
        port=config.PORT,
        debug=config.DEBUG,
        threaded=True,
    )


if __name__ == '__main__':
    main()

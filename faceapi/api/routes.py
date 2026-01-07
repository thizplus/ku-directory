"""
API routes/handlers
"""
import logging
from flask import Blueprint, request, jsonify

from application.face_service import FaceExtractionService

logger = logging.getLogger(__name__)

# Create blueprint
api = Blueprint('api', __name__)

# Service instance (injected)
face_service: FaceExtractionService = None


def init_routes(service: FaceExtractionService):
    """Initialize routes with service dependency"""
    global face_service
    face_service = service


@api.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    status = face_service.get_health()
    return jsonify(status.to_dict())


@api.route('/extract', methods=['POST'])
def extract_from_url():
    """Extract faces from an image URL"""
    if not face_service.is_ready():
        return jsonify({
            "success": False,
            "error": "Service not ready",
            "faces": [],
        }), 503

    # Get request data
    data = request.get_json()
    if not data or 'image_url' not in data:
        return jsonify({
            "success": False,
            "error": "Missing image_url in request body",
            "faces": [],
        }), 400

    image_url = data['image_url']

    # Extract faces
    result = face_service.extract_from_url(image_url)

    if not result.success:
        return jsonify(result.to_dict()), 500

    return jsonify(result.to_dict())


@api.route('/extract-bytes', methods=['POST'])
def extract_from_bytes():
    """Extract faces from image bytes"""
    if not face_service.is_ready():
        return jsonify({
            "success": False,
            "error": "Service not ready",
            "faces": [],
        }), 503

    # Get image data from request body
    image_data = request.get_data()
    if not image_data:
        return jsonify({
            "success": False,
            "error": "No image data in request body",
            "faces": [],
        }), 400

    # Extract faces
    result = face_service.extract_from_bytes(image_data)

    if not result.success:
        return jsonify(result.to_dict()), 500

    return jsonify(result.to_dict())


@api.route('/ready', methods=['GET'])
def ready():
    """Readiness check endpoint"""
    is_ready = face_service.is_ready()
    if is_ready:
        return jsonify({"ready": True})
    return jsonify({"ready": False}), 503

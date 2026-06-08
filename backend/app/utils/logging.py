import logging
import json

class JSONFormatter(logging.Formatter):
    def format(self, record):
        log_data = {
            "timestamp": self.formatTime(record, "%Y-%m-%dT%H:%M:%SZ"),
            "level": record.levelname,
            "message": record.getMessage(),
            "service": "fuchibol_backend",
            "module": record.module,
            "filename": record.filename,
            "lineno": record.lineno
        }
        
        # Merge extra fields
        if hasattr(record, "stream_id"):
            log_data["stream_id"] = record.stream_id
        if hasattr(record, "user_id"):
            log_data["user_id"] = record.user_id
            
        return json.dumps(log_data)

def setup_json_logging():
    logger = logging.getLogger("uvicorn")
    logger.setLevel(logging.INFO)
    
    # Reconfigure uvicorn logger handlers
    for handler in logger.handlers[:]:
        handler.setFormatter(JSONFormatter())
        
    # Reconfigure root logger
    root_logger = logging.getLogger()
    for handler in root_logger.handlers[:]:
        handler.setFormatter(JSONFormatter())

import os

class Settings:
    DATABASE_URL: str = os.getenv("DATABASE_URL", "postgresql://fuchibol_user:fuchibol_password@db:5432/fuchibol_db")
    
    @property
    def ASYNC_DATABASE_URL(self) -> str:
        # SQLAlchemy requires postgresql+asyncpg:// for async pg driver
        if self.DATABASE_URL.startswith("postgresql://"):
            return self.DATABASE_URL.replace("postgresql://", "postgresql+asyncpg://")
        return self.DATABASE_URL
    
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://redis:6379/0")
    SECRET_KEY: str = os.getenv("SECRET_KEY", "fuchibol_super_secret_key_change_me_in_prod")
    JWT_ALGORITHM: str = os.getenv("JWT_ALGORITHM", "HS256")
    ACCESS_TOKEN_EXPIRE_MINUTES: int = int(os.getenv("ACCESS_TOKEN_EXPIRE_MINUTES", str(60 * 24 * 7))) # 7 days
    
    HLS_BASE_URL: str = os.getenv("HLS_BASE_URL", "http://localhost/hls")
    VOD_OUTPUT_DIR: str = os.getenv("VOD_OUTPUT_DIR", "/app/srs_hls/vod")

settings = Settings()

FROM python:3.11-slim

WORKDIR /app

# Install dependencies including ffmpeg for share_engine
RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir pyTelegramBotAPI requests flask

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV DOCKER_MODE=true

# The source code will be mounted via volumes for development/persistence
# but we can copy it as well for a self-contained image if desired.
# For now, we follow the prompt's volume mapping.

CMD ["python", "main.py"]

# Image légère pour Python
FROM python:3.11-slim

# Définition du répertoire de travail
WORKDIR /app

# Installation des dépendances système nécessaires
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Copie et installation des requirements (optimisation du cache)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install --no-cache-dir pytest pytest-mock

# Copie de l'intégralité du code
COPY . .

# Par défaut, on lance le bot
CMD ["python", "pi_bot.py"]

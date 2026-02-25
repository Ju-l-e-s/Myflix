# --- Stage 1: Builder ---
FROM python:3.11-slim as builder

WORKDIR /build

# Installation des outils de compilation pour les dépendances potentielles
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Installation des dépendances dans un répertoire local
COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt
RUN pip install --no-cache-dir --prefix=/install pytest pytest-mock

# --- Stage 2: Runtime ---
FROM python:3.11-slim

WORKDIR /app

# On ne garde que ffmpeg qui est nécessaire à l'exécution
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

# Copie des dépendances installées depuis le builder
COPY --from=builder /install /usr/local

# Création d'un utilisateur non-root pour la sécurité
RUN useradd -m -u 1000 appuser && \
    chown -R appuser:appuser /app

# Copie du code source avec les bons droits
COPY --chown=appuser:appuser . .

# Passage à l'utilisateur non-root
USER appuser

# Par défaut, on lance le bot
CMD ["python", "pi_bot.py"]

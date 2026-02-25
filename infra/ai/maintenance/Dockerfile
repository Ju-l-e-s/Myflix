FROM python:3.11-slim

WORKDIR /app

# Installation des dépendances système si nécessaire (ex: curl pour healthcheck)
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copie des fichiers de dépendances
COPY requirements.txt .

# Installation des librairies Python (on filtre les libs standards qui pourraient être dans requirements.txt par erreur)
RUN pip install --no-cache-dir pyTelegramBotAPI requests flask aiohttp

# Copie de l'ensemble du code de maintenance
COPY . .

# Variables d'environnement par défaut
ENV DOCKER_MODE=true
ENV PYTHONUNBUFFERED=1

# Port pour Flask (statut du bot)
EXPOSE 5001

# Lancement de l'orchestrateur de maintenance
CMD ["python", "main.py"]

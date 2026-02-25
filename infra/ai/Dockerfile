FROM python:3.11-slim

WORKDIR /app

# Création d'un utilisateur non-root
RUN useradd -m -u 1000 appuser && \
    chown -R appuser:appuser /app

# Installation de la librairie Python
RUN pip install --no-cache-dir ollama

# Copie du code avec les bons droits
COPY --chown=appuser:appuser . .

USER appuser

# On lance le script au démarrage
CMD ["python", "main.py"]

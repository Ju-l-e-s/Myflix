FROM python:3.11-slim

WORKDIR /app

# Installation de la librairie Python pour parler à Ollama
RUN pip install --no-cache-dir ollama

# On lance le script au démarrage
CMD ["python", "main.py"]

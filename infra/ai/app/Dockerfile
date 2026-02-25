FROM python:3.11-slim

# Définition du répertoire de travail
WORKDIR /app

# Installation des dépendances (optimisée pour le cache Docker)
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copie du code source
COPY . .

# Documentation du port utilisé par le webhook
EXPOSE 5000

# Lancement du script (qui démarre uvicorn en interne)
CMD ["python", "main.py"]

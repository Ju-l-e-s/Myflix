import telebot, threading, logging
from flask import Flask
from config import *
from handlers_admin import register_admin_handlers
from handlers_media import register_media_handlers
from handlers_search import register_search_handlers

# --- CONFIGURATION PRINCIPALE ---
bot = telebot.TeleBot(TOKEN)

# --- FLASK SERVER ---
webhook_app = Flask(__name__)

@webhook_app.route('/')
def index():
    return "Bot Active"

def run_flask():
    try:
        # Port 5001 as required
        webhook_app.run(host='0.0.0.0', port=5001)
    except Exception as e:
        logging.error(f"Flask Error: {e}")

if __name__ == "__main__":
    # 1. Start Flask in background
    threading.Thread(target=run_flask, daemon=True).start()
    
    # 2. Register Modules
    register_admin_handlers(bot, is_authorized)
    register_media_handlers(bot, is_authorized)
    register_search_handlers(bot, is_authorized)
    
    print("🚀 Bot démarré en mode modulaire (Orchestrateur Clean)...")
    
    # 3. Start Polling
    bot.infinity_polling()

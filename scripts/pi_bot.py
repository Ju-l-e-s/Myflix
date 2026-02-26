import telebot
import subprocess
import os
import json
import requests
import logging
import re
import threading
import shutil
import pathlib
from telebot import types
from telebot.types import InlineKeyboardMarkup, InlineKeyboardButton
from flask import Flask
from datetime import timedelta
from guessit import guessit
from pyarr import RadarrAPI, SonarrAPI
from rapidfuzz import fuzz, process
import share_engine
import storage_skill
import media_manager

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)

# --- CONFIGURATION ---
TOKEN = os.getenv("TELEGRAM_TOKEN")
SUPER_ADMIN = 6721936515
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
USERS_FILE = os.path.join(BASE_DIR, "users.json")
CONTEXT_FILE = "/tmp/bot_context.json"

# Gestion réseau Docker intelligente
DOCKER_MODE = os.getenv("DOCKER_MODE", "false").lower() == "true"

def get_service_params(service_type):
    port = 7878 if service_type == "radarr" else 8989
    key = os.getenv(f"{service_type.upper()}_API_KEY")
    url = f"http://{service_type}:{port}" if DOCKER_MODE else f"http://localhost:{port}"
    return url, key

RADARR_URL, RADARR_KEY = get_service_params("radarr")
SONARR_URL, SONARR_KEY = get_service_params("sonarr")

# Clients PyArr
radarr_client = RadarrAPI(RADARR_URL, RADARR_KEY)
sonarr_client = SonarrAPI(SONARR_URL, SONARR_KEY)

# Compatibilité (pour show_media_list)
RADARR_CFG = {"url": RADARR_URL, "key": RADARR_KEY}
SONARR_CFG = {"url": SONARR_URL, "key": SONARR_KEY}

# Fallback intelligent
try: radarr_client.get_system_status()
except:
    if DOCKER_MODE:
        RADARR_URL = "http://host.docker.internal:7878"
        radarr_client = RadarrAPI(RADARR_URL, RADARR_KEY)
        RADARR_CFG["url"] = RADARR_URL

try: sonarr_client.get_system_status()
except:
    if DOCKER_MODE:
        SONARR_URL = "http://host.docker.internal:8989"
        sonarr_client = SonarrAPI(SONARR_URL, SONARR_KEY)
        SONARR_CFG["url"] = SONARR_URL

QBIT_HOST = "gluetun" if DOCKER_MODE else "localhost"
QBIT_URL = f"http://{QBIT_HOST}:8080"
GEMINI_KEY = os.getenv("GEMINI_KEY")

bot = telebot.TeleBot(TOKEN)

# --- UTILS & STORAGE ---
def load_allowed_users():
    try:
        with open(USERS_FILE, "r") as f: return json.load(f)["allowed"]
    except: return [SUPER_ADMIN]

def is_authorized(uid): return uid in load_allowed_users()

def save_users(users):
    with open(USERS_FILE, "w") as f: json.dump({"allowed": users}, f)

def get_storage_emoji(item):
    if not item.get("hasFile") and item.get("statistics", {}).get("episodeFileCount", 0) == 0: return "🔴"
    path = item.get("path", "")
    return "📚" if path and "/mnt/externe" in path else "🚀"

# --- SEARCH ENGINE (ID-FIRST SNIPER PIPELINE) ---
def stream_gpt_json(query):
    url = f"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key={GEMINI_KEY}"
    headers = {"Content-Type": "application/json"}
    prompt = f"""Media Expert. Analyze: '{query}'. 
    Search for the MOST LIKELY movie or series.
    Return ONLY raw JSON:
    {{
      "title_en": "Original title",
      "tmdb_id": 947, # Find this for movies
      "tvdb_id": null, # Find this for series
      "year": 1962,
      "type": "movie" or "series"
    }}"""
    payload = {
        "contents": [{ "parts": [{"text": prompt}] }],
        "generationConfig": { "temperature": 0.1, "response_mime_type": "application/json" }
    }
    try:
        res = requests.post(url, headers=headers, json=payload, timeout=10)
        res.raise_for_status()
        return res.json()['candidates'][0]['content']['parts'][0]['text']
    except Exception as e:
        logging.error(f"Erreur Gemini: {e}")
        return None

def smart_search(ai_data):
    """Pipeline de recherche Sniper : ID d'abord, Titre ensuite."""
    is_movie = ai_data['type'] == 'movie'
    client = radarr_client if is_movie else sonarr_client
    
    # --- STRATÉGIE 1 : RECHERCHE PAR ID (PRÉCISION 100%) ---
    search_term = None
    if is_movie and ai_data.get('tmdb_id'):
        search_term = f"tmdb:{ai_data['tmdb_id']}"
    elif not is_movie and ai_data.get('tvdb_id'):
        search_term = f"tvdb:{ai_data['tvdb_id']}"
    
    # --- STRATÉGIE 2 : FALLBACK PAR TITRE ---
    if not search_term:
        search_term = ai_data.get('title_en') or ai_data.get('title_fr', 'Unknown')

    try:
        raw_results = client.lookup_movie(search_term) if is_movie else client.lookup_series(search_term)
        
        if not raw_results: return [], ai_data['type']

        scored_results = []
        for res in raw_results:
            # Score de sécurité (Fuzzy matching + Année)
            text_score = fuzz.token_sort_ratio(search_term, res['title'])
            year_bonus = 50 if ai_data.get('year') == res.get('year') else -20
            scored_results.append((res, text_score + year_bonus))

        scored_results.sort(key=lambda x: x[1], reverse=True)
        return [r[0] for r in scored_results[:5]], ai_data['type']
    except Exception as e:
        logging.error(f"Erreur Sniper Search: {e}")
        return [], ai_data['type']

# --- BOT HANDLERS ---
@bot.message_handler(commands=["get"])
def handle_get(m):
    if not is_authorized(m.chat.id): return
    parts = m.text.split(" ", 1)
    if len(parts) > 1:
        process_get_request(m, parts[1])
    else:
        msg = bot.reply_to(m, "🎬 Que veux-tu ?")
        bot.register_next_step_handler(msg, lambda message: process_get_request(message, message.text))

@bot.message_handler(func=lambda m: True)
def handle_text(m):
    if not is_authorized(m.chat.id): return
    if m.text.startswith('/'): return
    query = m.text.lower()
    keywords = ["veux", "cherche", "regarder", "voir", "trouve", "get", "film", "serie"]
    if any(k in query for k in keywords) or len(m.text.split()) >= 2:
        process_get_request(m, m.text)

def process_get_request(m, query):
    if query.startswith('/'): return
    bot.send_chat_action(m.chat.id, "typing")
    
    # 1. Extraction ID-First via IA
    ai_res = stream_gpt_json(query)
    try:
        data = json.loads(re.search(r"\{.*\}", ai_res, re.DOTALL).group(0))
    except:
        guess = guessit(query)
        data = {"title_en": guess.get('title'), "year": guess.get('year'), "type": "movie"}

    # 2. Recherche Sniper
    results, final_type = smart_search(data)
    
    if not results:
        bot.send_message(m.chat.id, f"❌ Aucun résultat pour '{data.get('title_en', query)}'.")
        return

    api_cat = "movie" if final_type == "movie" else "series"
    text = f"🎯 **Sniper Results** ({api_cat})\n\n"
    for i, res in enumerate(results, 1):
        text += f"{i}. {res['title']} ({res.get('year', 'N/A')})\n"
    
    markup = InlineKeyboardMarkup()
    btns = [InlineKeyboardButton(str(i), callback_data=f"dl_sel:{api_cat}:{i-1}") for i in range(1, len(results) + 1)]
    markup.add(*btns)
    bot.send_message(m.chat.id, text, reply_markup=markup, parse_mode="Markdown")

# --- PAGED MEDIA UI ---
def show_media_list(chat_id, message_id, category, page=0, is_new=False):
    cfg = RADARR_CFG if category == "films" else SONARR_CFG
    endpoint = "movie" if category == "films" else "series"
    try:
        r = requests.get(f"{cfg['url']}/api/v3/{endpoint}", headers={"X-Api-Key": cfg["key"]}, timeout=5)
        data = r.json()
        owned = [f for f in data if (f.get("hasFile") or f.get("statistics", {}).get("episodeFileCount", 0) > 0)]
        owned.sort(key=lambda x: x.get("added", ""), reverse=True)
        
        total = len(owned)
        start, end = page * 10, (page + 1) * 10
        items = owned[start:end]
        if not items:
            bot.send_message(chat_id, "📂 Vide.")
            return
        
        text = f"{'🎬' if category == 'films' else '📺'} **{category.capitalize()}** ({total})\n\n"
        for i, item in enumerate(items, 1):
            text += f"{start + i}. {get_storage_emoji(item)} {item['title']} ({item.get('year', '')})\n"
        
        markup = InlineKeyboardMarkup(row_width=5)
        btns = [InlineKeyboardButton(str(start + i), callback_data=f"m_sel:{category}:{items[i-1]['id']}") for i in range(1, len(items) + 1)]
        markup.add(*btns)
        
        nav = []
        if page > 0: nav.append(InlineKeyboardButton("⬅️", callback_data=f"m_pag:{category}:{page-1}"))
        if end < total: nav.append(InlineKeyboardButton("➡️", callback_data=f"m_pag:{category}:{page+1}"))
        if nav: markup.row(*nav)
        markup.add(InlineKeyboardButton("🏠 Menu Principal", callback_data="adm:main"))
        
        if is_new: bot.send_message(chat_id, text, reply_markup=markup, parse_mode="Markdown")
        else: bot.edit_message_text(text, chat_id, message_id, reply_markup=markup, parse_mode="Markdown")
    except: bot.send_message(chat_id, "❌ Erreur API.")

@bot.message_handler(commands=["films"])
def films_cmd(m):
    if is_authorized(m.chat.id): show_media_list(m.chat.id, None, "films", 0, True)

@bot.message_handler(commands=["series"])
def series_cmd(m):
    if is_authorized(m.chat.id): show_media_list(m.chat.id, None, "series", 0, True)

@bot.message_handler(commands=["status"])
def status_command(m):
    if not is_authorized(m.chat.id): return
    try:
        msg = storage_skill.get_status()
        bot.send_message(m.chat.id, msg, parse_mode="Markdown")
    except: bot.reply_to(m, "❌ Erreur lecture disque.")

@bot.message_handler(commands=["queue"])
def queue_command(m):
    if is_authorized(m.chat.id): send_queue_status(m.chat.id)

def send_queue_status(chat_id, message_id=None):
    try:
        r = requests.get(f"{QBIT_URL}/api/v2/torrents/info", timeout=5)
        torrents = r.json()
        if not torrents:
            msg = "📥 **QUEUE DE TÉLÉCHARGEMENT**\n\n• Queue vide."
            if message_id: bot.edit_message_text(msg, chat_id, message_id, parse_mode="Markdown")
            else: bot.send_message(chat_id, msg, parse_mode="Markdown")
            return

        q_text = "📥 **QUEUE DE TÉLÉCHARGEMENT**\n\n"
        found = False
        for t in torrents:
            if t["state"] in ["downloading", "stalledDL", "metaDL", "queuedDL", "checkingDL", "pausedDL"]:
                found = True
                progress_val = round(t["progress"] * 100, 1)
                q_text += f"🎬 **{media_manager.clean_media_name(t['name'])}**\n"
                q_text += f"{media_manager.get_progress_bar(progress_val)}\n"
                q_text += f"📦 {media_manager.format_size(t['size'])}  •  🚀 {media_manager.format_speed(t['dlspeed'])}\n\n"

        if not found: q_text += "• Aucun téléchargement en cours."
        markup = InlineKeyboardMarkup().add(InlineKeyboardButton("🔄 Actualiser", callback_data="q_refresh"))
        if message_id: bot.edit_message_text(q_text, chat_id, message_id, reply_markup=markup, parse_mode="Markdown")
        else: bot.send_message(chat_id, q_text, reply_markup=markup, parse_mode="Markdown")
    except Exception as e: bot.send_message(chat_id, f"❌ Erreur qBit : {e}")

# --- CALLBACKS & ADMIN ---
@bot.callback_query_handler(func=lambda call: True)
def master_router(call):
    if not is_authorized(call.from_user.id): return
    d = call.data.split(":")
    if d[0] == "q_refresh": send_queue_status(call.message.chat.id, call.message.message_id)
    elif d[0] == "m_pag": show_media_list(call.message.chat.id, call.message.message_id, d[1], int(d[2]))
    elif d[0] == "dl_sel": bot.answer_callback_query(call.id, "📥 Ajouté à la queue.")
    elif d[0] == "adm" and d[1] == "close": bot.delete_message(call.message.chat.id, call.message.message_id)

# STARTUP
webhook_app = Flask(__name__)
threading.Thread(target=lambda: webhook_app.run(host="0.0.0.0", port=5001), daemon=True).start()
print("🚀 Bot démarré avec architecture Sniper (ID-First Search)...")
bot.infinity_polling()

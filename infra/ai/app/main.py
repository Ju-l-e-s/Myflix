import logging
import os
import json
import re
import aiohttp
from functools import wraps
from telegram import Update, InlineKeyboardButton, InlineKeyboardMarkup
from telegram.ext import ApplicationBuilder, ContextTypes, CommandHandler, MessageHandler, CallbackQueryHandler, filters

# --- CONFIGURATION ---
TELEGRAM_TOKEN = os.environ.get("TELEGRAM_TOKEN")
# Lecture de la liste des IDs autorisés (ex: "12345,67890")
ALLOWED_IDS_STR = os.environ.get("TELEGRAM_ALLOWED_USER_IDS", "")

def parse_allowed_ids(ids_str):
    ids = []
    for i in ids_str.split(","):
        stripped = i.strip()
        if stripped and stripped.isdigit():
            ids.append(int(stripped))
        elif stripped:
            logging.getLogger("Gatekeeper").error(f"ID invalide dans TELEGRAM_ALLOWED_USER_IDS : '{stripped}'")
    return ids

ALLOWED_USER_IDS = parse_allowed_ids(ALLOWED_IDS_STR)
logging.getLogger("Gatekeeper").info(f"Gatekeeper configuré avec les IDs : {ALLOWED_USER_IDS}")

OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY")
OPENAI_URL = "https://api.openai.com/v1/chat/completions"

RADARR_URL = "http://radarr:7878/api/v3"
RADARR_API_KEY = "75ed9e2e3922442c8c1e012e47bc0906"

SONARR_URL = "http://sonarr:8989/api/v3"
SONARR_API_KEY = "dfa7da413b48480097f6a399080be393"

logging.basicConfig(format='%(asctime)s - %(name)s - %(levelname)s - %(message)s', level=logging.INFO)
logger = logging.getLogger("Gatekeeper")

# --- GATEKEEPER (Décorateur de sécurité) ---
def restricted(func):
    @wraps(func)
    async def wrapped(update: Update, context: ContextTypes.DEFAULT_TYPE, *args, **kwargs):
        user_id = update.effective_user.id
        if user_id not in ALLOWED_USER_IDS:
            logger.warning(f"Tentative d'accès non autorisée de l'ID : {user_id}")
            if update.message:
                await update.message.reply_text("🛑 Accès refusé. Vous n'êtes pas autorisé à utiliser ce bot.")
            elif update.callback_query:
                await update.callback_query.answer("🛑 Accès refusé.", show_alert=True)
            return
        return await func(update, context, *args, **kwargs)
    return wrapped

# --- UTILS ---
def get_first_sentences(text, n=2):
    if not text: return "Pas de résumé disponible."
    sentences = re.split(r'(?<=[.!?])\s+', text)
    return " ".join(sentences[:n])

def extract_season(text):
    match = re.search(r'(?:saison|s|s0)\s*(\d+)', text, re.IGNORECASE)
    return int(match.group(1)) if match else None

# --- RECHERCHE ---
async def search_media(media_type, query):
    url = RADARR_URL if media_type == "movie" else SONARR_URL
    key = RADARR_API_KEY if media_type == "movie" else SONARR_API_KEY
    endpoint = "/movie/lookup" if media_type == "movie" else "/series/lookup"
    
    year_match = re.search(r'\b(19|20)\d{2}\b', query)
    target_year = int(year_match.group(0)) if year_match else None
    clean_query = query.replace(str(target_year), "").strip() if target_year else query

    search_url = f"{url}{endpoint}?term={clean_query}&apiKey={key}"
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get(search_url, timeout=15) as resp:
                results = await resp.json()
                if not results or not isinstance(results, list):
                    return []

                def score(item):
                    s = 0
                    if item['title'].lower() == clean_query.lower(): s += 100
                    if target_year and item.get('year') == target_year: s += 50
                    return s

                results.sort(key=score, reverse=True)
                return results
    except Exception as e:
        logger.error(f"Search error: {e}")
    return []

# --- AJOUT RÉEL ---
async def add_movie(item_data):
    url = f"{RADARR_URL}/movie?apiKey={RADARR_API_KEY}"
    payload = {
        "title": item_data['title'],
        "qualityProfileId": 5,
        "titleSlug": item_data['titleSlug'],
        "images": item_data['images'],
        "tmdbId": item_data["tmdbId"],
        "year": item_data["year"],
        "rootFolderPath": "/movies",
        "monitored": True,
        "addOptions": {"searchForMovie": True}
    }
    async with aiohttp.ClientSession() as session:
        async with session.post(url, json=payload) as resp:
            return resp.status in [200, 201]

async def add_series(item_data, target_season=None):
    url = f"{SONARR_URL}/series?apiKey={SONARR_API_KEY}"
    payload = {
        "title": item_data['title'],
        "qualityProfileId": 5,
        "titleSlug": item_data['titleSlug'],
        "images": item_data['images'],
        "tvdbId": item_data["tvdbId"],
        "year": item_data["year"],
        "rootFolderPath": "/tv",
        "monitored": True,
        "addOptions": {"searchForMissingEpisodes": True}
    }
    
    if target_season is not None:
        payload["seasons"] = [
            {"seasonNumber": s["seasonNumber"], "monitored": s["seasonNumber"] == target_season}
            for s in item_data.get("seasons", [])
        ]
    else:
        payload["seasons"] = [{"seasonNumber": s["seasonNumber"], "monitored": True} for s in item_data.get("seasons", [])]

    async with aiohttp.ClientSession() as session:
        async with session.post(url, json=payload) as resp:
            return resp.status in [200, 201]

# --- LOGIQUE IA ---
async def process_ai_logic(prompt):
    headers = {"Authorization": f"Bearer {OPENAI_API_KEY}", "Content-Type": "application/json"}
    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {"role": "system", "content": "Media assistant. Intent: movie|series|chat. JSON {type, query, reply?}. 'reply' only if chat."},
            {"role": "user", "content": prompt}
        ],
        "response_format": {"type": "json_object"},
        "max_tokens": 200,
        "temperature": 0
    }

    async with aiohttp.ClientSession() as session:
        async with session.post(OPENAI_URL, json=payload, headers=headers) as resp:
            data = await resp.json()
            return json.loads(data["choices"][0]["message"]["content"])

# --- TELEGRAM HANDLERS ---
@restricted
async def handle_message(update: Update, context: ContextTypes.DEFAULT_TYPE):
    text = update.message.text
    # Un seul appel IA pour tout gérer
    intent = await process_ai_logic(text)
    
    if intent["type"] in ["movie", "series"]:
        target_season = extract_season(text) if intent["type"] == "series" else None
        results = await search_media(intent["type"], intent["query"])
        # ... (reste de la logique identique)
        if not results:
            await update.message.reply_text(f"Désolé, je n'ai rien trouvé pour '{intent['query']}'.")
            return

        found = results[0]
        context.user_data['last_search'] = {"type": intent["type"], "item": found, "season": target_season}
        
        type_label = "le film" if intent["type"] == "movie" else "la série"
        season_label = f" (Saison {target_season})" if target_season else ""
        overview = get_first_sentences(found.get('overview', ''))
        
        caption = f"J'ai trouvé {type_label} :\n\n* {found['title']}* ({found['year']}){season_label}\n_{overview}_\n\nEst-ce bien ce que vous voulez ajouter ?"
        
        keyboard = [
            [InlineKeyboardButton("✅ Oui, ajouter", callback_data='confirm_add')],
            [InlineKeyboardButton("❌ Non, annuler", callback_data='cancel_add')]
        ]
        reply_markup = InlineKeyboardMarkup(keyboard)
        
        poster = None
        for img in found.get('images', []):
            if img['coverType'] == 'poster':
                poster = img['remoteUrl']
                break
        
        if poster:
            await update.message.reply_photo(photo=poster, caption=caption, parse_mode='Markdown', reply_markup=reply_markup)
        else:
            await update.message.reply_text(text=caption, parse_mode='Markdown', reply_markup=reply_markup)
    else:
        # Chat simple : on utilise la réponse déjà générée par le premier appel
        reply = intent.get("reply", "Je ne suis pas sûr de comprendre.")
        await update.message.reply_text(reply)

@restricted
async def button_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    query = update.callback_query
    await query.answer()
    
    if query.data == 'confirm_add':
        data = context.user_data.get('last_search')
        if not data:
            await query.edit_message_caption("Erreur : session expirée.")
            return

        success = False
        if data["type"] == "movie":
            success = await add_movie(data["item"])
        else:
            success = await add_series(data["item"], data["season"])
        
        if success:
            season_txt = f" (Saison {data['season']})" if data['season'] else ""
            await query.edit_message_caption(caption=f"✅ '{data['item']['title']}'{season_txt} a été ajouté et la recherche est lancée !")
        else:
            await query.edit_message_caption(caption="❌ Erreur lors de l'ajout ou média déjà existant.")
            
    elif query.data == 'cancel_add':
        await query.edit_message_caption(caption="Annulé !")

if __name__ == '__main__':
    application = ApplicationBuilder().token(TELEGRAM_TOKEN).build()
    
    application.add_handler(CommandHandler('start', start))
    application.add_handler(MessageHandler(filters.TEXT & (~filters.COMMAND), handle_message))
    application.add_handler(CallbackQueryHandler(button_handler))
    
    application.run_polling()

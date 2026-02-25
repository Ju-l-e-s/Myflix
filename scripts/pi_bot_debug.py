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
from datetime import timedelta
from telebot.types import InlineKeyboardMarkup, InlineKeyboardButton
from flask import Flask
import share_engine

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

TOKEN = os.getenv("TELEGRAM_TOKEN")
SUPER_ADMIN = 6721936515
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
USERS_FILE = os.path.join(BASE_DIR, "users.json")
CONTEXT_FILE = "/tmp/bot_context.json"
SCRIPT_PATH = os.path.join(BASE_DIR, "media_manager.py")
NODE_PATH = "/home/jules/.nvm/versions/node/v22.22.0/bin/node"
OPENCLAW_PATH = "/home/jules/.nvm/versions/node/v22.22.0/bin/openclaw"
OPENAI_KEY = os.getenv("OPENAI_KEY")

SONARR_CFG = {"url": "http://localhost:8989", "key": os.getenv("SONARR_API_KEY"), "root": "/tv"}
RADARR_CFG = {"url": "http://localhost:7878", "key": os.getenv("RADARR_API_KEY"), "root": "/movies"}
PLEX_CFG = {"url": "http://localhost:32400", "token": os.getenv("PLEX_TOKEN")}

bot = telebot.TeleBot(TOKEN)

def load_allowed_users():
    try: return json.load(open(USERS_FILE))["allowed"]
    except: return [SUPER_ADMIN]

def is_authorized(uid): return uid in load_allowed_users()

# --- ADMIN MENU ---
@bot.message_handler(commands=['admin'])
def admin_main(m):
    if m.from_user.id != SUPER_ADMIN:
        bot.send_message(SUPER_ADMIN, f"⚠️ **Alerte Sécurité** : Tentative d'accès au menu admin par {m.from_user.id} (@{m.from_user.username})")
        return
    markup = InlineKeyboardMarkup(row_width=1)
    markup.add(InlineKeyboardButton("📂 Partage", callback_data="adm:share"),
               InlineKeyboardButton("👥 Utilisateurs", callback_data="adm:users"),
               InlineKeyboardButton("🧹 Nettoyage SSD", callback_data="adm:clean"),
               InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"))
    bot.send_message(m.chat.id, "🛠️ **Menu Administrateur**", reply_markup=markup, parse_mode='Markdown')

def save_users(users):
    with open(USERS_FILE, "w") as f:
        json.dump({"allowed": users}, f)

@bot.callback_query_handler(func=lambda call: call.data.startswith("adm:"))
def admin_router(call):
    if call.from_user.id != SUPER_ADMIN: return
    parts = call.data.split(":")
    cmd = parts[1]
    
    if cmd == "main":
        markup = InlineKeyboardMarkup(row_width=1)
        markup.add(InlineKeyboardButton("📂 Partage", callback_data="adm:share"), InlineKeyboardButton("👥 Utilisateurs", callback_data="adm:users"), InlineKeyboardButton("🧹 Nettoyage SSD", callback_data="adm:clean"), InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"))
        bot.edit_message_text("🛠️ **Menu Administrateur**", call.message.chat.id, call.message.message_id, reply_markup=markup)
    
    elif cmd == "share":
        markup = InlineKeyboardMarkup(row_width=1)
        markup.add(InlineKeyboardButton("✅ Contenu possédé", callback_data="adm:share_owned"), InlineKeyboardButton("❌ Contenu non possédé", callback_data="adm:share_new"), InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"))
        bot.edit_message_text("📂 **Gestion du Partage**", call.message.chat.id, call.message.message_id, reply_markup=markup)
    
    elif cmd == "users":
        markup = InlineKeyboardMarkup(row_width=1)
        markup.add(InlineKeyboardButton("📋 Lister les accès", callback_data="adm:users_list"),
                   InlineKeyboardButton("➕ Ajouter un utilisateur", callback_data="adm:users_add"),
                   InlineKeyboardButton("❌ Révoquer un accès", callback_data="adm:users_rev"),
                   InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"))
        bot.edit_message_text("👥 **Gestion des Utilisateurs**", call.message.chat.id, call.message.message_id, reply_markup=markup)
    
    elif cmd == "users_list":
        users = load_allowed_users()
        text = "📋 **Utilisateurs autorisés :**\n\n" + "\n".join([f"• `{u}`" for u in users])
        markup = InlineKeyboardMarkup().add(InlineKeyboardButton("⬅️ Retour", callback_data="adm:users"))
        bot.edit_message_text(text, call.message.chat.id, call.message.message_id, reply_markup=markup, parse_mode='Markdown')
    
    elif cmd == "users_add":
        msg = bot.send_message(call.message.chat.id, "👤 Envoyez l'ID Telegram de l'utilisateur à ajouter :")
        bot.register_next_step_handler(msg, process_user_add)
    
    elif cmd == "users_rev":
        users = load_allowed_users()
        markup = InlineKeyboardMarkup(row_width=1)
        for u in users:
            if u == SUPER_ADMIN: continue
            markup.add(InlineKeyboardButton(f"❌ {u}", callback_data=f"adm:users_del:{u}"))
        markup.add(InlineKeyboardButton("⬅️ Retour", callback_data="adm:users"))
        bot.edit_message_text("Sélectionnez l'ID à révoquer :", call.message.chat.id, call.message.message_id, reply_markup=markup)
    
    elif cmd == "users_del":
        uid = int(parts[2])
        users = load_allowed_users()
        if uid in users:
            users.remove(uid)
            save_users(users)
            bot.answer_callback_query(call.id, f"✅ ID {uid} révoqué.")
        admin_router(call) # Refresh menu

    elif cmd == "share_owned":
        markup = InlineKeyboardMarkup(row_width=2)
        markup.add(InlineKeyboardButton("🎬 Films", callback_data="adm:list_owned:movies"), InlineKeyboardButton("📺 Séries", callback_data="adm:list_owned:series"), InlineKeyboardButton("⬅️ Retour", callback_data="adm:share"))
        bot.edit_message_text("Sélectionnez la catégorie :", call.message.chat.id, call.message.message_id, reply_markup=markup)
    
    elif cmd == "list_owned":
        list_media_unified(call.message, parts[2], "Partage", is_edit=True)
    
    elif cmd == "share_new":
        bot.edit_message_text("🔍 **Partage de Nouveau Contenu**\n\nEnvoyez-moi le nom du film ou de la série que vous souhaitez rechercher et partager. L'IA s'occupera de la recherche.", call.message.chat.id, call.message.message_id)
    
    elif cmd == "clean":
        bot.edit_message_text("🧹 **Nettoyage SSD**\n\nFonctionnalité en cours de développement.", call.message.chat.id, call.message.message_id)
    
    elif cmd == "close": bot.delete_message(call.message.chat.id, call.message.message_id)

def process_user_add(m):
    try:
        uid = int(m.text)
        users = load_allowed_users()
        if uid not in users:
            users.append(uid)
            save_users(users)
            bot.reply_to(m, f"✅ ID {uid} ajouté aux autorisations NVMe/HDD")
        else:
            bot.reply_to(m, "ℹ️ Cet utilisateur est déjà autorisé.")
    except:
        bot.reply_to(m, "❌ ID invalide. Veuillez envoyer un nombre.")

# --- MEDIA ENGINE ---
def list_media_unified(m, cat, title, is_edit=False):
    res = subprocess.run(["python3", SCRIPT_PATH, "json_list", cat], capture_output=True, text=True)
    try:
        items = json.loads(res.stdout)
        ctx = {}
        if os.path.exists(CONTEXT_FILE):
            try: ctx = json.load(open(CONTEXT_FILE))
            except: ctx = {}
        ctx[str(m.chat.id)] = {"category": cat, "list": items}
        with open(CONTEXT_FILE, "w") as f: json.dump(ctx, f)
    except Exception as e:
        logging.error(f"Context save failed: {e}")
        return
    if not items:
        msg = f"📂 Aucun(e) {cat} trouvé(e)."
        if is_edit: bot.edit_message_text(msg, m.chat.id, m.message_id)
        else: bot.send_message(m.chat.id, msg)
        return
    text = f"🎬 {title} : {cat}\n\n"
    for i, f in enumerate(items, 1): text += f"{i}. {f['name']} ({f['size_gb']}G)\n"
    markup = InlineKeyboardMarkup(row_width=5)
    markup.add(*[InlineKeyboardButton(str(i), callback_data=f"sel:{cat}:{i-1}") for i in range(1, len(items)+1)])
    markup.add(InlineKeyboardButton("⬅️ Retour", callback_data="adm:share_owned"))
    if is_edit: bot.edit_message_text(text, m.chat.id, m.message_id, reply_markup=markup)
    else: bot.send_message(m.chat.id, text, reply_markup=markup)

@bot.callback_query_handler(func=lambda call: call.data.startswith(("sel:", "act:", "share_exec:", "cancel", "dl:")))
def media_callbacks(call):
    if not is_authorized(call.from_user.id): return
    d = call.data.split(":")
    if d[0] == "sel":
        try:
            with open(CONTEXT_FILE, "r") as f: ctx = json.load(f)
            item = ctx[str(call.message.chat.id)]["list"][int(d[2])]
        except Exception as e:
            logging.error(f"Context load failed for sel: {e}")
            bot.answer_callback_query(call.id, "❌ Session expirée, relancez la recherche.")
            return
        markup = InlineKeyboardMarkup(row_width=2)
        act = "arc" if item['is_nvme'] else "res"
        markup.add(InlineKeyboardButton("📦 Archiv/Rest", callback_data=f"act:{act}:{d[1]}:{d[2]}"),
                   InlineKeyboardButton("🗑️ Supprimer", callback_data=f"act:del:{d[1]}:{d[2]}"))
        if call.from_user.id == SUPER_ADMIN:
            markup.add(InlineKeyboardButton("🔗 Partager", callback_data=f"share_exec:{d[2]}"))
        markup.add(InlineKeyboardButton("❌ Annuler", callback_data="cancel"))
        bot.edit_message_text(f"🛠️ {item['name']}", call.message.chat.id, call.message.message_id, reply_markup=markup)
    elif d[0] == "act":
        try:
            with open(CONTEXT_FILE, "r") as f: ctx = json.load(f)
            item = ctx[str(call.message.chat.id)]["list"][int(d[3])]
        except Exception as e:
            logging.error(f"Context load failed for act: {e}")
            bot.answer_callback_query(call.id, "❌ Session expirée.")
            return
        res = subprocess.run(["python3", SCRIPT_PATH, "action", d[1], d[2], item['path']], capture_output=True, text=True)
        bot.edit_message_text(f"✅ {item['name']}: {res.stdout.strip()}", call.message.chat.id, call.message.message_id)
    elif d[0] == "share_exec":
        try:
            with open(CONTEXT_FILE, "r") as f: ctx = json.load(f)
            item = ctx[str(call.message.chat.id)]["list"][int(d[1])]
        except Exception as e:
            logging.error(f"Context load failed for share_exec: {e}")
            bot.answer_callback_query(call.id, "❌ Session expirée.")
            return
        
        # Check for subtitles before offering choice
        src = pathlib.Path(item['path'])
        subtitles = list(src.parent.glob(f"{src.stem}*.srt"))
        
        if subtitles:
            markup = InlineKeyboardMarkup(row_width=1)
            markup.add(InlineKeyboardButton("⚡ Lien Rapide (Symlink)", callback_data=f"share_final:{d[1]}:link"),
                       InlineKeyboardButton("🎬 Remux avec Sous-titres", callback_data=f"share_final:{d[1]}:remux"),
                       InlineKeyboardButton("❌ Annuler", callback_data="cancel"))
            bot.edit_message_text(f"🎯 **Options de Partage**\n{len(subtitles)} sous-titre(s) trouvé(s).", call.message.chat.id, call.message.message_id, reply_markup=markup)
        else:
            # Direct link if no subs
            admin_router(telebot.types.CallbackQuery(id=call.id, from_user=call.from_user, message=call.message, data=f"share_final:{d[1]}:link", chat_instance=call.chat_instance))

    elif d[0] == "share_final":
        try:
            with open(CONTEXT_FILE, "r") as f: ctx = json.load(f)
            item = ctx[str(call.message.chat.id)]["list"][int(d[1])]
        except Exception as e:
            logging.error(f"Context load failed for share_final: {e}")
            bot.answer_callback_query(call.id, "❌ Session expirée.")
            return

        remux_opt = True if d[2] == "remux" else False
        bot.edit_message_text("⏳ Traitement en cours..." if remux_opt else "⏳ Génération du lien...", call.message.chat.id, call.message.message_id)
        
        data = share_engine.generate_secure_link(item['path'], remux=remux_opt)
        
        if data and "video" in data:
            if data.get("type") == "remuxed":
                msg = f"🎬 **Contenu Remuxé (Softsubs inclus)**\n\n🔗 Vidéo : `{data['video']}`\n\n✅ Les sous-titres ont été intégrés au fichier MKV."
            else:
                msg = f"✅ **Contenu Partagé (Symlink)**\n\n🔗 Vidéo : `{data['video']}`\n"
                if data.get("subtitles"):
                    msg += "\n📄 **Sous-titres (liés et sécurisés) :**\n"
                    for sub in data["subtitles"]:
                        msg += f"• `{sub}`\n"
            bot.send_message(call.message.chat.id, msg, parse_mode='Markdown')
        else:
            bot.send_message(call.message.chat.id, "❌ Erreur: Échec du partage ou contenu hors limites.")
    elif d[0] == "cancel": bot.edit_message_text("❌ Fermé.", call.message.chat.id, call.message.message_id)
    elif d[0] == "dl":
        if d[1] == "no":
            bot.edit_message_text("❌ Téléchargement annulé.", call.message.chat.id, call.message.message_id)
            return
        elif d[1] == "yes":
            try:
                with open(CONTEXT_FILE, "r") as f: ctx = json.load(f)
                dl_data = ctx.get(str(call.message.chat.id) + "_dl")
                if not dl_data:
                    bot.answer_callback_query(call.id, "❌ Session expirée.")
                    return
            except Exception:
                bot.answer_callback_query(call.id, "❌ Erreur de session.")
                return

            type_media = dl_data['type_media']
            cfg = RADARR_CFG if type_media == 'film' else SONARR_CFG
            endpoint = 'movie' if type_media == 'film' else 'series'
            id_key = 'tmdbId' if type_media == 'film' else 'tvdbId'

            payload = {
                "title": dl_data['found_title'],
                "qualityProfileId": 1,
                "titleSlug": dl_data['titleSlug'],
                "images": dl_data['images'],
                "rootFolderPath": cfg['root'],
                "monitored": True,
                "addOptions": {"searchForMovie": True} if type_media == 'film' else {"searchForMissingEpisodes": True}
            }
            payload[id_key] = dl_data['item_id']
            if type_media == 'serie':
                payload["languageProfileId"] = 1

            bot.edit_message_text("⏳ Ajout en cours...", call.message.chat.id, call.message.message_id)
            
            try:
                r_add = requests.post(f"{cfg['url']}/api/v3/{endpoint}", headers={"X-Api-Key": cfg['key']}, json=payload, timeout=10)
                if r_add.status_code in [200, 201]:
                    bot.edit_message_text(f"📥 **Ajouté à la file d'attente :** `{dl_data['found_title']} ({dl_data['year']})`\nLe téléchargement va débuter.", call.message.chat.id, call.message.message_id, parse_mode='Markdown')
                else:
                    logging.error(f"Arr Add Error: {r_add.text}")
                    bot.edit_message_text(f"❌ Échec de l'ajout (Erreur {r_add.status_code}).", call.message.chat.id, call.message.message_id)
            except Exception as e:
                logging.error(f"Arr API Error: {e}")
                bot.edit_message_text("❌ Erreur de communication avec le serveur.", call.message.chat.id, call.message.message_id)

@bot.message_handler(commands=['status'])
def status_command(m):
    if not is_authorized(m.chat.id): return
    try:
        nvme = shutil.disk_usage('/')
        hdd = shutil.disk_usage('/mnt/externe') # Chemin cible HDD
        def fmt(u):
            return f"{u.used/(1024**3):.1f}/{u.total/(1024**3):.1f}Go ({(u.used/u.total)*100:.1f}%)"
        msg = f"📊 **Stockage Temps Réel**\n\n🚀 **NVMe** : `{fmt(nvme)}`\n📚 **HDD** : `{fmt(hdd)}`"
        bot.reply_to(m, msg, parse_mode='Markdown')
    except Exception:
        bot.reply_to(m, "❌ Erreur de lecture disque temps réel.")

@bot.message_handler(commands=['films'])
def films_command(m):
    if not is_authorized(m.chat.id): return
    try:
        logging.info(f"[DEBUG_API] Interrogation Radarr à {RADARR_CFG['url']}")
        r = requests.get(f"{RADARR_CFG['url']}/api/v3/movie", headers={"X-Api-Key": RADARR_CFG['key']}, timeout=5)
        r.raise_for_status()
        data = r.json()
        logging.info(f"[DEBUG_API] Status: {r.status_code} | Nombre de films reçus: {len(data)}")
        owned = [f for f in data if f.get('hasFile') is True]
        owned.sort(key=lambda x: x.get('added', ''), reverse=True)
        text = f"🎬 **Bibliothèque Radarr ({len(owned)} films)**\n\n"
        text += "\n".join([f"• {f['title']} ({f['year']})" for f in owned[:50]])
        bot.reply_to(m, text, parse_mode='Markdown')
    except Exception:
        bot.reply_to(m, "❌ Erreur API Radarr (Consultez les logs).")

@bot.message_handler(commands=['series'])
def series_command(m):
    if not is_authorized(m.chat.id): return
    try:
        r = requests.get(f"{SONARR_CFG['url']}/api/v3/series", headers={"X-Api-Key": SONARR_CFG['key']}, timeout=5)
        r.raise_for_status()
        # Une série est présente si episodeFileCount > 0
        owned = [s for s in r.json() if s.get('statistics', {}).get('episodeFileCount', 0) > 0]
        text = f"📺 **Bibliothèque Sonarr ({len(owned)} séries)**\n\n"
        text += "\n".join([f"• {s['title']} — `{s['statistics']['episodeFileCount']}` épisodes" for s in owned])
        bot.reply_to(m, text, parse_mode='Markdown')
    except Exception:
        bot.reply_to(m, "❌ Erreur API Sonarr (Consultez les logs).")

@bot.message_handler(commands=['queue'])
def queue_command(m):
    if not is_authorized(m.chat.id): return
    
    def clean_title(name):
        name = os.path.splitext(name)[0]
        name = re.sub(r'[\._]', ' ', name)
        match = re.search(r'(.*?)\s*[\(\[]?((?:19|20)\d{2})[\)\]]?(.*)', name, re.IGNORECASE)
        title = match.group(1).strip() if match else name
        year = match.group(2) if match else ""
        tags = [r'\b\d{3,4}p\b', r'\b2160p\b', r'\b4k\b', r'\bHEVC\b', r'\bx26[45]\b', r'\bBluRay\b', r'\bWEB-DL\b', r'\bCOMPLETE\b', r'\bSeason\s*\d*\b']
        clean = f"{title} ({year})" if year else title
        for tag in tags: clean = re.sub(tag, '', clean, flags=re.IGNORECASE)
        return re.sub(r'\s+', ' ', clean).strip()

    q_text = "📥 **Téléchargements (qBittorrent) :**\n\n"
    try:
        r = requests.get("http://localhost:8090/api/v2/torrents/info", timeout=5)
        torrents = r.json()
        if not torrents:
            q_text += "• Aucun téléchargement en cours."
        else:
            for t in torrents:
                if t['state'] in ['downloading', 'stalledDL', 'metaDL', 'queuedDL']:
                    progress = round(t['progress'] * 100, 1)
                    eta_sec = t['eta']
                    if eta_sec >= 8640000: eta = "∞"
                    else:
                        eta = str(timedelta(seconds=eta_sec)).split(".")[0]
                    
                    size = round(t['size'] / 1024**3, 2)
                    clean_name = clean_title(t['name'])
                    q_text += f"• **{clean_name}**\n  └ 📶 {t['state']} | 📊 {progress}% | ⏳ {eta} | 📂 {size}GB\n"
    except Exception as e:
        logging.error(f"qBittorrent Error: {e}")
        q_text += "❌ Erreur de connexion à qBittorrent."
    
    bot.reply_to(m, q_text, parse_mode='Markdown')

# --- AI HANDLER & ARR INTEGRATION ---
def stream_gpt_json(query):
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {OPENAI_KEY}"}
    prompt = f"Analyse cette demande : '{query}'. Tu DOIS extraire le titre, la saison et l'épisode.\n\nRègles de parsing :\n'S01E01' ou 'S1E1' -> saison: 1, episode: 1\n'Saison 2' -> saison: 2, episode: null\nSi c'est un film ou incertain -> type: 'film', saison/episode: null\nSi c'est une série -> type: 'serie'\n\nRenvoie UNIQUEMENT un JSON pur : {{\"titre\": \"str\", \"saison\": \"int|null\", \"episode\": \"int|null\", \"type\": \"serie|film\"}}"
    
    try:
        res = requests.post("https://api.openai.com/v1/chat/completions", headers=headers, json={
            "model": "gpt-4o-mini",
            "messages": [{"role": "user", "content": prompt}],
            "temperature": 0.1,
            "stream": True
        }, stream=True, timeout=10)
        
        if res.status_code != 200:
            logging.error(f"[API_ERROR] HTTP {res.status_code} : {res.text}")
            yield {"error": f"API HTTP {res.status_code}"}
            return
            
        buffer = ""
        for line in res.iter_lines():
            if line:
                line_str = line.decode('utf-8')
                if line_str.startswith("data: ") and line_str != "data: [DONE]":
                    try:
                        chunk = json.loads(line_str[6:])
                        delta = chunk['choices'][0]['delta'].get('content', '')
                        if delta:
                            buffer += delta
                    except Exception:
                        continue
        
        # --- PARSING FINAL SÉCURISÉ ---
        clean_buffer = buffer.replace('```json', '').replace('```', '').strip()
        
        logging.info(f"[DEBUG_PIPE] Buffer brut de l'IA : {repr(clean_buffer)}")
        
        match = re.search(r'\{.*\}', clean_buffer, re.DOTALL)
        if match:
            logging.info(f"[DEBUG_PIPE] Regex Match : {repr(match.group(0))}")
            try:
                parsed = json.loads(match.group(0))
                # Typage fort (Sécurité)
                parsed['titre'] = str(parsed.get('titre', ''))
                parsed['saison'] = int(parsed.get('saison')) if parsed.get('saison') is not None else None
                parsed['episode'] = int(parsed.get('episode')) if parsed.get('episode') is not None else None
                parsed['type'] = str(parsed.get('type', ''))
                
                yield parsed
                return
            except (ValueError, TypeError) as e:
                logging.error(f"Erreur de typage JSON IA : {e}")
                
        logging.error(f"Échec du Regex JSON sur le buffer : {clean_buffer}")
        yield {"error": "Pipe Break"}
            
    except Exception as e:
        logging.error(f"AI Stream Pipe Error: {e}")
        yield {"error": str(e)}

@bot.message_handler(commands=['get'])
def handle_get_command_entry(m):
    if not is_authorized(m.chat.id): return
    parts = m.text.split(' ', 1)
    if len(parts) > 1 and parts[1].strip():
        query = parts[1].strip()
        process_get_request(m, query)
    else:
        msg = bot.reply_to(m, "🎬 Que veux-tu télécharger ? (Ex: Shogun S01E01 ou Dune)")
        bot.register_next_step_handler(msg, process_get_step)

def process_get_step(m):
    if not is_authorized(m.chat.id): return
    if m.text.startswith('/'):
        return
    process_get_request(m, m.text)

def process_get_request(m, query):
    bot.send_chat_action(m.chat.id, 'typing')
    
    # 1. Phase d'Extraction (Pipe Stream)
    data = None
    for result in stream_gpt_json(query):
        if result:
            data = result
            break

    if not data or "error" in data:
        bot.reply_to(m, "❌ Le flux d'analyse a échoué (Pipe Break). Données invalides.")
        return

    # Sécurité: Forcer la conversion des types natifs
    try:
        titre = str(data.get('titre'))
        type_media = str(data.get('type'))
        saison = int(data['saison']) if data.get('saison') is not None else None
        episode = int(data['episode']) if data.get('episode') is not None else None
    except (ValueError, TypeError):
        bot.reply_to(m, "❌ Erreur de formatage des métadonnées extraites.")
        return

    bot.send_message(m.chat.id, f"🔍 **Recherche en cours :** `{titre}` ({type_media.capitalize()})", parse_mode='Markdown')

    # 2. Phase de Routage & Vérification de Disponibilité locale
    try:
        res_local = subprocess.run(["python3", SCRIPT_PATH, "search_local", titre], capture_output=True, text=True)
        local_items = json.loads(res_local.stdout)
        if local_items:
            bot.send_message(m.chat.id, f"✅ `{titre}` est déjà disponible localement dans la bibliothèque.")
            return
    except Exception as e:
        logging.error(f"Erreur recherche locale: {e}")

    cfg = RADARR_CFG if type_media == 'film' else SONARR_CFG
    endpoint = 'movie' if type_media == 'film' else 'series'
    id_key = 'tmdbId' if type_media == 'film' else 'tvdbId'
    
    try:
        # Recherche API Sonarr/Radarr
        r_lookup = requests.get(f"{cfg['url']}/api/v3/{endpoint}/lookup?term={titre}", headers={"X-Api-Key": cfg['key']}, timeout=10)
        results = r_lookup.json()
        
        if not results:
            bot.send_message(m.chat.id, f"❌ `{titre}` introuvable dans les bases de données (TMDb/TVDb).")
            return

        item = results[0]
        found_title = item.get('title', titre)
        year = item.get('year', '')
        item_id = item.get(id_key)

        # Validation de Flux: Vérification si déjà dans Sonarr/Radarr
        r_check = requests.get(f"{cfg['url']}/api/v3/{endpoint}", headers={"X-Api-Key": cfg['key']}, timeout=10)
        exists = any(str(i.get(id_key)) == str(item_id) for i in r_check.json())
        
        if exists:
            bot.send_message(m.chat.id, f"✅ `{found_title} ({year})` est déjà possédé ou en cours de téléchargement sur {'Radarr' if type_media == 'film' else 'Sonarr'}.")
            return

        # 3. Phase de Contrainte & Confirmation
        try:
            ctx = {}
            if os.path.exists(CONTEXT_FILE):
                try: ctx = json.load(open(CONTEXT_FILE))
                except: ctx = {}
            ctx[str(m.chat.id) + "_dl"] = {
                "type_media": type_media,
                "found_title": found_title,
                "year": year,
                "item_id": item_id,
                "titleSlug": item.get('titleSlug'),
                "images": item.get('images')
            }
            with open(CONTEXT_FILE, "w") as f: json.dump(ctx, f)
        except Exception as e:
            logging.error(f"Context save failed: {e}")
            bot.send_message(m.chat.id, "❌ Erreur interne lors de la préparation du téléchargement.")
            return

        markup = InlineKeyboardMarkup(row_width=2)
        markup.add(
            InlineKeyboardButton("✅ Oui, télécharger", callback_data="dl:yes"),
            InlineKeyboardButton("❌ Non, annuler", callback_data="dl:no")
        )
        
        bot.send_message(
            m.chat.id, 
            f"🎯 **Correspondance trouvée :**\n\n"
            f"🎬 **Titre :** `{found_title}`\n"
            f"📅 **Année :** `{year}`\n"
            f"📺 **Type :** `{'Série' if type_media == 'serie' else 'Film'}`\n\n"
            f"Voulez-vous lancer le téléchargement ?",
            reply_markup=markup,
            parse_mode='Markdown'
        )

    except requests.exceptions.RequestException as e:
        logging.error(f"Arr API Error: {e}")
        bot.send_message(m.chat.id, "❌ Erreur de communication avec les services de téléchargement.")

@bot.message_handler(func=lambda m: True)
def handle_ai_request(m):
    if not is_authorized(m.chat.id): return
    if m.text.startswith('/'): return # Ignore native commands
    process_get_request(m, m.text)

# Webhook server start
webhook_app = Flask(__name__)
threading.Thread(target=lambda: webhook_app.run(host='0.0.0.0', port=5001), daemon=True).start()

bot.infinity_polling()

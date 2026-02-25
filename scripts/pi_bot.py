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
import share_engine

logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)

# --- CONFIGURATION ---
TOKEN = os.getenv("TELEGRAM_TOKEN")
SUPER_ADMIN = 6721936515
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
USERS_FILE = os.path.join(BASE_DIR, "users.json")
CONTEXT_FILE = "/tmp/bot_context.json"
SCRIPT_PATH = os.path.join(BASE_DIR, "media_manager.py")
RADARR_CFG = {"url": "http://localhost:7878", "key": os.getenv("RADARR_API_KEY")}
SONARR_CFG = {"url": "http://localhost:8989", "key": os.getenv("SONARR_API_KEY")}
OPENAI_KEY = os.getenv("OPENAI_KEY")

bot = telebot.TeleBot(TOKEN)


# --- UTILS & STORAGE ---
def load_allowed_users():
    try:
        return json.load(open(USERS_FILE))["allowed"]
    except:
        return [SUPER_ADMIN]


def is_authorized(uid):
    return uid in load_allowed_users()


def save_users(users):
    with open(USERS_FILE, "w") as f:
        json.dump({"allowed": users}, f)


def get_storage_emoji(item):
    if (
        not item.get("hasFile")
        and item.get("statistics", {}).get("episodeFileCount", 0) == 0
    ):
        return "🔴"
    path = item.get("path", "")
    return "📚" if path and "/mnt/externe" in path else "🚀"


# --- ADMIN HANDLERS ---
@bot.message_handler(commands=["admin"])
def admin_main(m):
    if m.from_user.id != SUPER_ADMIN:
        return
    markup = InlineKeyboardMarkup(row_width=1)
    markup.add(
        InlineKeyboardButton("📂 Partage", callback_data="adm:share"),
        InlineKeyboardButton("👥 Utilisateurs", callback_data="adm:users"),
        InlineKeyboardButton("🧹 Nettoyage SSD", callback_data="adm:clean"),
        InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"),
    )
    bot.send_message(
        m.chat.id,
        "🛠️ **Menu Administrateur**",
        reply_markup=markup,
        parse_mode="Markdown",
    )


def process_user_add(m):
    try:
        uid = int(m.text)
        users = load_allowed_users()
        if uid not in users:
            users.append(uid)
            save_users(users)
            bot.reply_to(m, f"✅ ID {uid} ajouté.")
        else:
            bot.reply_to(m, "ℹ️ Déjà autorisé.")
    except:
        bot.reply_to(m, "❌ ID invalide.")


# --- MEDIA ENGINE (ORIGINAL) ---
def list_media_unified(m, cat, title, is_edit=False):
    res = subprocess.run(
        ["python3", SCRIPT_PATH, "json_list", cat], capture_output=True, text=True
    )
    try:
        items = json.loads(res.stdout)
        ctx = {}
        if os.path.exists(CONTEXT_FILE):
            try:
                ctx = json.load(open(CONTEXT_FILE))
            except:
                ctx = {}
        ctx[str(m.chat.id)] = {"category": cat, "list": items}
        with open(CONTEXT_FILE, "w") as f:
            json.dump(ctx, f)
    except:
        return
    if not items:
        msg = f"📂 Aucun(e) {cat} trouvé(e)."
        if is_edit:
            bot.edit_message_text(msg, m.chat.id, m.message_id)
        else:
            bot.send_message(m.chat.id, msg)
        return
    text = f"🎬 {title} : {cat}\n\n"
    for i, f in enumerate(items, 1):
        text += f"{i}. {f['name']} ({f['size_gb']}G)\n"
    markup = InlineKeyboardMarkup(row_width=5)
    markup.add(
        *[
            InlineKeyboardButton(str(i), callback_data=f"sel:{cat}:{i - 1}")
            for i in range(1, len(items) + 1)
        ]
    )
    markup.add(InlineKeyboardButton("⬅️ Retour", callback_data="adm:share_owned"))
    if is_edit:
        bot.edit_message_text(text, m.chat.id, m.message_id, reply_markup=markup)
    else:
        bot.send_message(m.chat.id, text, reply_markup=markup)


# --- PAGED MEDIA UI (LIVE API) ---
def show_media_list(chat_id, message_id, category, page=0, is_new=False):
    cfg = RADARR_CFG if category == "films" else SONARR_CFG
    endpoint = "movie" if category == "films" else "series"
    try:
        r = requests.get(
            f"{cfg['url']}/api/v3/{endpoint}",
            headers={"X-Api-Key": cfg["key"]},
            timeout=5,
        )
        r.raise_for_status()
        data = r.json()
        if category == "films":
            owned = [f for f in data if f.get("hasFile") is True]
        else:
            owned = [
                s
                for s in data
                if s.get("statistics", {}).get("episodeFileCount", 0) > 0
            ]
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
        btns = [
            InlineKeyboardButton(
                str(start + i), callback_data=f"m_sel:{category}:{items[i - 1]['id']}"
            )
            for i in range(1, len(items) + 1)
        ]
        markup.add(*btns)
        nav = []
        if page > 0:
            nav.append(
                InlineKeyboardButton("⬅️", callback_data=f"m_pag:{category}:{page - 1}")
            )
        if end < total:
            nav.append(
                InlineKeyboardButton("➡️", callback_data=f"m_pag:{category}:{page + 1}")
            )
        if nav:
            markup.row(*nav)
        markup.add(InlineKeyboardButton("🏠 Menu Principal", callback_data="adm:main"))
        if is_new:
            bot.send_message(chat_id, text, reply_markup=markup, parse_mode="Markdown")
        else:
            bot.edit_message_text(
                text, chat_id, message_id, reply_markup=markup, parse_mode="Markdown"
            )
    except:
        bot.send_message(chat_id, "❌ Erreur API.")


@bot.message_handler(commands=["films"])
def films_cmd(m):
    if is_authorized(m.chat.id):
        show_media_list(m.chat.id, None, "films", 0, True)


@bot.message_handler(commands=["series"])
def series_cmd(m):
    if is_authorized(m.chat.id):
        show_media_list(m.chat.id, None, "series", 0, True)


# --- SEARCH ENGINE (/GET) ---
def stream_gpt_json(query):
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {OPENAI_KEY}",
    }
    prompt = f'Analyse : \'{query}\'. Renvoie JSON : {{"titre": "str", "type": "serie|film"}}'
    try:
        res = requests.post(
            "https://api.openai.com/v1/chat/completions",
            headers=headers,
            json={
                "model": "gpt-4o-mini",
                "messages": [{"role": "user", "content": prompt}],
                "temperature": 0.1,
            },
            timeout=10,
        )
        return res.json()["choices"][0]["message"]["content"]
    except:
        return None


@bot.message_handler(commands=["get"])
def handle_get(m):
    if not is_authorized(m.chat.id):
        return
    parts = m.text.split(" ", 1)
    if len(parts) > 1:
        process_get_request(m, parts[1])
    else:
        bot.reply_to(m, "🎬 Que veux-tu ?")


def process_get_request(m, query):
    bot.send_chat_action(m.chat.id, "typing")
    ai_res = stream_gpt_json(query)
    try:
        data = json.loads(re.search(r"\{.*\}", ai_res, re.DOTALL).group(0))
        titre = data["titre"]
        cat = "movie" if data["type"] == "film" else "series"
    except:
        titre = query
        cat = "movie"
    cfg = RADARR_CFG if cat == "movie" else SONARR_CFG
    try:
        r = requests.get(
            f"{cfg['url']}/api/v3/{cat}/lookup?term={titre}",
            headers={"X-Api-Key": cfg["key"]},
            timeout=5,
        )
        results = r.json()
        if not results:
            bot.send_message(m.chat.id, "❌ Aucun résultat.")
            return
        text = f"🔍 **Résultats pour '{titre}'**\n\n"
        limit = min(len(results), 5)
        for i, res in enumerate(results[:limit], 1):
            text += f"{i}. {res['title']} ({res.get('year', 'N/A')})\n"
        markup = InlineKeyboardMarkup()
        btns = [
            InlineKeyboardButton(str(i), callback_data=f"dl_sel:{cat}:{i}")
            for i in range(1, limit + 1)
        ]
        markup.add(*btns)
        bot.send_message(m.chat.id, text, reply_markup=markup, parse_mode="Markdown")
    except:
        bot.send_message(m.chat.id, "❌ Erreur.")


# --- MASTER CALLBACK ROUTER ---
@bot.callback_query_handler(func=lambda call: True)
def master_router(call):
    if not is_authorized(call.from_user.id):
        return
    d = call.data.split(":")

    # 1. New Paged UI Callbacks
    if d[0] in ["m_pag", "m_sel", "m_act", "dl_sel"]:
        if d[0] == "m_pag":
            show_media_list(
                call.message.chat.id, call.message.message_id, d[1], int(d[2])
            )
        elif d[0] == "m_sel":
            markup = InlineKeyboardMarkup()
            markup.add(
                InlineKeyboardButton(
                    "🗑️ Supprimer", callback_data=f"m_act:del:{d[1]}:{d[2]}"
                ),
                InlineKeyboardButton(
                    "🔗 Partager", callback_data=f"m_act:share:{d[1]}:{d[2]}"
                ),
                InlineKeyboardButton("⬅️ Retour", callback_data=f"m_pag:{d[1]}:0"),
            )
            bot.edit_message_text(
                f"🎯 **Action sur ID {d[2]}**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif d[0] == "dl_sel":
            bot.answer_callback_query(call.id, "📥 Ajouté.")
        elif d[0] == "m_act":
            bot.answer_callback_query(
                call.id, f"Action {d[1]} enregistrée.", show_alert=True
            )
        return

    # 2. Admin Callbacks
    if d[0] == "adm":
        cmd = d[1]
        if cmd == "main":
            markup = InlineKeyboardMarkup(row_width=1)
            markup.add(
                InlineKeyboardButton("📂 Partage", callback_data="adm:share"),
                InlineKeyboardButton("👥 Utilisateurs", callback_data="adm:users"),
                InlineKeyboardButton("🧹 Nettoyage SSD", callback_data="adm:clean"),
                InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"),
            )
            bot.edit_message_text(
                "🛠️ **Menu Administrateur**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "share":
            markup = InlineKeyboardMarkup(row_width=1)
            markup.add(
                InlineKeyboardButton(
                    "✅ Contenu possédé", callback_data="adm:share_owned"
                ),
                InlineKeyboardButton(
                    "❌ Contenu non possédé", callback_data="adm:share_new"
                ),
                InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"),
            )
            bot.edit_message_text(
                "📂 **Gestion du Partage**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users":
            markup = InlineKeyboardMarkup(row_width=1)
            markup.add(
                InlineKeyboardButton("📋 Lister", callback_data="adm:users_list"),
                InlineKeyboardButton("➕ Ajouter", callback_data="adm:users_add"),
                InlineKeyboardButton("❌ Révoquer", callback_data="adm:users_rev"),
                InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"),
            )
            bot.edit_message_text(
                "👥 **Gestion des Utilisateurs**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users_list":
            users = load_allowed_users()
            text = "📋 **Utilisateurs :**\n\n" + "\n".join([f"• `{u}`" for u in users])
            bot.edit_message_text(
                text,
                call.message.chat.id,
                call.message.message_id,
                reply_markup=InlineKeyboardMarkup().add(
                    InlineKeyboardButton("⬅️ Retour", callback_data="adm:users")
                ),
                parse_mode="Markdown",
            )
        elif cmd == "users_add":
            msg = bot.send_message(call.message.chat.id, "👤 ID Telegram :")
            bot.register_next_step_handler(msg, process_user_add)
        elif cmd == "users_rev":
            users = load_allowed_users()
            markup = InlineKeyboardMarkup(row_width=1)
            for u in users:
                if u == SUPER_ADMIN:
                    continue
                markup.add(
                    InlineKeyboardButton(f"❌ {u}", callback_data=f"adm:users_del:{u}")
                )
            markup.add(InlineKeyboardButton("⬅️ Retour", callback_data="adm:users"))
            bot.edit_message_text(
                "Sélectionnez l'ID à révoquer :",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users_del":
            uid = int(d[2])
            users = load_allowed_users()
            if uid in users:
                users.remove(uid)
                save_users(users)
                bot.answer_callback_query(call.id, "✅ Révoqué.")
            master_router(
                types.CallbackQuery(
                    id=call.id,
                    from_user=call.from_user,
                    message=call.message,
                    data="adm:users",
                    chat_instance=call.chat_instance,
                )
            )
        elif cmd == "share_owned":
            markup = InlineKeyboardMarkup(row_width=2)
            markup.add(
                InlineKeyboardButton("🎬 Films", callback_data="adm:list_owned:movies"),
                InlineKeyboardButton(
                    "📺 Séries", callback_data="adm:list_owned:series"
                ),
                InlineKeyboardButton("⬅️ Retour", callback_data="adm:share"),
            )
            bot.edit_message_text(
                "Catégorie :",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "list_owned":
            list_media_unified(call.message, d[2], "Partage", True)
        elif cmd == "close":
            bot.delete_message(call.message.chat.id, call.message.message_id)
        return

    # 3. Media Engine Original Callbacks (sel, act, share_exec, share_final)
    if d[0] == "sel":
        try:
            with open(CONTEXT_FILE, "r") as f:
                ctx = json.load(f)
                item = ctx[str(call.message.chat.id)]["list"][int(d[2])]
            markup = InlineKeyboardMarkup(row_width=2)
            act = "arc" if item["is_nvme"] else "res"
            markup.add(
                InlineKeyboardButton(
                    "📦 Archiv/Rest", callback_data=f"act:{act}:{d[1]}:{d[2]}"
                ),
                InlineKeyboardButton(
                    "🗑️ Supprimer", callback_data=f"act:del:{d[1]}:{d[2]}"
                ),
            )
            if call.from_user.id == SUPER_ADMIN:
                markup.add(
                    InlineKeyboardButton(
                        "🔗 Partager", callback_data=f"share_exec:{d[2]}"
                    )
                )
            markup.add(InlineKeyboardButton("❌ Annuler", callback_data="cancel"))
            bot.edit_message_text(
                f"🛠️ {item['name']}",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        except:
            bot.answer_callback_query(call.id, "❌ Session expirée.")
    elif d[0] == "act":
        try:
            with open(CONTEXT_FILE, "r") as f:
                ctx = json.load(f)
                item = ctx[str(call.message.chat.id)]["list"][int(d[3])]
            res = subprocess.run(
                ["python3", SCRIPT_PATH, "action", d[1], d[2], item["path"]],
                capture_output=True,
                text=True,
            )
            bot.edit_message_text(
                f"✅ {item['name']}: {res.stdout.strip()}",
                call.message.chat.id,
                call.message.message_id,
            )
        except:
            bot.answer_callback_query(call.id, "❌ Erreur.")
    elif d[0] == "share_exec":
        try:
            with open(CONTEXT_FILE, "r") as f:
                ctx = json.load(f)
                item = ctx[str(call.message.chat.id)]["list"][int(d[1])]
            src = pathlib.Path(item["path"])
            subtitles = list(src.parent.glob(f"{src.stem}*.srt"))
            if subtitles:
                markup = InlineKeyboardMarkup(row_width=1)
                markup.add(
                    InlineKeyboardButton(
                        "⚡ Symlink", callback_data=f"share_final:{d[1]}:link"
                    ),
                    InlineKeyboardButton(
                        "🎬 Remux", callback_data=f"share_final:{d[1]}:remux"
                    ),
                    InlineKeyboardButton("❌ Annuler", callback_data="cancel"),
                )
                bot.edit_message_text(
                    f"🎯 **Options Partage**\n{len(subtitles)} sous-titres trouvés.",
                    call.message.chat.id,
                    call.message.message_id,
                    reply_markup=markup,
                )
            else:
                master_router(
                    types.CallbackQuery(
                        id=call.id,
                        from_user=call.from_user,
                        message=call.message,
                        data=f"share_final:{d[1]}:link",
                        chat_instance=call.chat_instance,
                    )
                )
        except:
            bot.answer_callback_query(call.id, "❌ Erreur.")
    elif d[0] == "share_final":
        try:
            with open(CONTEXT_FILE, "r") as f:
                ctx = json.load(f)
                item = ctx[str(call.message.chat.id)]["list"][int(d[1])]
            remux_opt = True if d[2] == "remux" else False
            bot.edit_message_text(
                "⏳ Traitement...", call.message.chat.id, call.message.message_id
            )
            data = share_engine.generate_secure_link(item["path"], remux=remux_opt)
            if data and "video" in data:
                msg = f"✅ **Partagé**\n\n🔗 Vidéo : `{data['video']}`"
                bot.send_message(call.message.chat.id, msg, parse_mode="Markdown")
            else:
                bot.send_message(call.message.chat.id, "❌ Échec du partage.")
        except:
            bot.send_message(call.message.chat.id, "❌ Erreur interne.")
    elif d[0] == "cancel":
        bot.edit_message_text(
            "❌ Fermé.", call.message.chat.id, call.message.message_id
        )


# --- STATUS & QUEUE ---
@bot.message_handler(commands=["status"])
def status_command(m):
    if not is_authorized(m.chat.id):
        return
    try:
        nvme = shutil.disk_usage("/")

        def fmt(u):
            return f"{u.used / (1024**3):.1f}/{u.total / (1024**3):.1f}Go ({(u.used / u.total) * 100:.1f}%)"

        msg = f"📊 **Stockage Temps Réel**\n\n🚀 **NVMe** : `{fmt(nvme)}`"
        if os.path.exists("/mnt/externe"):
            hdd = shutil.disk_usage("/mnt/externe")
            msg += f"\n📚 **HDD** : `{fmt(hdd)}`"
        bot.reply_to(m, msg, parse_mode="Markdown")
    except:
        bot.reply_to(m, "❌ Erreur lecture disque.")


@bot.message_handler(commands=["queue"])
def queue_command(m):
    if not is_authorized(m.chat.id):
        return
    try:
        r = requests.get("http://localhost:8090/api/v2/torrents/info", timeout=5)
        torrents = r.json()
        if not torrents:
            bot.reply_to(m, "• Queue vide.")
            return
        q_text = "📥 **Queue :**\n\n"
        for t in torrents:
            if t["state"] in ["downloading", "stalledDL", "metaDL", "queuedDL"]:
                name = re.sub(r"[\._]", " ", t["name"])
                q_text += f"• **{name}**\n  └ 📊 {round(t['progress'] * 100, 1)}% | 📂 {round(t['size'] / 1024**3, 2)}GB\n"
        bot.reply_to(m, q_text, parse_mode="Markdown")
    except:
        bot.reply_to(m, "❌ Erreur qBit.")


# --- STARTUP ---
webhook_app = Flask(__name__)


@webhook_app.route("/")
def index():
    return "Bot Active"


threading.Thread(
    target=lambda: webhook_app.run(host="0.0.0.0", port=5001), daemon=True
).start()

print("🚀 Bot démarré avec Integrity Shield...")
bot.infinity_polling()

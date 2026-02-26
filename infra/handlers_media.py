import requests
import os
import json
import subprocess
import re
import pathlib
import sys
import time
from datetime import timedelta
from telebot import types
from config import *
import share_engine

# Add scripts directory to path to import media_manager
sys.path.append("/home/jules/scripts")
import media_manager


def get_storage_emoji(item):
    if (
        not item.get("hasFile")
        and item.get("statistics", {}).get("episodeFileCount", 0) == 0
    ):
        return "🔴"
    path = item.get("path", "")
    return "📚" if path and "/mnt/externe" in path else "🚀"


def list_media_unified(bot, m, cat, title, is_edit=False):
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
    text = f"{title} : {cat}\n\n"
    for i, f in enumerate(items, 1):
        text += f"{i}. {f['name']} ({f['size_gb']}G)\n"
    markup = types.InlineKeyboardMarkup(row_width=5)
    markup.add(
        *[
            types.InlineKeyboardButton(str(i), callback_data=f"sel:{cat}:{i - 1}")
            for i in range(1, len(items) + 1)
        ]
    )
    markup.add(types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:share_owned"))
    if is_edit:
        bot.edit_message_text(text, m.chat.id, m.message_id, reply_markup=markup)
    else:
        bot.send_message(m.chat.id, text, reply_markup=markup)


def show_media_list(
    bot, chat_id, message_id, category, page=0, is_new=False, is_authorized=None
):
    cfg = RADARR_CFG if category == "films" else SONARR_CFG
    endpoint = "movie" if category == "films" else "series"
    try:
        r = requests.get(
            f"{cfg['url']}/api/v3/{endpoint}",
            headers={"X-Api-Key": cfg["key"]},
            timeout=10,
        )
        r.raise_for_status()
        data = r.json()
        data.sort(key=lambda x: x.get("added", ""), reverse=True)
        total = len(data)
        start, end = page * 10, (page + 1) * 10
        items = data[start:end]
        if not items:
            bot.send_message(chat_id, "Vide.")
            return
        text = f"{'🎬' if category == 'films' else '📺'} **{category.upper()}** ({total})\n\n"
        for i, item in enumerate(items, 1):
            clean_title = media_manager.clean_title(item['title'])
            text += f"{clean_title} ({item.get('year', '')})\n"
        markup = types.InlineKeyboardMarkup(row_width=5)
        btns = [
            types.InlineKeyboardButton(
                str(start + i), callback_data=f"m_sel:{category}:{items[i - 1]['id']}"
            )
            for i in range(1, len(items) + 1)
        ]
        markup.add(*btns)
        nav_row = []
        if page > 0:
            nav_row.append(
                types.InlineKeyboardButton(
                    "⬅️", callback_data=f"m_pag:{category}:{page - 1}"
                )
            )
        if end < total:
            nav_row.append(
                types.InlineKeyboardButton(
                    f"➡️ ({total - end})", callback_data=f"m_pag:{category}:{page + 1}"
                )
            )
        if nav_row:
            markup.row(*nav_row)
        markup.add(
            types.InlineKeyboardButton("🏠 Menu Admin", callback_data="adm:main")
        )
        if is_new:
            bot.send_message(chat_id, text, reply_markup=markup, parse_mode="Markdown")
        else:
            bot.edit_message_text(
                text, chat_id, message_id, reply_markup=markup, parse_mode="Markdown"
            )
    except Exception as e:
        bot.send_message(chat_id, f"❌ Erreur API : {e}")


def register_media_handlers(bot, is_authorized):
    @bot.message_handler(commands=["films"])
    def films_cmd(m):
        if is_authorized(m.chat.id):
            show_media_list(bot, m.chat.id, None, "films", 0, True, is_authorized)

    @bot.message_handler(commands=["series"])
    def series_cmd(m):
        if is_authorized(m.chat.id):
            show_media_list(bot, m.chat.id, None, "series", 0, True, is_authorized)

    @bot.message_handler(commands=["queue"])
    def queue_command(m):
        if not is_authorized(m.chat.id):
            return
        send_queue_status(bot, m.chat.id)

    def send_queue_status(bot, chat_id, message_id=None):
        try:
            r = requests.get(f"{QBIT_URL}/api/v2/torrents/info", timeout=5)
            torrents = r.json()
            if not torrents:
                msg = "📥 **QUEUE DE TÉLÉCHARGEMENT**\n\n• Queue vide."
                if message_id:
                    bot.edit_message_text(msg, chat_id, message_id, parse_mode="Markdown")
                else:
                    bot.send_message(chat_id, msg, parse_mode="Markdown")
                return

            q_text = "📥 **QUEUE DE TÉLÉCHARGEMENT**\n\n"
            found = False
            for t in torrents:
                if t["state"] in [
                    "downloading",
                    "stalledDL",
                    "metaDL",
                    "queuedDL",
                    "checkingDL",
                    "pausedDL",
                ]:
                    found = True
                    clean_name = media_manager.clean_media_name(t["name"])
                    progress_val = round(t["progress"] * 100, 1)
                    progress_bar = media_manager.get_progress_bar(progress_val)
                    size = media_manager.format_size(t["size"])
                    speed = media_manager.format_speed(t["dlspeed"])
                    eta_sec = t.get("eta", 8640000)
                    eta = "∞" if eta_sec >= 8640000 else str(timedelta(seconds=eta_sec))

                    q_text += f"🎬 **{clean_name}**\n"
                    q_text += f"{progress_bar}\n"
                    q_text += f"📦 {size}  •  🚀 {speed}  •  ⏳ {eta}\n\n"

            if not found:
                q_text += "• Aucun téléchargement en cours."

            markup = types.InlineKeyboardMarkup()
            markup.add(types.InlineKeyboardButton("🔄 Actualiser", callback_data="q_refresh"))

            if message_id:
                try:
                    bot.edit_message_text(q_text, chat_id, message_id, reply_markup=markup, parse_mode="Markdown")
                except Exception as e:
                    if "message is not modified" not in str(e):
                        raise e
            else:
                bot.send_message(chat_id, q_text, reply_markup=markup, parse_mode="Markdown")
        except Exception as e:
            bot.send_message(chat_id, f"❌ Erreur qBit : {e}")

    @bot.callback_query_handler(
        func=lambda call: call.data.startswith(
            (
                "m_pag",
                "m_sel",
                "m_act",
                "sel",
                "act",
                "share_exec",
                "share_final",
                "cancel",
                "q_refresh",
            )
        )
    )
    def media_callback_router(call):
        if not is_authorized(call.from_user.id):
            return
        d = call.data.split(":")

        if d[0] == "q_refresh":
            send_queue_status(bot, call.message.chat.id, call.message.message_id)
            bot.answer_callback_query(call.id, "✅ Mis à jour.")
        elif d[0] == "m_pag":
            show_media_list(
                bot, call.message.chat.id, call.message.message_id, d[1], int(d[2])
            )
        elif d[0] == "m_sel":
            cat, item_id = d[1], d[2]
            cfg = RADARR_CFG if cat == "films" else SONARR_CFG
            endpoint = "movie" if cat == "films" else "series"
            display_title = f"ID {item_id}"
            try:
                r = requests.get(
                    f"{cfg['url']}/api/v3/{endpoint}/{item_id}",
                    headers={"X-Api-Key": cfg["key"]},
                    timeout=5,
                )
                if r.status_code == 200:
                    display_title = r.json().get("title", display_title)
            except:
                pass
            markup = types.InlineKeyboardMarkup()
            markup.add(
                types.InlineKeyboardButton(
                    "🗑️ Supprimer", callback_data=f"m_act:del:{d[1]}:{d[2]}"
                ),
                types.InlineKeyboardButton(
                    "🔗 Partager", callback_data=f"m_act:share:{d[1]}:{d[2]}"
                ),
                types.InlineKeyboardButton("⬅️ Retour", callback_data=f"m_pag:{d[1]}:0"),
            )
            bot.edit_message_text(
                f"🎯 **Action : {display_title}**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
                parse_mode="Markdown",
            )
        elif d[0] == "m_act":
            action, cat, item_id = d[1], d[2], d[3]
            if action == "share":
                bot.answer_callback_query(call.id, "⏳ Génération du lien...")
                cfg = RADARR_CFG if cat == "films" else SONARR_CFG
                endpoint = "movie" if cat == "films" else "series"
                try:
                    r = requests.get(
                        f"{cfg['url']}/api/v3/{endpoint}/{item_id}",
                        headers={"X-Api-Key": cfg["key"]},
                        timeout=5,
                    )
                    if r.status_code == 200:
                        path = r.json().get("path")
                        if path:
                            data = share_engine.generate_secure_link(path)
                            if data and "video" in data:
                                msg = f"✅ **Lien de partage généré :**\n\n🔗 `{data['video']}`"
                                bot.send_message(
                                    call.message.chat.id, msg, parse_mode="Markdown"
                                )
                            else:
                                bot.send_message(call.message.chat.id, "❌ Échec lien.")
                        else:
                            bot.send_message(call.message.chat.id, "❌ Chemin absent.")
                except:
                    bot.send_message(call.message.chat.id, "❌ Erreur API.")
            else:
                bot.answer_callback_query(call.id, f"Action {action} ok.")

        elif d[0] == "sel":
            try:
                with open(CONTEXT_FILE, "r") as f:
                    ctx = json.load(f)
                    item = ctx[str(call.message.chat.id)]["list"][int(d[2])]
                markup = types.InlineKeyboardMarkup(row_width=2)
                act = "arc" if item["is_nvme"] else "res"
                markup.add(
                    types.InlineKeyboardButton(
                        "📦 Archiv/Rest", callback_data=f"act:{act}:{d[1]}:{d[2]}"
                    ),
                    types.InlineKeyboardButton(
                        "🗑️ Supprimer", callback_data=f"act:del:{d[1]}:{d[2]}"
                    ),
                )
                if call.from_user.id == SUPER_ADMIN:
                    markup.add(
                        types.InlineKeyboardButton(
                            "🔗 Partager", callback_data=f"share_exec:{d[2]}"
                        )
                    )
                markup.add(
                    types.InlineKeyboardButton("❌ Annuler", callback_data="cancel")
                )
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
                    markup = types.InlineKeyboardMarkup(row_width=1)
                    markup.add(
                        types.InlineKeyboardButton(
                            "⚡ Symlink", callback_data=f"share_final:{d[1]}:link"
                        ),
                        types.InlineKeyboardButton(
                            "🎬 Remux", callback_data=f"share_final:{d[1]}:remux"
                        ),
                        types.InlineKeyboardButton(
                            "❌ Annuler", callback_data="cancel"
                        ),
                    )
                    bot.edit_message_text(
                        f"🎯 **Options Partage**\n{len(subtitles)} sous-titres trouvés.",
                        call.message.chat.id,
                        call.message.message_id,
                        reply_markup=markup,
                    )
                else:
                    media_callback_router(
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
                    bot.send_message(
                        call.message.chat.id,
                        f"✅ **Partagé**\n\n🔗 Vidéo : `{data['video']}`",
                        parse_mode="Markdown",
                    )
                else:
                    bot.send_message(call.message.chat.id, "❌ Échec.")
            except:
                bot.send_message(call.message.chat.id, "❌ Erreur interne.")
        elif d[0] == "cancel":
            bot.edit_message_text(
                "❌ Fermé.", call.message.chat.id, call.message.message_id
            )

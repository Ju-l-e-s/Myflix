import json
import requests
import re
import logging
import os
from telebot.types import InlineKeyboardMarkup, InlineKeyboardButton
from config import *


def stream_gpt_json(query):
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {OPENAI_KEY}",
    }
    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {
                "role": "system",
                "content": "Media parser. Output JSON: {titre:str, type:movie|series}",
            },
            {"role": "user", "content": query},
        ],
        "response_format": {"type": "json_object"},
        "temperature": 0,
        "max_tokens": 100,
    }
    try:
        res = requests.post(
            "https://api.openai.com/v1/chat/completions",
            headers=headers,
            json=payload,
            timeout=10,
        )
        return res.json()["choices"][0]["message"]["content"]
    except Exception as e:
        logging.error(f"AI Search Error: {e}")
        return None


def register_search_handlers(bot, is_authorized):
    @bot.message_handler(commands=["get"])
    def handle_get(m):
        if not is_authorized(m.from_user.id):
            return
        parts = m.text.split(" ", 1)
        if len(parts) > 1:
            process_get_request(bot, m, parts[1])
        else:
            bot.reply_to(m, "🎬 Que veux-tu ? (Ex: /get Titanic)")

    @bot.message_handler(func=lambda m: True)
    def handle_ai_request(m):
        if not is_authorized(m.chat.id):
            return
        if m.text.startswith("/"):
            return
        process_get_request(bot, m, m.text)

    @bot.callback_query_handler(func=lambda call: call.data.startswith("dl_sel:"))
    def download_callback_handler(call):
        if not is_authorized(call.from_user.id):
            return
        d = call.data.split(":")
        cat, idx = d[1], int(d[2])
        try:
            with open(CONTEXT_FILE, "r") as f:
                ctx = json.load(f)
            search_data = ctx.get(f"{call.message.chat.id}_search")
            if not search_data:
                bot.answer_callback_query(call.id, "❌ Session expirée.")
                return

            item = search_data["results"][idx]
            cfg = RADARR_CFG if cat == "movie" else SONARR_CFG
            endpoint = "movie" if cat == "movie" else "series"
            id_key = "tmdbId" if cat == "movie" else "tvdbId"

            payload = {
                "title": item["title"],
                "qualityProfileId": 1,
                "titleSlug": item["titleSlug"],
                "images": item["images"],
                "rootFolderPath": "/movies" if cat == "movie" else "/tv",
                "monitored": True,
                "addOptions": {"searchForMovie": True}
                if cat == "movie"
                else {"searchForMissingEpisodes": True},
            }
            payload[id_key] = item[id_key]
            if cat == "series":
                payload["languageProfileId"] = 1

            r = requests.post(
                f"{cfg['url']}/api/v3/{endpoint}",
                headers={"X-Api-Key": cfg["key"]},
                json=payload,
                timeout=10,
            )
            if r.status_code in [200, 201]:
                bot.edit_message_text(
                    f"✅ **Ajouté :** `{item['title']} ({item.get('year')})`",
                    call.message.chat.id,
                    call.message.message_id,
                    parse_mode="Markdown",
                )
            else:
                bot.answer_callback_query(call.id, f"❌ Erreur {r.status_code}")
        except Exception as e:
            logging.error(f"DL Error: {e}")
            bot.answer_callback_query(call.id, "❌ Erreur interne.")


def process_get_request(bot, m, query):
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
        ctx = {}
        if os.path.exists(CONTEXT_FILE):
            try:
                ctx = json.load(open(CONTEXT_FILE))
            except:
                ctx = {}
        ctx[f"{m.chat.id}_search"] = {"cat": cat, "results": results[:5]}
        with open(CONTEXT_FILE, "w") as f:
            json.dump(ctx, f)
        text = f"🔍 **Résultats pour '{titre}'**\n\n"
        limit = min(len(results), 5)
        for i, res in enumerate(results[:limit], 1):
            text += f"{i}. {res['title']} ({res.get('year', 'N/A')})\n"
        markup = InlineKeyboardMarkup()
        btns = [
            InlineKeyboardButton(str(i), callback_data=f"dl_sel:{cat}:{i - 1}")
            for i in range(1, limit + 1)
        ]
        markup.add(*btns)
        bot.send_message(m.chat.id, text, reply_markup=markup, parse_mode="Markdown")
    except:
        bot.send_message(m.chat.id, "❌ Erreur.")

import os
import shutil
import json
from telebot import types
from config import *


def save_users(users):
    with open(USERS_FILE, "w") as f:
        json.dump({"allowed": users}, f)


def register_admin_handlers(bot, is_authorized):
    @bot.message_handler(commands=["status"])
    def status_cmd(m):
        if not is_authorized(m.from_user.id):
            return
        fmt = lambda u: (
            f"{u.used / (1024**3):.1f}/{u.total / (1024**3):.1f}Go ({int(u.used / u.total * 100)}%)"
        )
        try:
            nvme = shutil.disk_usage("/")
            msg = f"📊 **Statut Stockage**\n\n🚀 **NVMe** : `{fmt(nvme)}`"
            if os.path.exists("/mnt/externe"):
                hdd = shutil.disk_usage("/mnt/externe")
                msg += f"\n📚 **HDD** : `{fmt(hdd)}`"
            else:
                msg += "\n⚠️ **HDD** : `Non détecté sur /mnt/externe`"
            bot.reply_to(m, msg, parse_mode="Markdown")
        except Exception as e:
            bot.reply_to(m, f"❌ Erreur lecture disques : {e}")

    @bot.message_handler(commands=["admin"])
    def admin_menu(m):
        if not is_authorized(m.from_user.id):
            return
        markup = types.InlineKeyboardMarkup(row_width=1)
        markup.add(
            types.InlineKeyboardButton("📂 Partage", callback_data="adm:share"),
            types.InlineKeyboardButton("👥 Utilisateurs", callback_data="adm:users"),
            types.InlineKeyboardButton("🧹 Nettoyage SSD", callback_data="adm:clean"),
            types.InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"),
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
            with open(USERS_FILE, "r") as f:
                users = json.load(f)["allowed"]
            if uid not in users:
                users.append(uid)
                save_users(users)
                bot.reply_to(m, f"✅ ID {uid} ajouté.")
            else:
                bot.reply_to(m, "ℹ️ Déjà autorisé.")
        except:
            bot.reply_to(m, "❌ ID invalide.")

    @bot.callback_query_handler(func=lambda call: call.data.startswith("adm:"))
    def admin_callback_router(call):
        if not is_authorized(call.from_user.id):
            return
        d = call.data.split(":")
        cmd = d[1]

        if cmd == "main":
            markup = types.InlineKeyboardMarkup(row_width=1)
            markup.add(
                types.InlineKeyboardButton("📂 Partage", callback_data="adm:share"),
                types.InlineKeyboardButton(
                    "👥 Utilisateurs", callback_data="adm:users"
                ),
                types.InlineKeyboardButton(
                    "🧹 Nettoyage SSD", callback_data="adm:clean"
                ),
                types.InlineKeyboardButton("🔙 Fermer", callback_data="adm:close"),
            )
            bot.edit_message_text(
                "🛠️ **Menu Administrateur**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "share":
            markup = types.InlineKeyboardMarkup(row_width=1)
            markup.add(
                types.InlineKeyboardButton(
                    "✅ Contenu possédé", callback_data="adm:share_owned"
                ),
                types.InlineKeyboardButton(
                    "❌ Contenu non possédé", callback_data="adm:share_new"
                ),
                types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"),
            )
            bot.edit_message_text(
                "📂 **Gestion du Partage**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users":
            markup = types.InlineKeyboardMarkup(row_width=1)
            markup.add(
                types.InlineKeyboardButton("📋 Lister", callback_data="adm:users_list"),
                types.InlineKeyboardButton("➕ Ajouter", callback_data="adm:users_add"),
                types.InlineKeyboardButton(
                    "❌ Révoquer", callback_data="adm:users_rev"
                ),
                types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:main"),
            )
            bot.edit_message_text(
                "👥 **Gestion des Utilisateurs**",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users_list":
            with open(USERS_FILE, "r") as f:
                users = json.load(f)["allowed"]
            text = "📋 **Utilisateurs :**\n\n" + "\n".join([f"• `{u}`" for u in users])
            bot.edit_message_text(
                text,
                call.message.chat.id,
                call.message.message_id,
                reply_markup=types.InlineKeyboardMarkup().add(
                    types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:users")
                ),
                parse_mode="Markdown",
            )
        elif cmd == "users_add":
            msg = bot.send_message(call.message.chat.id, "👤 ID Telegram :")
            bot.register_next_step_handler(msg, process_user_add)
        elif cmd == "users_rev":
            with open(USERS_FILE, "r") as f:
                users = json.load(f)["allowed"]
            markup = types.InlineKeyboardMarkup(row_width=1)
            for u in users:
                if u == SUPER_ADMIN:
                    continue
                markup.add(
                    types.InlineKeyboardButton(
                        f"❌ {u}", callback_data=f"adm:users_del:{u}"
                    )
                )
            markup.add(
                types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:users")
            )
            bot.edit_message_text(
                "Sélectionnez l'ID à révoquer :",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "users_del":
            uid = int(d[2])
            with open(USERS_FILE, "r") as f:
                users = json.load(f)["allowed"]
            if uid in users:
                users.remove(uid)
                save_users(users)
                bot.answer_callback_query(call.id, "✅ Révoqué.")
            admin_callback_router(
                types.CallbackQuery(
                    id=call.id,
                    from_user=call.from_user,
                    message=call.message,
                    data="adm:users",
                    chat_instance=call.chat_instance,
                )
            )
        elif cmd == "share_owned":
            markup = types.InlineKeyboardMarkup(row_width=2)
            markup.add(
                types.InlineKeyboardButton(
                    "🎬 Films", callback_data="adm:list_owned:movies"
                ),
                types.InlineKeyboardButton(
                    "📺 Séries", callback_data="adm:list_owned:series"
                ),
                types.InlineKeyboardButton("⬅️ Retour", callback_data="adm:share"),
            )
            bot.edit_message_text(
                "Catégorie :",
                call.message.chat.id,
                call.message.message_id,
                reply_markup=markup,
            )
        elif cmd == "list_owned":
            from handlers_media import list_media_unified

            list_media_unified(bot, call.message, d[2], "Partage", True)
        elif cmd == "close":
            bot.delete_message(call.message.chat.id, call.message.message_id)

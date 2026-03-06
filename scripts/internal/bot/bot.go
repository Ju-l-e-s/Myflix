package bot

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"myflixbot.local/internal/ai"
	"myflixbot.local/internal/arrclient"
	"myflixbot.local/internal/config"
	"myflixbot.local/internal/share"
	"myflixbot.local/internal/system"
	"myflixbot.local/vpnmanager"

	tele "gopkg.in/telebot.v3"
)

type BotHandler struct {
	cfg      *config.Config
	bot      *tele.Bot
	arr      *arrclient.ArrClient
	ai       *ai.GeminiClient
	sys      *system.SystemManager
	vpn      *vpnmanager.Manager
	share    *share.ShareEngine
	reClean  *regexp.Regexp
	reSpaces *regexp.Regexp
	reTMDB   *regexp.Regexp
	pathMap  map[string]string // Mapping for short path keys
	pathMu   sync.RWMutex
}

func NewBotHandler(b *tele.Bot, cfg *config.Config, arr *arrclient.ArrClient, aiClient *ai.GeminiClient, sys *system.SystemManager, vpn *vpnmanager.Manager, shareSrv *share.ShareEngine) (*BotHandler, error) {
	return &BotHandler{
		cfg:      cfg,
		bot:      b,
		arr:      arr,
		ai:       aiClient,
		sys:      sys,
		vpn:      vpn,
		share:    shareSrv,
		reClean:  regexp.MustCompile(`(?i)(?:1080p|720p|4k|uhd|x26[45]|h26[45]|web[- ]?(dl|rip)|bluray|aac|dd[p]?5\.1|atmos|repack|playweb|max|[\d\s]+A M|[\d\s]+P M|-NTb|-playWEB)`),
		reSpaces: regexp.MustCompile(`\s+`),
		reTMDB:   regexp.MustCompile(`^(?i)(série|serie|film)?\s*(.+?)\s*(\b(19|20)\d{2}\b)?$`),
		pathMap:  make(map[string]string),
	}, nil
}

func (h *BotHandler) GetBot() *tele.Bot { return h.bot }
func (h *BotHandler) Start() { 
	h.setupHandlers()
	// Préchauffage du cache en arrière-plan
	go h.warmUpCache()
	h.bot.Start() 
}
func (h *BotHandler) Stop() { h.bot.Stop() }

func (h *BotHandler) warmUpCache() {
	ctx := context.Background()
	h.arr.RefreshLibrary(ctx, "films")
	h.arr.RefreshLibrary(ctx, "series")
}

func (h *BotHandler) authMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Sender().ID != h.cfg.SuperAdmin {
			slog.Warn("Tentative d'accès non autorisée", "user_id", c.Sender().ID, "username", c.Sender().Username)
			return nil // On ignore silencieusement les autres utilisateurs
		}
		return next(c)
	}
}

func (h *BotHandler) setupHandlers() {
	// Middleware d'autorisation global pour toutes les commandes
	admin := h.bot.Group()
	admin.Use(h.authMiddleware)

	admin.Handle("/start", h.handleStart)
	admin.Handle("/films", func(c tele.Context) error { return h.showLibrary(c, "films", 0, false) })
	admin.Handle("/series", func(c tele.Context) error { return h.showLibrary(c, "series", 0, false) })
	admin.Handle("/status", h.handleStatus)
	admin.Handle("/vpn", func(c tele.Context) error { return h.showVpnStatus(c, false) })
	admin.Handle("/queue", func(c tele.Context) error { return h.refreshQueue(c, false) })
	admin.Handle("/maintenance_test", func(c tele.Context) error {
		c.Send("🚀 <b>Test manuel de la maintenance nocturne initié...</b>", tele.ModeHTML)
		go h.sys.ExecuteMaintenance()
		go h.vpn.RotateVPN()
		return nil
	})

	// Handlers pour le clavier fixe (Reply Keyboard)
	admin.Handle("🎬 Films", func(c tele.Context) error { return h.showLibrary(c, "films", 0, false) })
	admin.Handle("📺 Séries", func(c tele.Context) error { return h.showLibrary(c, "series", 0, false) })
	admin.Handle("📥 Flux", func(c tele.Context) error { return h.refreshQueue(c, false) })
	admin.Handle("📊 Système", h.handleStatus)
	admin.Handle("🛡️ VPN", func(c tele.Context) error { return h.showVpnStatus(c, false) })
	admin.Handle("❓ Aide", func(c tele.Context) error {
		return c.Send("🔍 <b>COMMENT RECHERCHER ?</b>\n\nC'est très simple : <b>tapez directement le nom</b> d'un film ou d'une série dans le chat (ex: <i>Inception</i>).\n\nL'IA analysera votre demande et cherchera dans votre bibliothèque ou sur internet pour vous proposer l'ajout.", tele.ModeHTML)
	})

	admin.Handle(tele.OnText, h.handleText)

	// Callback Handlers (Boutons)
	h.bot.Handle(&tele.Btn{Unique: "lib"}, h.authMiddleware(func(c tele.Context) error { return h.showLibrary(c, c.Args()[0], 0, true) }))
	h.bot.Handle(&tele.Btn{Unique: "lib_page"}, h.authMiddleware(func(c tele.Context) error {
		p, _ := strconv.Atoi(c.Args()[1])
		return h.showLibrary(c, c.Args()[0], p, true)
	}))
	h.bot.Handle(&tele.Btn{Unique: "q_refresh"}, h.authMiddleware(func(c tele.Context) error { return h.refreshQueue(c, true) }))
	h.bot.Handle(&tele.Btn{Unique: "status_refresh"}, h.authMiddleware(h.handleStart))
	h.bot.Handle(&tele.Btn{Unique: "sys_status"}, h.authMiddleware(h.handleStatus))
	h.bot.Handle(&tele.Btn{Unique: "vpn_refresh"}, h.authMiddleware(func(c tele.Context) error { return h.showVpnStatus(c, true) }))
	h.bot.Handle(&tele.Btn{Unique: "vpn_rotate"}, h.authMiddleware(h.handleVpnRotate))
	h.bot.Handle(&tele.Btn{Unique: "m_sel"}, h.authMiddleware(h.handleSelection))
	h.bot.Handle(&tele.Btn{Unique: "m_del"}, h.authMiddleware(h.handleDelete))
	h.bot.Handle(&tele.Btn{Unique: "m_share"}, h.authMiddleware(h.handleShare))
	h.bot.Handle(&tele.Btn{Unique: "dl_add"}, h.authMiddleware(h.handleAdd))

	// Nouveaux handlers de stockage
	h.bot.Handle(&tele.Btn{Unique: "browse_storage"}, h.authMiddleware(func(c tele.Context) error {
		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("🚀 NVMe", "show_files", "nvme", "0"), menu.Data("📚 HDD", "show_files", "hdd", "0")),
			menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")),
		)
		return c.Edit("📂 <b>CHOIX DU STOCKAGE</b>\n\nQuel tier souhaitez-vous explorer ?", menu, tele.ModeHTML)
	}))

	h.bot.Handle(&tele.Btn{Unique: "show_files"}, h.authMiddleware(func(c tele.Context) error {
		return h.handleShowFiles(c)
	}))

	h.bot.Handle(&tele.Btn{Unique: "file_confirm"}, h.authMiddleware(func(c tele.Context) error {
		pathKey := c.Args()[1]
		h.pathMu.RLock()
		fullPath := h.pathMap[pathKey]
		h.pathMu.RUnlock()

		if fullPath == "" { return c.Edit("❌ Référence de fichier expirée.", tele.ModeHTML) }

		menu := &tele.ReplyMarkup{}
		menu.Inline(
			menu.Row(menu.Data("🔥 CONFIRMER", "file_delete", c.Args()[0], pathKey)),
			menu.Row(menu.Data("❌ ANNULER", "show_files", c.Args()[0], "0")),
		)
		return c.Edit(fmt.Sprintf("⚠️ <b>CONFIRMATION REQUISE</b>\n\nVoulez-vous vraiment supprimer définitivement :\n<code>%s</code> ?", filepath.Base(fullPath)), menu, tele.ModeHTML)
	}))

	h.bot.Handle(&tele.Btn{Unique: "file_delete"}, h.authMiddleware(func(c tele.Context) error {
		pathKey := c.Args()[1]
		h.pathMu.RLock()
		fullPath := h.pathMap[pathKey]
		h.pathMu.RUnlock()

		if fullPath == "" { return c.Send("❌ Référence expirée.") }

		if err := h.sys.DeletePath(fullPath); err != nil {
			return c.Send("❌ Erreur suppression : " + err.Error())
		}
		c.Send("🗑 <b>Fichier supprimé avec succès.</b>", tele.ModeHTML)
		return h.handleShowFiles(c) // Refresh
	}))
}

func (h *BotHandler) handleShowFiles(c tele.Context) error {
	tier := c.Args()[0]
	page, _ := strconv.Atoi(c.Args()[1])
	files, err := h.sys.ListStorageFiles(tier)
	if err != nil { return c.Edit("❌ Erreur accès disque.", tele.ModeHTML) }
	
	pageSize := 8
	start := page * pageSize
	if start >= len(files) { start, page = 0, 0 }
	end := start + pageSize
	if end > len(files) { end = len(files) }

	msg := fmt.Sprintf("📂 <b>FICHIERS (%s)</b> [%d]\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n", strings.ToUpper(tier), len(files))
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row

	h.pathMu.Lock()
	for i, f := range files[start:end] {
		// On crée une clé courte pour le chemin
		pathKey := fmt.Sprintf("%x", sha256.Sum256([]byte(f.Path)))[:8]
		h.pathMap[pathKey] = f.Path
		
		msg += fmt.Sprintf("%d. <code>%s</code> (%.1f GB)\n", i+1, f.Name, f.Size)
		rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("%d. 🗑 Supprimer", i+1), "file_confirm", tier, pathKey)))
	}
	h.pathMu.Unlock()

	nav := []tele.Btn{}
	if page > 0 { nav = append(nav, menu.Data("⬅️", "show_files", tier, strconv.Itoa(page-1))) }
	if end < len(files) { nav = append(nav, menu.Data("➡️", "show_files", tier, strconv.Itoa(page+1))) }
	if len(nav) > 0 { rows = append(rows, menu.Row(nav...)) }
	
	rows = append(rows, menu.Row(menu.Data("⬅️ Retour", "browse_storage")))
	menu.Inline(rows...)
	return c.Edit(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) handleStart(c tele.Context) error {
	msg := "🏛 <b>MYFLIX : CENTRE DE CONTRÔLE</b>\n\nBienvenue dans votre interface de gestion multimédia.\n\n👉 <i>Tapez le nom d'un média pour le rechercher ou utilisez les boutons ci-dessous.</i>"
	
	// On envoie le clavier fixe (Reply Keyboard)
	menu := h.buildReplyKeyboard()
	
	// Optionnel: on peut aussi envoyer un menu inline en même temps si on veut
	// mais pour la clarté, le clavier fixe suffit souvent au début.
	return c.Send(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) handleStatus(c tele.Context) error {
	msg := h.sys.GetStorageStatus()
	
	menu := &tele.ReplyMarkup{}
	btnBrowse := menu.Data("📂 Gérer les fichiers", "browse_storage")
	btnRefresh := menu.Data("🔄 Actualiser", "sys_status")
	btnMenu := menu.Data("🏠 Menu Principal", "status_refresh")
	
	menu.Inline(
		menu.Row(btnBrowse),
		menu.Row(btnRefresh, btnMenu),
	)

	if c.Callback() != nil {
		return c.Edit(msg, menu, tele.ModeHTML)
	}
	return c.Send(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) buildReplyKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{ResizeKeyboard: true}
	
	btnF := menu.Text("🎬 Films")
	btnS := menu.Text("📺 Séries")
	btnQ := menu.Text("📥 Flux")
	btnSt := menu.Text("📊 Système")
	btnVpn := menu.Text("🛡️ VPN")
	btnHelp := menu.Text("❓ Aide")
	
	menu.Reply(
		menu.Row(btnF, btnS),
		menu.Row(btnQ, btnSt, btnVpn),
		menu.Row(btnHelp),
	)
	return menu
}

func (h *BotHandler) showLibrary(c tele.Context, cat string, page int, edit bool) error {
	items, expired := h.arr.GetCachedLibrary(cat)
	if expired || items == nil { go h.arr.RefreshLibrary(context.Background(), cat) }
	if items == nil { return c.Send("⏳ <i>Synchronisation de la bibliothèque...</i>", tele.ModeHTML) }

	// On ne filtre plus, on affiche tout mais avec un statut
	pageSize := 10
	start, end := page*pageSize, (page+1)*pageSize
	if start >= len(items) { start, page = 0, 0 }
	if end > len(items) { end = len(items) }

	titleLabel := "MES FILMS"
	if cat == "series" { titleLabel = "MES SÉRIES" }

	msg := fmt.Sprintf("📂 <b>%s</b> (%d)\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n", titleLabel, len(items))
	menu := &tele.ReplyMarkup{}
	var btns []tele.Btn
	for i, it := range items[start:end] {
		ready := false
		if cat == "films" { ready, _ = it["hasFile"].(bool) } else {
			if st, ok := it["statistics"].(map[string]interface{}); ok {
				if ec, _ := st["episodeFileCount"].(float64); ec > 0 { ready = true }
			}
		}

		status := "⏳"; if ready { status = "✅" }
		title, _ := it["title"].(string)
		year := 0
		if y, ok := it["year"].(float64); ok { year = int(y) }
		
		// Utilisation de i+1 pour avoir 1️⃣ à 🔟 sur chaque page
		msg += h.formatMediaLine(i+1, title, year, status) + "\n"
		btns = append(btns, menu.Data(strconv.Itoa(i+1), "m_sel", cat, fmt.Sprintf("%v", it["id"])))
	}

	var rows []tele.Row
	for i := 0; i < len(btns); i += 5 {
		l := i+5; if l > len(btns) { l = len(btns) }
		rows = append(rows, menu.Row(btns[i:l]...))
	}
	
	nav := []tele.Btn{menu.Data("🏠 Menu", "status_refresh")}
	if page > 0 { nav = append(nav, menu.Data("⬅️", "lib_page", cat, strconv.Itoa(page-1))) }
	if end < len(items) { nav = append(nav, menu.Data("➡️", "lib_page", cat, strconv.Itoa(page+1))) }
	rows = append(rows, menu.Row(nav...))
	menu.Inline(rows...)

	msg += "⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n"
	msg += "<code>✅ Prêt  ⏳ En attente</code>"

	if edit { return c.Edit(msg, menu, tele.ModeHTML) }
	return c.Send(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) handleText(c tele.Context) error {
	query := c.Text()
	if strings.HasPrefix(query, "/") || len(query) < 3 { return nil }

	c.Send("🔍 <i>Analyse de votre requête par l'IA...</i>", tele.ModeHTML)

	local := h.arr.SearchLocalCache(query)
	ext := h.tmdbSmartResolve(query)
	results := local
	for _, r := range ext {
		found := false
		for _, l := range local { if l["title"] == r["title"] { found = true; break } }
		if !found { results = append(results, r) }
	}

	if len(results) == 0 { return c.Send("❌ <b>Aucun résultat trouvé.</b>\nEssayez d'être plus spécifique (ex: Nom du film + année).", tele.ModeHTML) }
	
	text := fmt.Sprintf("🎯 <b>RÉSULTATS DE RECHERCHE</b>\n<i>Query: %s</i>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n", html.EscapeString(query))
	menu := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, res := range results {
		if i >= 6 { break }
		icon := "🎬"; if res["type"] == "tv" || res["type"] == "series" { icon = "📺" }
		status := "📥"
		if res["is_local"] == true {
			status = "⏳"
			if ready, ok := res["is_ready"].(bool); ok && ready {
				status = "✅"
			}
		}
		
		year := 0
		if y, ok := res["year"].(string); ok {
			fmt.Sscanf(y, "%d", &year)
		} else if y, ok := res["year"].(float64); ok {
			year = int(y)
		}

		title := res["title"].(string)
		text += fmt.Sprintf("%s %s\n", icon, h.formatMediaLine(i+1, title, year, status))
		
		if res["is_local"] == true {
			rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("%d", i+1), "m_sel", res["type"].(string), "0")))
		} else {
			rows = append(rows, menu.Row(menu.Data(fmt.Sprintf("%d", i+1), "dl_add", res["type"].(string), fmt.Sprintf("%v", res["tmdb_id"]))))
		}
	}
	rows = append(rows, menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")))
	menu.Inline(rows...)
	return c.Send(text, menu, tele.ModeHTML)
}

func (h *BotHandler) handleSelection(c tele.Context) error {
	cat, id := c.Args()[0], c.Args()[1]
	items, _ := h.arr.GetCachedLibrary(cat)
	for _, it := range items {
		if fmt.Sprintf("%v", it["id"]) == id {
			return h.sendDetailedMedia(c, cat, it)
		}
	}
	return c.Send("❌ <b>Contenu introuvable.</b>\nLa bibliothèque a peut-être été mise à jour entre-temps.", tele.ModeHTML)
}

func (h *BotHandler) handleDelete(c tele.Context) error {
	cat := c.Args()[0]
		if err := h.arr.DeleteItem(context.Background(), cat, c.Args()[1]); err != nil {
	 
		return c.Send("❌ <b>Échec de la suppression.</b>\nLe service est temporairement indisponible.", tele.ModeHTML) 
	}
	// Force refresh cache immediately
	go h.arr.RefreshLibrary(context.Background(), cat)
	
	c.Send("🗑 <b>Fichier retiré de la bibliothèque.</b>", tele.ModeHTML)
	return h.showLibrary(c, cat, 0, true)
}

func (h *BotHandler) handleShare(c tele.Context) error {
	cat, id := c.Args()[0], c.Args()[1]
	items, _ := h.arr.GetCachedLibrary(cat)
	for _, it := range items {
		if fmt.Sprintf("%v", it["id"]) == id {
			path, _ := it["path"].(string)
			base := filepath.Base(path)
			mount := h.cfg.MoviesMount; if cat == "series" { mount = h.cfg.TvMount }
			finalPath := h.findFirstVideo(filepath.Join(mount, base))
			if finalPath == "" { return c.Send("❌ <b>Erreur : Fichier vidéo introuvable.</b>", tele.ModeHTML) }
			
			link := h.share.GenerateLink(finalPath)
			msg := fmt.Sprintf("🔗 <b>Lien de Partage Généré</b>\n\n🎬 <b>%s</b>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n<code>%s</code>\n\n⚠️ <i>Lien actif tant que le serveur est en ligne.</i>", it["title"], link)
			return c.Send(msg, tele.ModeHTML)
		}
	}
	return nil
}

func (h *BotHandler) handleAdd(c tele.Context) error {
	mType, id := c.Args()[0], c.Args()[1]
	results, err := h.arr.Lookup(context.Background(), mType, id)
	if err != nil || len(results) == 0 { return c.Send("❌ <b>Erreur de communication avec le serveur d'ajout.</b>", tele.ModeHTML) }
	
	item := results[0]
	if exID, ok := item["id"].(float64); ok && exID > 0 { return c.Send("✅ <b>Contenu déjà présent dans la bibliothèque.</b>", tele.ModeHTML) }
	
	item["monitored"] = true
	item["qualityProfileId"] = 1
	item["rootFolderPath"] = "/movies"; if mType != "movie" { 
		item["rootFolderPath"] = "/tv"
		item["languageProfileId"] = 1
		item["addOptions"] = map[string]interface{}{"searchForMissingEpisodes": true}
	}
	
	if err := h.arr.AddItem(context.Background(), mType, item); err != nil { return c.Send("❌ <b>L'ajout a échoué.</b>\nVérifiez les journaux du serveur.", tele.ModeHTML) }
	
	msg := fmt.Sprintf("🚀 <b>TÉLÉCHARGEMENT INITIÉ</b>\n\n🎬 %s\nLe contenu sera disponible dans votre bibliothèque dès la fin du flux.", item["title"])
	return c.Edit(msg, tele.ModeHTML)
}

func (h *BotHandler) renderProgressBar(progress float64) string {
	const width = 10
	filled := int(progress / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}
	return fmt.Sprintf("<code>[%s] %.1f%%</code>", bar, progress)
}

func (h *BotHandler) refreshQueue(c tele.Context, edit bool) error {
	req, _ := http.NewRequest("GET", h.cfg.QbitURL+"/api/v2/torrents/info", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return c.Send("❌ <b>Service qBittorrent inaccessible.</b>", tele.ModeHTML)
	}
	defer resp.Body.Close()

	var torrents []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&torrents)

	text := "📥 <b>FLUX DE TÉLÉCHARGEMENT ACTIFS</b>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n"
	count := 0
	for _, t := range torrents {
		st := t["state"].(string)
		// On affiche les téléchargements actifs ou en cours de vérification
		if strings.Contains(strings.ToLower(st), "dl") || strings.Contains(strings.ToLower(st), "check") {
			prog := t["progress"].(float64) * 100
			cat, _ := t["category"].(string)

			icon := "📦"
			if cat == "radarr" {
				icon = "🎬"
			} else if cat == "sonarr" {
				icon = "📺"
			}

			text += fmt.Sprintf("%s <b>%s</b>\n  %s\n\n", icon, h.cleanTitle(t["name"].(string)), h.renderProgressBar(prog))
			count++
		}
	}
	if count == 0 {
		text += "<i>Aucun flux actif pour le moment.</i>"
	}

	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔄 Actualiser", "q_refresh"), menu.Data("🏠 Menu Principal", "status_refresh")))

	if edit {
		return c.Edit(text, menu, tele.ModeHTML)
	}
	return c.Send(text, menu, tele.ModeHTML)
}

func (h *BotHandler) showVpnStatus(c tele.Context, edit bool) error {
        ip := h.vpn.GetCurrentIP()
        
        // Tentative de récupération du pays
        flag := "🏳️"
        countryName := "Inconnu"
        
        resp, err := http.Get("http://ip-api.com/json/" + ip)
        if err == nil {
                defer resp.Body.Close()
                var geo struct {
                        CountryCode string `json:"countryCode"`
                        Country     string `json:"country"`
                }
                if json.NewDecoder(resp.Body).Decode(&geo) == nil {
                        countryName = geo.Country
                        // Conversion du code pays ISO en emoji drapeau
                        if len(geo.CountryCode) == 2 {
                                flag = ""
                                for _, r := range strings.ToUpper(geo.CountryCode) {
                                        flag += string(rune(r + 0x1F1A5))
                                }
                        }
                }
        }

        msg := fmt.Sprintf("🛡️ <b>PROTECTION VPN & INFRASTRUCTURE</b>\n\n🌍 Adresse IP : <code>%s</code> (%s %s)\n🛰 Statut : Protégé ✅", ip, flag, countryName)
        menu := &tele.ReplyMarkup{}
        menu.Inline(menu.Row(menu.Data("🔄 Rotation Manuelle", "vpn_rotate"), menu.Data("🔄 Refresh", "vpn_refresh")), menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")))
        if edit { return c.Edit(msg, menu, tele.ModeHTML) }
        return c.Send(msg, menu, tele.ModeHTML)
}
func (h *BotHandler) handleVpnRotate(c tele.Context) error {
	go h.vpn.RotateVPN(); return c.Send("🔄 <b>Rotation VPN initiée.</b>\nLe serveur sera redémarré avec une nouvelle IP.", tele.ModeHTML)
}

func (h *BotHandler) tmdbSmartResolve(query string) []map[string]interface{} {
	m := h.reTMDB.FindStringSubmatch(strings.TrimSpace(query))
	aiData := map[string]interface{}{"title": query, "type": "", "year": 0}
	if len(m) > 0 && m[2] != "" {
		if strings.Contains(strings.ToLower(m[1]), "ser") { aiData["type"] = "tv" } else if m[1] != "" { aiData["type"] = "movie" }
		aiData["title"] = m[2]
		if m[3] != "" { y, _ := strconv.Atoi(m[3]); aiData["year"] = y }
	} else {
		res := h.ai.StreamGptJson(context.Background(), query)
		json.Unmarshal([]byte(res), &aiData)
	}

	searchTitle := fmt.Sprintf("%v", aiData["title"])
	if searchTitle == "" { searchTitle = query }
	searchType := fmt.Sprintf("%v", aiData["type"])

	var results []map[string]interface{}
	
	// 1. Recherche Films (via Radarr pour avoir le runtime)
	if searchType == "" || searchType == "movie" {
		movies, err := h.arr.LookupByTerm(context.Background(), "movie", searchTitle)
		if err == nil {
			for _, m := range movies {
				// Exclusion des courts-métrages (runtime < 40 mins)
				runtime, _ := m["runtime"].(float64)
				if runtime > 0 && runtime < 40 { continue }
				
				year := 0
				if y, ok := m["year"].(float64); ok { year = int(y) }
				
				results = append(results, map[string]interface{}{
					"tmdb_id":  int(m["tmdbId"].(float64)),
					"title":    m["title"].(string),
					"year":     fmt.Sprintf("%d", year),
					"type":     "movie",
					"is_local": false,
				})
				if len(results) >= 5 { break }
			}
		}
	}

	// 2. Recherche Séries (via Sonarr)
	if (searchType == "" || searchType == "tv") && len(results) < 5 {
		series, err := h.arr.LookupByTerm(context.Background(), "series", searchTitle)
		if err == nil {
			for _, s := range series {
				year := 0
				if y, ok := s["year"].(float64); ok { year = int(y) }

				results = append(results, map[string]interface{}{
					"tmdb_id":  int(s["tvdbId"].(float64)), // Sonarr utilise tvdbId par défaut pour l'ID principal
					"title":    s["title"].(string),
					"year":     fmt.Sprintf("%d", year),
					"type":     "tv",
					"is_local": false,
				})
				if len(results) >= 5 { break }
			}
		}
	}

	return results
}

func (h *BotHandler) cleanTitle(t string) string {
	c := h.reClean.Split(t, -1)[0]
	return strings.TrimSpace(h.reSpaces.ReplaceAllString(strings.ReplaceAll(c, ".", " "), " "))
}

func (h *BotHandler) numberToEmoji(n int) string {
	digits := []string{"0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}
	if n == 10 { return "🔟" }
	if n < 10 { return digits[n] }
	res := ""; s := fmt.Sprintf("%d", n)
	for _, char := range s { res += digits[char-'0'] }
	return res
}

func (h *BotHandler) formatMediaLine(index int, title string, year int, status string) string {
        const targetWidth = 28 // Largeur maximale avant le statut

        fullTitle := title
        if year > 0 {
                fullTitle = fmt.Sprintf("%s (%d)", title, year)
        }

        prefix := h.numberToEmoji(index) + " "
        
        // Calcul de la largeur visuelle (approximation monospace Telegram)
        prefixWidth := 3 // Emoji (2) + Espace (1)
        if index > 10 { prefixWidth = 5 } // ex: 1️⃣1️⃣ (4) + Espace (1)

        text := fullTitle
        runes := []rune(text)
        var finalLines []string
        
        if len(runes)+prefixWidth > targetWidth {
                limit := targetWidth - prefixWidth
                if limit < 0 { limit = 0 }
                line1 := string(runes[:limit])
                line2 := string(runes[limit:])
                
                if len([]rune(line2)) > targetWidth-prefixWidth {
                    line2 = string([]rune(line2)[:targetWidth-prefixWidth-3]) + "..."
                }

                padding1 := strings.Repeat(" ", targetWidth - (prefixWidth + len([]rune(line1))))
                finalLines = append(finalLines, fmt.Sprintf("%s%s%s %s", prefix, line1, padding1, status))
                
                indent := strings.Repeat(" ", prefixWidth)
                paddingCount2 := targetWidth - (prefixWidth + len([]rune(line2)))
                if paddingCount2 < 0 { paddingCount2 = 0 }
                padding2 := strings.Repeat(" ", paddingCount2)
                finalLines = append(finalLines, fmt.Sprintf("%s%s%s", indent, line2, padding2))
        } else {
                padding := strings.Repeat(" ", targetWidth - (prefixWidth + len(runes)))
                finalLines = append(finalLines, fmt.Sprintf("%s%s%s %s", prefix, text, padding, status))
        }

        return "<code>" + strings.Join(finalLines, "\n") + "</code>"
}
func (h *BotHandler) findFirstVideo(root string) string {
	var found string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(p))
			if ext == ".mkv" || ext == ".mp4" || ext == ".avi" { found = p; return fmt.Errorf("done") }
		}
		return nil
	})
	return found
}

func (h *BotHandler) sendDetailedMedia(c tele.Context, cat string, it map[string]interface{}) error {
	icon := "🎬"
	if cat == "series" || cat == "tv" {
		icon = "📺"
	}
	
	sizeStr := ""
	var size float64
	if s, ok := it["sizeOnDisk"].(float64); ok {
		size = s
	} else if stats, ok := it["statistics"].(map[string]interface{}); ok {
		if s, ok := stats["sizeOnDisk"].(float64); ok {
			size = s
		}
	}

	if size > 0 {
		sizeStr = fmt.Sprintf("\n⚖️ Taille : %.1f GB", size / (1024*1024*1024))
	}

	msg := fmt.Sprintf("%s <b>%s</b> (%v)\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n💾 Emplacement : Stockage Rapide (NVMe)%s\n\nQue souhaitez-vous faire ?", 
		icon, it["title"], it["year"], sizeStr)
	
	menu := &tele.ReplyMarkup{}
	btnShare := menu.Data("🔗 Partager", "m_share", cat, fmt.Sprintf("%v", it["id"]))
	btnDelete := menu.Data("🗑 Supprimer", "m_del", cat, fmt.Sprintf("%v", it["id"]))
	menu.Inline(menu.Row(btnShare, btnDelete), menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")))
	
	return c.Send(msg, menu, tele.ModeHTML)
}


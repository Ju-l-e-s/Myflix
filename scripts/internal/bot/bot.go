package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

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
}

func NewBotHandler(cfg *config.Config, arr *arrclient.ArrClient, aiClient *ai.GeminiClient, sys *system.SystemManager, vpn *vpnmanager.Manager, shareSrv *share.ShareEngine) (*BotHandler, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.Token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil { return nil, err }

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
	}, nil
}

func (h *BotHandler) GetBot() *tele.Bot { return h.bot }
func (h *BotHandler) Start() { h.setupHandlers(); h.bot.Start() }
func (h *BotHandler) Stop() { h.bot.Stop() }

func (h *BotHandler) setupHandlers() {
	h.bot.Handle("/start", h.handleStart)
	h.bot.Handle("/films", func(c tele.Context) error { return h.showLibrary(c, "films", 0, false) })
	h.bot.Handle("/series", func(c tele.Context) error { return h.showLibrary(c, "series", 0, false) })
	h.bot.Handle("/status", h.handleStatus)
	h.bot.Handle("/vpn", func(c tele.Context) error { return h.showVpnStatus(c, false) })
	h.bot.Handle("/queue", func(c tele.Context) error { return h.refreshQueue(c, false) })

	h.bot.Handle(tele.OnText, h.handleText)

	// Callback Handlers
	h.bot.Handle(&tele.Btn{Unique: "lib"}, func(c tele.Context) error { return h.showLibrary(c, c.Args()[0], 0, true) })
	h.bot.Handle(&tele.Btn{Unique: "lib_page"}, func(c tele.Context) error {
		p, _ := strconv.Atoi(c.Args()[1])
		return h.showLibrary(c, c.Args()[0], p, true)
	})
	h.bot.Handle(&tele.Btn{Unique: "q_refresh"}, func(c tele.Context) error { return h.refreshQueue(c, true) })
	h.bot.Handle(&tele.Btn{Unique: "status_refresh"}, h.handleStart)
	h.bot.Handle(&tele.Btn{Unique: "sys_status"}, h.handleStatus)
	h.bot.Handle(&tele.Btn{Unique: "vpn_refresh"}, func(c tele.Context) error { return h.showVpnStatus(c, true) })
	h.bot.Handle(&tele.Btn{Unique: "vpn_rotate"}, h.handleVpnRotate)
	h.bot.Handle(&tele.Btn{Unique: "m_sel"}, h.handleSelection)
	h.bot.Handle(&tele.Btn{Unique: "m_del"}, h.handleDelete)
	h.bot.Handle(&tele.Btn{Unique: "m_share"}, h.handleShare)
	h.bot.Handle(&tele.Btn{Unique: "dl_add"}, h.handleAdd)
}

func (h *BotHandler) handleStart(c tele.Context) error {
	msg := "🏛 <b>MYFLIX : CENTRE DE CONTRÔLE</b>\n\nBienvenue dans votre interface de gestion multimédia. Que souhaitez-vous faire ?"
	return c.Send(msg, h.buildMainMenu(), tele.ModeHTML)
}

func (h *BotHandler) handleStatus(c tele.Context) error {
	return c.Edit(h.sys.GetStorageStatus(), h.buildMainMenu(), tele.ModeHTML)
}

func (h *BotHandler) buildMainMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnF := menu.Data("🎬 Bibliothèque Films", "lib", "films")
	btnS := menu.Data("📺 Bibliothèque Séries", "lib", "series")
	btnQ := menu.Data("📥 Flux Téléchargements", "q_refresh")
	btnSt := menu.Data("📊 État du Serveur", "sys_status")
	btnVpn := menu.Data("🛡️ Protection VPN", "vpn_refresh")
	
	menu.Inline(
		menu.Row(btnF, btnS),
		menu.Row(btnQ, btnSt),
		menu.Row(btnVpn),
	)
	return menu
}

func (h *BotHandler) showLibrary(c tele.Context, cat string, page int, edit bool) error {
	items, expired := h.arr.GetCachedLibrary(cat)
	if expired || items == nil { go h.arr.RefreshLibrary(context.Background(), cat) }
	if items == nil { return c.Send("⏳ <i>Synchronisation de la bibliothèque...</i>", tele.ModeHTML) }

	var filtered []map[string]interface{}
	for _, it := range items {
		ready := false
		if cat == "films" { ready, _ = it["hasFile"].(bool) } else {
			if st, ok := it["statistics"].(map[string]interface{}); ok {
				if ec, _ := st["episodeFileCount"].(float64); ec > 0 { ready = true }
			}
		}
		if ready { filtered = append(filtered, it) }
	}

	if len(filtered) == 0 { 
		return c.Send("📂 <b>La bibliothèque est vide.</b>\nUtilisez la recherche pour ajouter du contenu.", tele.ModeHTML)
	}

	start, end := page*10, (page+1)*10
	if start >= len(filtered) { start, page = 0, 0 }
	if end > len(filtered) { end = len(filtered) }

	titleLabel := "MES FILMS"
	if cat == "series" { titleLabel = "MES SÉRIES" }

	msg := fmt.Sprintf("📂 <b>%s</b> (%d)\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n", titleLabel, len(filtered))
	menu := &tele.ReplyMarkup{}
	var btns []tele.Btn
	for i, it := range filtered[start:end] {
		msg += fmt.Sprintf("%s <code>%s</code>\n", h.getIndexEmoji(i+1), h.formatTitle(it["title"].(string), it["year"], 22, 25))
		btns = append(btns, menu.Data(strconv.Itoa(i+1), "m_sel", cat, fmt.Sprintf("%v", it["id"])))
	}

	var rows []tele.Row
	for i := 0; i < len(btns); i += 5 {
		l := i+5; if l > len(btns) { l = len(btns) }
		rows = append(rows, menu.Row(btns[i:l]...))
	}
	
	nav := []tele.Btn{menu.Data("🏠 Menu Principal", "status_refresh")}
	if end < len(filtered) { nav = append(nav, menu.Data("Suivant ➡️", "lib_page", cat, strconv.Itoa(page+1))) }
	rows = append(rows, menu.Row(nav...))
	menu.Inline(rows...)

	if edit { return c.Edit(msg, menu, tele.ModeHTML) }
	return c.Send(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) handleText(c tele.Context) error {
	if c.Sender().ID != h.cfg.SuperAdmin { return nil }
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
		if res["is_local"] == true { status = "✅ (Disponible)" }
		
		text += fmt.Sprintf("%d %s <b>%s</b> (%s)\n      Status: %s\n\n", i+1, icon, res["title"], res["year"], status)
		
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
			if cat == "films" { return h.sendDetailedMovie(c, it) }
			return h.sendDetailedSeries(c, it)
		}
	}
	return c.Send("❌ <b>Contenu introuvable.</b>\nLa bibliothèque a peut-être été mise à jour entre-temps.", tele.ModeHTML)
}

func (h *BotHandler) handleDelete(c tele.Context) error {
	if err := h.arr.DeleteItem(c.Args()[0], c.Args()[1]); err != nil { 
		return c.Send("❌ <b>Échec de la suppression.</b>\nLe service est temporairement indisponible.", tele.ModeHTML) 
	}
	c.Send("🗑 <b>Fichier retiré de la bibliothèque.</b>", tele.ModeHTML)
	return h.showLibrary(c, c.Args()[0], 0, true)
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

func (h *BotHandler) refreshQueue(c tele.Context, edit bool) error {
	req, _ := http.NewRequest("GET", h.cfg.QbitURL+"/api/v2/torrents/info", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return c.Send("❌ <b>Service qBittorrent inaccessible.</b>", tele.ModeHTML) }
	defer resp.Body.Close()
	
	var torrents []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&torrents)
	
	text := "📥 <b>FLUX DE TÉLÉCHARGEMENT ACTIFS</b>\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n\n"
	count := 0
	for _, t := range torrents {
		st := t["state"].(string)
		if strings.Contains(strings.ToLower(st), "dl") {
			prog := t["progress"].(float64) * 100
			text += fmt.Sprintf("• <b>%s</b>\n  <code>Progression : %.1f%%</code>\n\n", h.cleanTitle(t["name"].(string)), prog)
			count++
		}
	}
	if count == 0 { text += "<i>Aucun flux actif pour le moment.</i>" }
	
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("🔄 Actualiser", "q_refresh"), menu.Data("🏠 Menu Principal", "status_refresh")))
	
	if edit { return c.Edit(text, menu, tele.ModeHTML) }
	return c.Send(text, menu, tele.ModeHTML)
}

func (h *BotHandler) showVpnStatus(c tele.Context, edit bool) error {
	ip := h.vpn.GetCurrentIP()
	msg := fmt.Sprintf("🛡️ <b>PROTECTION VPN & INFRASTRUCTURE</b>\n\n🌍 Adresse IP Publique : <code>%s</code>\n🛰 Statut : Protégé ✅", ip)
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
	// Note: Logic for TMDB search would go here, reusing old logic from main.go
	return nil
}

func (h *BotHandler) cleanTitle(t string) string {
	c := h.reClean.Split(t, -1)[0]
	return strings.TrimSpace(h.reSpaces.ReplaceAllString(strings.ReplaceAll(c, ".", " "), " "))
}

func (h *BotHandler) getIndexEmoji(i int) string {
	e := []string{"", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
	if i > 0 && i < len(e) { return e[i] }
	return strconv.Itoa(i)
}

func (h *BotHandler) formatTitle(t string, y interface{}, max, target int) string {
	f := fmt.Sprintf("%s (%v)", t, y); r := []rune(f)
	if len(r) > max { r = append(r[:max-3], []rune("...")...) }
	res := string(r)
	for len([]rune(res)) < target { res += " " }
	return res
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

func (h *BotHandler) sendDetailedMovie(c tele.Context, it map[string]interface{}) error {
	msg := fmt.Sprintf("🎬 <b>%s</b> (%v)\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n💾 Emplacement : Stockage Rapide (NVMe)\n⚖️ Taille : %.1f GB\n\nQue souhaitez-vous faire ?", 
		it["title"], it["year"], it["sizeOnDisk"].(float64) / (1024*1024*1024))
	
	menu := &tele.ReplyMarkup{}
	btnShare := menu.Data("🔗 Partager", "m_share", "films", fmt.Sprintf("%v", it["id"]))
	btnDelete := menu.Data("🗑 Supprimer", "m_del", "films", fmt.Sprintf("%v", it["id"]))
	menu.Inline(menu.Row(btnShare, btnDelete), menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")))
	
	return c.Send(msg, menu, tele.ModeHTML)
}

func (h *BotHandler) sendDetailedSeries(c tele.Context, it map[string]interface{}) error {
	msg := fmt.Sprintf("📺 <b>%s</b> (%v)\n⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯\n💾 Emplacement : Stockage Rapide (NVMe)\n\nQue souhaitez-vous faire ?", 
		it["title"], it["year"])
	
	menu := &tele.ReplyMarkup{}
	btnShare := menu.Data("🔗 Partager", "m_share", "series", fmt.Sprintf("%v", it["id"]))
	btnDelete := menu.Data("🗑 Supprimer", "m_del", "series", fmt.Sprintf("%v", it["id"]))
	menu.Inline(menu.Row(btnShare, btnDelete), menu.Row(menu.Data("🏠 Menu Principal", "status_refresh")))
	
	return c.Send(msg, menu, tele.ModeHTML)
}

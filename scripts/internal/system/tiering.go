package system

import (
	"fmt"
	"time"
)

// GenerateWeeklyUsageReport analyse les médias et génère un rapport de visionnage réel
func (s *SystemManager) GenerateWeeklyUsageReport() string {
	report := "📊 <b>RAPPORT DE VISIONNAGE & TIERING</b>\n" +
		"--------------------------------------\n\n"

	// 1. Appel API Plex pour les contenus "froids" (6 mois d'inactivité)
	coldMedia, err := s.plex.GetColdMedia(6)
	if err != nil {
		return report + "❌ Erreur API Plex : <code>" + err.Error() + "</code>"
	}

	report += "🧊 <b>Contenus Froids (Oubliés > 6 mois) :</b>\n"
	if len(coldMedia) == 0 {
		report += "✅ Aucun contenu obsolète détecté.\n"
	} else {
		// On limite au top 5 des plus anciens
		limit := 5
		if len(coldMedia) < 5 { limit = len(coldMedia) }
		
		for i := 0; i < limit; i++ {
			m := coldMedia[i]
			t := time.Unix(m.LastViewed, 0)
			if m.LastViewed == 0 { t = time.Unix(m.AddedAt, 0) }
			
			months := int(time.Since(t).Hours() / 24 / 30)
			icon := "🎥"
			if m.Type == "show" { icon = "📺" }
			
			report += fmt.Sprintf("%s <b>%s</b>\n  └ ⏳ Inactif : %d mois\n", icon, m.Title, months)
		}
	}

	report += "\n📈 <b>Activité Système :</b>\n"
	report += "• Statut : Opérationnel\n"
	
	report += "\n💡 <i>Conseil : Ces fichiers consomment du stockage sans être vus. Envisagez le tiering ou la suppression.</i>"

	return report
}

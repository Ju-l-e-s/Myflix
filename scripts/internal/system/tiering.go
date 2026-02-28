package system

import (
	"fmt"
	"time"
)

// ContentReport représente un média candidat au nettoyage ou au tiering
type ContentReport struct {
	Title      string
	SizeGB     float64
	LastViewed time.Time
	AddedDate  time.Time
	Type       string // Movie or Series
}

// GenerateWeeklyUsageReport analyse les médias et génère un rapport de visionnage
func (s *SystemManager) GenerateWeeklyUsageReport() string {
	report := "📊 <b>RAPPORT DE VISIONNAGE & TIERING</b>\n" +
		"--------------------------------------\n\n"

	// 1. Simulation de la collecte
	candidates := []ContentReport{
		{Title: "Inception", SizeGB: 25.4, LastViewed: time.Now().AddDate(0, -14, 0), Type: "🎥"},
		{Title: "The Boys S01", SizeGB: 45.2, LastViewed: time.Now().AddDate(0, -8, 0), Type: "📺"},
	}

	report += "🧊 <b>Contenus Froids (Oubliés) :</b>\n"
	if len(candidates) == 0 {
		report += "✅ Aucun contenu obsolète détecté.\n"
	} else {
		for _, c := range candidates {
			months := int(time.Since(c.LastViewed).Hours() / 24 / 30)
			report += fmt.Sprintf("• %s <b>%s</b>\n  └ 💾 %.1f GB | ⏳ Non vu : %d mois\n",
				c.Type, c.Title, c.SizeGB, months)
		}
	}

	report += "\n📈 <b>Activité de la semaine :</b>\n"
	report += "• Nouveaux médias : 12\n"
	report += "• Heures visionnées : 24h\n"

	report += "\n💡 <i>Conseil : Déplacez ces fichiers vers le Tier HDD pour libérer le NVMe.</i>"

	return report
}

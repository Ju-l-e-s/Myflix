package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token           string
	GeminiKey       string
	TmdbBearerToken string
	DockerMode      bool
	RealIP          string
	QbitURL         string
	PlexURL         string
	PlexToken       string
	SuperAdmin      int64
	PosterCacheDir  string
	RadarrKey       string
	SonarrKey       string
	RadarrURL       string
	SonarrURL       string
	ShareDomain     string
	VpnPortFile     string
	MoviesMount     string
	TvMount         string
	StorageNvmePath string
	StorageHddPath  string
}

func LoadConfig() *Config {
	return &Config{
		Token:           getSecretOrEnv("TELEGRAM_TOKEN", ""),
		GeminiKey:       getSecretOrEnv("GEMINI_KEY", ""),
		TmdbBearerToken: getSecretOrEnv("TMDB_API_KEY", ""),
		DockerMode:      strings.ToLower(os.Getenv("DOCKER_MODE")) == "true",
		RealIP:          os.Getenv("REAL_IP"),
		QbitURL:         getEnv("QBIT_URL", "http://gluetun:8080"),
		PlexURL:         getEnv("PLEX_URL", "http://plex:32400"),
		PlexToken:       getSecretOrEnv("PLEX_TOKEN", ""),
		SuperAdmin:      GetEnvInt64("SUPER_ADMIN", 6721936515),
		PosterCacheDir:  getEnv("POSTER_CACHE_DIR", "/tmp/myflix_cache/posters/"),
		RadarrKey:       getSecretOrEnv("MYFLIX_RADARR_KEY", ""),
		SonarrKey:       getSecretOrEnv("MYFLIX_SONARR_KEY", ""),
		RadarrURL:       getEnv("RADARR_URL", "http://radarr:7878"),
		SonarrURL:       getEnv("SONARR_URL", "http://sonarr:8989"),
		ShareDomain:     getEnv("SHARE_DOMAIN", "https://share.juleslaconfourque.fr"),
		VpnPortFile:     getEnv("VPN_PORT_FILE", "/tmp/gluetun/forwarded_port"),
		MoviesMount:     getEnv("MOVIES_MOUNT", "/movies"),
		TvMount:         getEnv("TV_MOUNT", "/tv"),
		StorageNvmePath: getEnv("STORAGE_NVME_PATH", "/data/internal"),
		StorageHddPath:  getEnv("STORAGE_HDD_PATH", "/data/external"),
	}
}

func getSecretOrEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	secretPath := "/run/secrets/" + strings.ToLower(key)
	if data, err := os.ReadFile(secretPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val != "" {
			return val
		}
	}
	return fallback
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

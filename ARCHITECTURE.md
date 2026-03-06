# 🏗 ARCHITECTURE - MyFlix

## 🧱 Architectural Model
MyFlix follows a **Modular Monolith** architecture inspired by *Clean Architecture* principles:
- **CMD Layer:** Bootstrap and dependency injection.
- **Internal Layer:** Strict encapsulation of business logic by domain.
- **Infrastructure:** Abstraction of external APIs (Radarr, Sonarr, Plex, Gemini).

> [!IMPORTANT]
> Any modifications to the infrastructure (docker-compose.yml or deployment scripts) must be analyzed through the lens of the **AWS Well-Architected Framework**, prioritizing the **Reliability** and **Cost Optimization** pillars.

## 🔄 Request Flow Diagram

```text
User (Telegram) 
   │
   ▼
[Bot Handler] (internal/bot)
   │
   ├─► [AI Resolve] (internal/ai): Intent analysis (Title, Year, Type)
   │
   ├─► [Arr Client] (internal/arrclient): Search/Add in Radarr/Sonarr
   │
   ├─► [System] (internal/system): Disk space check (Tiering)
   │
   └─► [Feedback]: Confirmation & Status (Inline Buttons)
```

## ⚡ Concurrency & Performance
1. **Background Tasks:** Heavy use of Goroutines for cache "warmup" and nightly maintenance tasks.
2. **Thread-Safety:** Protection of shared states (API caches, short path mapping) using `sync.RWMutex`.
3. **Flow Monitoring:** Direct interaction with qBittorrent to display real-time progress via polling.
4. **Storage Tiering:** Detection algorithm for "cold" content (6 months of Plex inactivity) to automate moves or suggest deletion.

# CashbackTV Transcoder V2 - Tam Teknik Spesifikasyon

## 📋 Proje Özeti

**Amaç:** Video stream'lerine logo ekleyerek HLS formatında yeniden yayınlayan, enterprise-grade bir transcoding platformu.

**Hedef Sunucu:**
- CPU: Dual Intel Xeon Gold 6152 (44 çekirdek / 88 thread)
- RAM: 256GB DDR4
- Depolama: 1TB NVMe
- Ağ: 2×10 Gbps (20 Gbps toplam, unlimited bandwidth)

**Kapasite Hedefi:** 80-120 eşzamanlı kanal, broadcast kalitesinde

---

## 🏗️ Sistem Mimarisi

### Katmanlı Mimari

```
                         ┌─────────────────┐
                         │     Client      │
                         │  (Browser/App)  │
                         └────────┬────────┘
                                  │
                         ┌────────▼────────┐
                         │  Nginx Reverse  │
                         │     Proxy       │
                         │  (SSL + Static) │
                         │   Port 443/80   │
                         └────────┬────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
              ▼                   ▼                   ▼
     ┌───────────────┐   ┌───────────────┐   ┌───────────────┐
     │   Frontend    │   │   Backend     │   │  HLS Streams  │
     │   (Next.js)   │   │   (Go API)    │   │   /streams/*  │
     │   Port 3000   │   │   Port 8080   │   │   (Static)    │
     └───────────────┘   └───────┬───────┘   └───────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
     ┌───────────────┐   ┌───────────────┐   ┌───────────────┐
     │   PostgreSQL  │   │    Redis      │   │  Transcoder   │
     │   (Metadata)  │   │ (Cache/Queue) │   │   Workers     │
     │   Port 5432   │   │   Port 6379   │   │   (FFmpeg)    │
     └───────────────┘   └───────────────┘   └───────────────┘
                                                    │
                                                    ▼
                                           ┌───────────────┐
                                           │   HLS Output  │
                                           │   (RAM Disk)  │
                                           │    100GB      │
                                           └───────────────┘
```

### Bileşen Detayları

#### 1. Nginx Reverse Proxy
- SSL termination (Let's Encrypt + Certbot auto-renewal)
- Static file serving (HLS streams, uploads)
- HTTP/2 desteği
- Gzip/Brotli sıkıştırma
- WebSocket proxy (real-time updates için)
- Rate limiting

#### 2. Frontend (Next.js 14 + shadcn/ui)
- App Router mimarisi
- Server-Side Rendering (SSR)
- Server Actions (form işlemleri)
- Real-time dashboard (WebSocket)
- shadcn/ui + Tailwind CSS
- Dark/Light mode
- Responsive design

#### 3. Backend API (Go + Fiber/Echo)
- Clean Architecture (Domain-Driven Design)
- RESTful API + WebSocket
- JWT authentication + RBAC
- Swagger/OpenAPI documentation
- Structured logging (zerolog)
- Prometheus metrics endpoint
- Health check endpoints

#### 4. Database Layer
- **PostgreSQL:** Kanal metadata, kullanıcılar, loglar
- **Redis:** Session cache, job queue, real-time metrics
- **File System:** Logo uploads, HLS segments

#### 5. Transcoder Engine
- Worker pool pattern (CPU core başına worker)
- FFmpeg process management
- Real-time progress tracking
- Auto-restart on failure
- Resource isolation (cgroups)

---

## 📁 Proje Yapısı

```
cashbacktv-v2/
├── docker/
│   ├── docker-compose.local.yml      # Local development
│   ├── docker-compose.prod.yml       # Production (SSL dahil)
│   ├── nginx/
│   │   ├── nginx.local.conf
│   │   ├── nginx.prod.conf
│   │   └── ssl/                      # SSL certificates
│   └── certbot/
│       └── renew-hook.sh
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # Entry point
│   │
│   ├── internal/
│   │   ├── domain/                   # Business entities
│   │   │   ├── channel.go
│   │   │   ├── user.go
│   │   │   └── transcoder.go
│   │   │
│   │   ├── application/              # Use cases / Services
│   │   │   ├── channel_service.go
│   │   │   ├── transcoder_service.go
│   │   │   └── auth_service.go
│   │   │
│   │   ├── infrastructure/           # External implementations
│   │   │   ├── repository/
│   │   │   │   ├── postgres/
│   │   │   │   └── redis/
│   │   │   ├── ffmpeg/
│   │   │   │   ├── process.go
│   │   │   │   ├── monitor.go
│   │   │   │   └── builder.go
│   │   │   └── storage/
│   │   │       └── filesystem.go
│   │   │
│   │   ├── interfaces/               # API handlers
│   │   │   ├── http/
│   │   │   │   ├── handlers/
│   │   │   │   ├── middleware/
│   │   │   │   └── routes.go
│   │   │   └── websocket/
│   │   │       └── hub.go
│   │   │
│   │   └── pkg/                      # Shared utilities
│   │       ├── config/
│   │       ├── logger/
│   │       └── validator/
│   │
│   ├── migrations/                   # Database migrations
│   ├── go.mod
│   └── Dockerfile
│
├── frontend/
│   ├── app/
│   │   ├── (auth)/
│   │   │   ├── login/
│   │   │   └── layout.tsx
│   │   ├── (dashboard)/
│   │   │   ├── channels/
│   │   │   ├── monitoring/
│   │   │   ├── settings/
│   │   │   └── layout.tsx
│   │   ├── api/                      # API routes
│   │   ├── layout.tsx
│   │   └── page.tsx
│   │
│   ├── components/
│   │   ├── ui/                       # shadcn/ui components
│   │   ├── channels/
│   │   ├── monitoring/
│   │   └── common/
│   │
│   ├── lib/
│   │   ├── api.ts                    # API client
│   │   ├── websocket.ts
│   │   └── utils.ts
│   │
│   ├── hooks/
│   ├── types/
│   ├── package.json
│   ├── tailwind.config.js
│   └── Dockerfile
│
├── scripts/
│   ├── setup-ssl.sh                  # Let's Encrypt setup
│   ├── backup.sh
│   └── restore.sh
│
├── docs/
│   ├── API.md
│   ├── DEPLOYMENT.md
│   └── ARCHITECTURE.md
│
└── README.md
```

---

## 🎯 Backend Mimarisi (Clean Architecture)

### Domain Layer (İş Kuralları)

#### Channel Entity
- ID (UUID)
- Name
- SourceURL
- Logo (path, x, y, width, height, opacity)
- OutputConfig (codec, bitrate, resolution)
- Status (stopped, starting, running, error, stopping)
- CreatedAt, UpdatedAt

#### TranscoderProcess Entity
- ChannelID
- PID (FFmpeg process ID)
- StartedAt
- CPU Usage (%)
- Memory Usage (MB)
- Input Bitrate (kbps)
- Output Bitrate (kbps)
- Dropped Frames
- FPS
- Speed (1.0x = real-time)
- Last Error
- Uptime

#### User Entity
- ID, Email, PasswordHash
- Role (admin, operator, viewer)
- Permissions

### Application Layer (Use Cases)

#### ChannelService
- CreateChannel
- UpdateChannel
- DeleteChannel
- ListChannels
- GetChannel

#### TranscoderService
- StartChannel
- StopChannel
- RestartChannel
- GetProcessMetrics
- GetAllProcessMetrics
- SetAutoRestart

#### AuthService
- Login
- Logout
- RefreshToken
- ValidateToken

### Infrastructure Layer

#### FFmpeg Process Manager
- Process başlatma/durdurma
- stderr parsing (progress extraction)
- Resource monitoring (CPU/RAM)
- Auto-restart logic
- Graceful shutdown

#### PostgreSQL Repository
- GORM veya sqlx
- Connection pooling
- Migration management

#### Redis Cache
- Session storage
- Real-time metrics buffer
- Job queue (kanal başlatma sırası)

---

## 📡 API Endpoints

### Authentication
```
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me
```

### Channels
```
GET    /api/v1/channels              # Liste (pagination, filter)
POST   /api/v1/channels              # Oluştur
GET    /api/v1/channels/:id          # Detay
PUT    /api/v1/channels/:id          # Güncelle
DELETE /api/v1/channels/:id          # Sil
POST   /api/v1/channels/:id/start    # Başlat
POST   /api/v1/channels/:id/stop     # Durdur
POST   /api/v1/channels/:id/restart  # Yeniden başlat
GET    /api/v1/channels/:id/metrics  # Process metrikleri
GET    /api/v1/channels/:id/logs     # FFmpeg logları
```

### Monitoring
```
GET    /api/v1/monitoring/system     # CPU, RAM, Disk, Network
GET    /api/v1/monitoring/processes  # Tüm FFmpeg process'leri
GET    /api/v1/monitoring/streams    # Aktif stream'ler
WS     /api/v1/ws/metrics            # Real-time metrics stream
```

### Settings
```
GET    /api/v1/settings              # Sistem ayarları
PUT    /api/v1/settings              # Güncelle
GET    /api/v1/settings/presets      # Encoding presetleri
```

### Uploads
```
POST   /api/v1/uploads/logo          # Logo yükle
DELETE /api/v1/uploads/:id           # Dosya sil
```

---

## 🖥️ Frontend Sayfaları

### 1. Login Sayfası
- Email/Password form
- Remember me
- Forgot password

### 2. Dashboard (Ana Sayfa)
- Özet kartlar: Aktif kanal, CPU, RAM, Bandwidth
- Son aktiviteler
- Hızlı eylemler
- System health indicator

### 3. Channels (Kanal Yönetimi)
- Kanal listesi (grid/list view)
- Her kanal kartında:
  - Thumbnail preview
  - Status indicator
  - Quick actions (start/stop/restart)
  - CPU/RAM kullanımı
- Kanal ekleme/düzenleme modal
- Toplu işlemler (çoklu seçim)

### 4. Channel Detail
- Canlı video preview
- FFmpeg log stream
- Real-time metrics grafikleri
- Logo pozisyon editörü (drag & drop)
- Encoding ayarları

### 5. Monitoring
- System metrics (CPU cores, RAM, Disk I/O, Network)
- Process listesi (tüm FFmpeg'ler)
- Bandwidth grafikleri
- Error log viewer
- Alert history

### 6. Settings
- Genel ayarlar
- Encoding presetleri
- Kullanıcı yönetimi
- API keys
- Backup/Restore

---

## ⚡ FFmpeg Process İzleme

### İzlenecek Metrikler

#### Input Metrikleri
- Input bitrate (kbps)
- Input FPS
- Input resolution
- Packet loss

#### Encoding Metrikleri
- Output bitrate (kbps)
- Output FPS
- Encoding speed (1.0x = real-time)
- Quality score (VMAF/SSIM estimate)

#### Resource Metrikleri
- CPU usage (per-process)
- Memory usage (RSS/VSZ)
- Thread count
- I/O wait

#### Stream Metrikleri
- Segment generation rate
- Playlist update frequency
- Client connections
- Bandwidth per channel

### Monitoring Implementasyonu

#### FFmpeg stderr Parsing
FFmpeg progress bilgisi stderr'den parse edilir:
- frame= (işlenen frame sayısı)
- fps= (encoding hızı)
- bitrate= (output bitrate)
- speed= (gerçek zamana oranı)
- drop= (düşürülen frame)
- dup= (duplicate frame)

#### Process Stats (/proc filesystem)
- /proc/[pid]/stat - CPU time
- /proc/[pid]/statm - Memory
- /proc/[pid]/io - I/O stats
- /proc/[pid]/fd - File descriptors

#### WebSocket Real-time Updates
Her 1 saniyede client'lara metrics push:
- Channel-specific metrics
- System-wide metrics
- Alert notifications

---

## 🐳 Docker Compose Yapılandırmaları

### Local Development (docker-compose.local.yml)

#### Servisler
- **frontend:** Next.js dev server (hot reload)
- **backend:** Go with air (hot reload)
- **postgres:** PostgreSQL 16
- **redis:** Redis 7

#### Özellikler
- Volume mounts for source code
- Debug ports exposed
- No resource limits
- Fake SSL (self-signed)

### Production (docker-compose.prod.yml)

#### Servisler
- **nginx:** Reverse proxy + SSL termination + Static files
- **frontend:** Next.js production build
- **backend:** Go production binary
- **postgres:** PostgreSQL 16 (with replication ready)
- **redis:** Redis 7 (persistence enabled)
- **certbot:** SSL certificate auto-renewal

#### Kaynak Tahsisi (Dual Xeon Gold 6152 için)

| Servis | CPU Cores | RAM | Açıklama |
|--------|-----------|-----|----------|
| Nginx | 2 core | 2GB | Reverse proxy, SSL, Static |
| Frontend | 2 core | 4GB | Next.js SSR |
| Backend | 8 core | 16GB | API + Transcoder Manager |
| PostgreSQL | 4 core | 8GB | Database |
| Redis | 2 core | 4GB | Cache + Queue |
| FFmpeg Pool | 66 core | 206GB | Transcoding workers |
| System | 4 core | 16GB | OS overhead |
| **Toplam** | **88 core** | **256GB** | |

#### RAM Disk Yapılandırması
- /mnt/ramdisk: 100GB tmpfs
- HLS segments bu alana yazılır
- Ultra-fast I/O (NVMe bile yetersiz kalabilir 120 kanal için)

#### CPU Affinity
- Nginx: CPU 0-1
- Frontend: CPU 2-3
- Backend: CPU 4-11
- PostgreSQL: CPU 12-15
- Redis: CPU 16-17
- FFmpeg Workers: CPU 18-85 (68 core)
- Reserved: CPU 86-87

---

## 🔐 SSL/TLS Yapılandırması

### Let's Encrypt + Certbot

#### Başlangıç Kurulumu
1. Domain DNS kaydı (A record → sunucu IP)
2. Certbot container başlat
3. HTTP-01 challenge ile sertifika al
4. Nginx reload

#### Auto-Renewal
- Certbot cron job (12 saatte bir kontrol)
- 30 gün kala yenileme
- Nginx graceful reload hook

#### SSL Parametreleri
- TLS 1.2 ve 1.3 only
- Modern cipher suite
- HSTS enabled
- OCSP stapling
- Certificate transparency

---

## 🚀 Performans Optimizasyonları

### FFmpeg Optimizasyonları

#### Encoding Parametreleri (80-120 kanal için)
- Preset: veryfast (CPU dengesi)
- Tune: zerolatency (düşük gecikme)
- CRF: 23 (kalite/boyut dengesi)
- Profile: high
- Level: 4.1
- Keyframe: 2 saniye
- B-frames: 0 (latency için)

#### HLS Parametreleri
- Segment süresi: 2 saniye
- Playlist boyutu: 6 segment (12 saniye)
- Delete segments: enabled
- Independent segments: enabled

### Sistem Optimizasyonları

#### Kernel Parametreleri (sysctl)
- net.core.somaxconn: 65535
- net.ipv4.tcp_max_syn_backlog: 65535
- net.core.netdev_max_backlog: 65535
- net.ipv4.ip_local_port_range: 1024 65535
- fs.file-max: 2097152
- fs.inotify.max_user_watches: 524288
- vm.swappiness: 10
- vm.dirty_ratio: 60
- vm.dirty_background_ratio: 2

#### File Descriptor Limits
- Soft: 1048576
- Hard: 1048576

#### Network Tuning
- TCP BBR congestion control
- Jumbo frames (MTU 9000) eğer network destekliyorsa
- Receive/Send buffer optimization

### Database Optimizasyonları

#### PostgreSQL
- shared_buffers: 2GB
- effective_cache_size: 6GB
- work_mem: 64MB
- maintenance_work_mem: 512MB
- max_connections: 200
- Connection pooling (PgBouncer)

#### Redis
- maxmemory: 4GB
- maxmemory-policy: allkeys-lru
- Persistence: RDB + AOF

---

## 📊 Kapasite Planlaması

### Kanal Başına Kaynak

| Metrik | Değer | Açıklama |
|--------|-------|----------|
| CPU | 0.5-0.8 core | veryfast preset |
| RAM | 1.5-2GB | FFmpeg + buffers |
| Disk I/O | 5-8 MB/s | HLS segment yazma |
| Network Out | 3-5 Mbps | Output stream |

### Toplam Kapasite (Dual Gold 6152)

| Kanal Sayısı | CPU (%) | RAM | Bandwidth | Durum |
|--------------|---------|-----|-----------|-------|
| 50 | 30-40% | 100GB | 250 Mbps | Rahat ✅ |
| 80 | 50-60% | 160GB | 400 Mbps | Optimal ✅ |
| 100 | 60-75% | 200GB | 500 Mbps | Yüksek ⚡ |
| 120 | 75-90% | 240GB | 600 Mbps | Maksimum 🔥 |

### Network Kapasitesi
- Toplam: 20 Gbps
- Kullanılabilir: ~18 Gbps (overhead)
- 120 kanal × 5 Mbps = 600 Mbps output
- Headroom: %97 (çok rahat)

---

## 🔄 Hata Yönetimi ve Recovery

### FFmpeg Process Monitoring
- Health check: 5 saniyede bir
- Timeout: 30 saniye segment üretimi yoksa
- Max restart: 5 (10 dakika içinde)
- Cooldown: 30 saniye restart arası

### Error Kategorileri

| Kategori | Aksiyon | Örnek |
|----------|---------|-------|
| Network | Auto-retry | Source timeout |
| Encoding | Restart | Codec error |
| Resource | Alert + Wait | Out of memory |
| Fatal | Stop + Alert | Invalid source |

### Alert Mekanizması
- Email notifications
- Webhook (Slack, Discord, Teams)
- Dashboard notifications
- SMS (opsiyonel)

---

## 📈 Monitoring ve Logging

### Metrics Stack
- **Prometheus:** Metrics collection
- **Grafana:** Visualization
- **Alertmanager:** Alert routing

### Log Management
- **Structured logging:** JSON format
- **Log levels:** DEBUG, INFO, WARN, ERROR
- **Rotation:** Günlük, 7 gün retention
- **Centralized:** Loki veya ELK stack (opsiyonel)

### Key Metrics

#### System
- cpu_usage_percent
- memory_usage_bytes
- disk_usage_bytes
- network_rx_bytes
- network_tx_bytes

#### Application
- active_channels_total
- ffmpeg_processes_total
- api_request_duration_seconds
- websocket_connections_total

#### Per-Channel
- channel_cpu_usage_percent
- channel_memory_usage_bytes
- channel_output_bitrate_kbps
- channel_encoding_speed
- channel_dropped_frames_total
- channel_uptime_seconds

---

## 🛠️ Development Workflow

### Local Setup
1. Docker Compose local başlat
2. Frontend: `npm run dev`
3. Backend: `air` (hot reload)
4. Access: http://localhost:3000

### Git Branching
- main: production-ready
- develop: integration
- feature/*: new features
- hotfix/*: urgent fixes

### CI/CD Pipeline
1. Push → GitHub Actions
2. Lint + Test
3. Build Docker images
4. Push to registry
5. Deploy to staging
6. Manual approval → Production

---

## 📝 Checklist

### MVP Features
- [ ] User authentication (JWT)
- [ ] Channel CRUD
- [ ] FFmpeg transcoding with logo
- [ ] HLS output serving
- [ ] Real-time process monitoring
- [ ] Basic dashboard
- [ ] Docker deployment

### Phase 2
- [ ] Multi-user support (RBAC)
- [ ] Encoding presets
- [ ] Scheduled start/stop
- [ ] Bandwidth limiting
- [ ] API rate limiting
- [ ] Webhook notifications

### Phase 3
- [ ] Multi-bitrate HLS (ABR)
- [ ] DVR/Recording
- [ ] Analytics dashboard
- [ ] Prometheus/Grafana integration
- [ ] Cluster mode (horizontal scaling)

---

## 🎯 Sonuç

Bu mimari, Dual Xeon Gold 6152 sunucunuzda **80-120 kanal** kapasitesiyle çalışacak şekilde optimize edilmiştir. Clean Architecture sayesinde kod bakımı kolay, test edilebilir ve genişletilebilir olacaktır. Real-time FFmpeg monitoring ile her process'in detaylı metriklerini izleyebilir, sorunları anında tespit edebilirsiniz.

**Teknoloji Stack Özeti:**
- Frontend: Next.js 14 + shadcn/ui + Tailwind
- Backend: Go + Fiber/Echo + Clean Architecture
- Database: PostgreSQL + Redis
- Transcoding: FFmpeg
- Container: Docker + Docker Compose
- SSL: Let's Encrypt + Certbot
- Monitoring: Prometheus + Grafana (opsiyonel)


# CashbackTV Transcoder V2

Enterprise-grade video transcoding platform with real-time monitoring. Designed for high-performance servers running 80-120 simultaneous channels.

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 20+ (for local development)
- Go 1.22+ (for local development)

### Local Development

1. **Start with Docker Compose:**
```bash
cd docker
docker-compose -f docker-compose.local.yml up -d
```

2. **Access the application:**
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- PostgreSQL: localhost:5432
- Redis: localhost:6379

3. **Default credentials (local development only):**
- Email: `admin@cashbacktv.local`
- Password: `admin123`

### Development Without Docker

**Backend:**
```bash
cd backend
go mod download
go run cmd/server/main.go
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev
```

## 📁 Project Structure

```
cashbacktv.live/
├── backend/           # Go API (Clean Architecture)
│   ├── cmd/server/    # Entry point
│   ├── internal/
│   │   ├── domain/        # Business entities
│   │   ├── application/   # Use cases
│   │   ├── infrastructure/# External implementations
│   │   └── interfaces/    # HTTP handlers
│   └── migrations/    # Database migrations
│
├── frontend/          # Next.js 14 + shadcn/ui
│   ├── app/           # App Router pages
│   ├── components/    # React components
│   └── lib/           # Utilities & API client
│
└── docker/            # Docker configurations
    ├── nginx/         # Nginx configs
    └── docker-compose.*.yml
```

## 🛠️ Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Next.js 14, React 18, shadcn/ui, Tailwind CSS |
| Backend | Go, Fiber, Clean Architecture |
| Database | PostgreSQL 16, Redis 7 |
| Transcoding | FFmpeg |
| Container | Docker, Docker Compose |
| Proxy | Nginx (SSL, HLS delivery) |

## 📡 API Endpoints

### Authentication
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/logout` - Logout
- `POST /api/v1/auth/refresh` - Refresh token
- `GET /api/v1/auth/me` - Current user

### Channels
- `GET /api/v1/channels` - List all channels
- `POST /api/v1/channels` - Create channel
- `GET /api/v1/channels/:id` - Get channel
- `PUT /api/v1/channels/:id` - Update channel
- `DELETE /api/v1/channels/:id` - Delete channel
- `POST /api/v1/channels/:id/start` - Start transcoding
- `POST /api/v1/channels/:id/stop` - Stop transcoding
- `POST /api/v1/channels/:id/restart` - Restart transcoding
- `GET /api/v1/channels/:id/metrics` - Get metrics

## 🔧 Configuration

Environment variables for backend:

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | API server port |
| `DATABASE_HOST` | localhost | PostgreSQL host |
| `DATABASE_PORT` | 5432 | PostgreSQL port |
| `DATABASE_USER` | cashbacktv | Database user |
| `DATABASE_PASSWORD` | cashbacktv | Database password |
| `REDIS_HOST` | localhost | Redis host |
| `JWT_SECRET` | - | JWT signing secret |
| `STORAGE_HLS_PATH` | /var/lib/cashbacktv/streams | HLS output path |

## 📊 Capacity Planning

For Dual Intel Xeon Gold 6152 (44 cores / 88 threads, 256GB RAM):

| Channels | CPU Usage | Memory | Bandwidth |
|----------|-----------|--------|-----------|
| 50 | 30-40% | 100GB | 250 Mbps |
| 80 | 50-60% | 160GB | 400 Mbps |
| 100 | 60-75% | 200GB | 500 Mbps |
| 120 | 75-90% | 240GB | 600 Mbps |

## 🔐 Production Deployment

1. **Update environment variables in `.env`**

2. **Setup SSL certificates for both domains:**
```bash
# For main domain
certbot certonly --standalone -d cashbacktv.live -d www.cashbacktv.live

# For CDN domain
certbot certonly --standalone -d cdn.cashbacktv.live
```

3. **Start production stack:**
```bash
cd docker
docker-compose -f docker-compose.prod.yml up -d
```

4. **Domain Configuration:**
- Main domain: `cashbacktv.live` - Frontend and API
- CDN domain: `cdn.cashbacktv.live` - HLS streaming content only

**Note:** Make sure DNS records are configured:
- `cashbacktv.live` → Server IP
- `www.cashbacktv.live` → Server IP
- `cdn.cashbacktv.live` → Server IP (or CDN provider)

## 📝 License

MIT License





# 🎮 4 in a Row - Real-Time Multiplayer Game

A production-ready implementation of Connect Four with real-time multiplayer, strategic AI bot, and full-stack deployment.

## 🌟 Features

### Core Gameplay
- **Real-time multiplayer** using WebSockets
- **1v1 gameplay** between two players
- **Competitive AI bot** that automatically joins if no opponent found within 10 seconds
- **Strategic bot AI** that blocks opponent moves and creates winning opportunities (not random)
- **Player reconnection** support (30-second grace period)
- **Automatic forfeiture** if player doesn't reconnect within 30 seconds

### Technical Features
- **Backend**: GoLang with goroutines for concurrent game management
- **Frontend**: React with real-time WebSocket communication
- **Database**: PostgreSQL for persistent storage
- **Analytics**: Kafka-based event streaming with dedicated consumer service
- **Leaderboard**: Track wins, losses, and player statistics
- **Containerized**: Full Docker Compose setup

## 🏗️ Architecture

```
┌─────────────┐
│   Frontend  │ (React)
│  Port 3000  │
└──────┬──────┘
       │ WebSocket
       ↓
┌─────────────┐     ┌──────────────┐
│   Backend   │────→│  PostgreSQL  │
│  Port 8080  │     │  Port 5432   │
└──────┬──────┘     └──────────────┘
       │ Events
       ↓
┌─────────────┐     ┌──────────────┐
│    Kafka    │────→│  Analytics   │
│  Port 9092  │     │   Consumer   │
└─────────────┘     └──────────────┘
```

### Components

1. **Backend Service** (`backend/`)
   - WebSocket server for real-time communication
   - Game logic and state management
   - Matchmaking system (10-second timeout)
   - Competitive bot AI
   - Player reconnection handling
   - Kafka event producer
   - REST API for leaderboard

2. **Analytics Service** (`analytics/`)
   - Kafka consumer
   - Event processing and storage
   - Metrics calculation (game duration, win rates, etc.)
   - Time-based analytics (games per hour/day)

3. **Frontend** (`frontend/`)
   - React-based UI
   - WebSocket client
   - Real-time game board updates
   - Leaderboard display

4. **Infrastructure**
   - PostgreSQL for data persistence
   - Kafka + Zookeeper for event streaming

## 📋 Prerequisites

- **Docker** and **Docker Compose** installed
- **Go 1.21+** (for local development)
- **Node.js 18+** (for local frontend development)

## 🚀 Quick Start with Docker

### 1. Clone the repository
```bash
git clone <your-repo-url>
cd 4-in-a-row
```

### 2. Start all services
```bash
docker-compose up --build
```

This will start:
- PostgreSQL on `localhost:5432`
- Kafka on `localhost:9092`
- Backend on `localhost:8080`
- Frontend on `localhost:3000`
- Analytics consumer (background service)

### 3. Access the game
Open your browser and navigate to:
```
http://localhost:3000
```

### 4. Stop all services
```bash
docker-compose down
```

To remove all data:
```bash
docker-compose down -v
```

## 🛠️ Local Development

### Backend Development

```bash
cd backend

# Install dependencies
go mod download

# Run the server
go run .
```

Environment variables (optional):
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=fourinarow
export KAFKA_BROKER=localhost:9092
export KAFKA_TOPIC=game-events
export PORT=8080
```

### Analytics Service Development

```bash
cd analytics

# Install dependencies
go mod download

# Run the consumer
go run .
```

### Frontend Development

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm start
```

Frontend runs on `http://localhost:3000` by default.

## 🎮 How to Play

1. **Enter your username** on the home screen
2. Click **"Join Game"**
3. Wait for an opponent (or bot will join after 10 seconds)
4. **Click on a column** to drop your disc
5. **Connect 4 discs** vertically, horizontally, or diagonally to win!

### Game Rules
- Board: 7 columns × 6 rows
- Players take turns dropping discs into columns
- Discs fall to the lowest available position
- First player to connect 4 discs wins
- If board fills up with no winner, it's a draw

## 🤖 Bot Behavior

The competitive bot uses strategic AI:

1. **Win Immediately**: If bot can win on this turn, it will
2. **Block Opponent**: If opponent can win next turn, bot blocks
3. **Strategic Positioning**: Prefers center columns
4. **Threat Creation**: Builds potential winning sequences

## 🔌 API Endpoints

### WebSocket
```
ws://localhost:8080/ws
```

**Message Types:**
- `join`: Join matchmaking queue
- `move`: Make a move
- `reconnect`: Reconnect to existing game

### REST API
```
GET /api/leaderboard - Get top 10 players
GET /api/health      - Health check
```

## 📊 Analytics & Metrics

The analytics service tracks:
- Game start/end events
- Player moves
- Game duration
- Win/loss statistics
- Games per hour/day
- Average game duration

All events are stored in PostgreSQL for historical analysis.

## 🗄️ Database Schema

### `games` table
- Game history
- Players
- Winner
- Duration
- Move count

### `players` table
- Username
- Games played/won/lost/drawn
- Total moves
- Created date

### `analytics_events` table
- Raw event storage (JSONB)
- Event type
- Timestamp

### `analytics_metrics` table
- Calculated metrics
- Time-series data

## 🐳 Docker Services

| Service | Port | Description |
|---------|------|-------------|
| frontend | 3000 | React app |
| backend | 8080 | Go WebSocket server |
| postgres | 5432 | PostgreSQL database |
| kafka | 9092 | Kafka broker |
| zookeeper | 2181 | Kafka coordination |
| analytics | - | Background consumer |

## 🔧 Configuration

### Backend Configuration
Edit environment variables in `docker-compose.yml` or set them locally.

### Frontend Configuration
Update `.env.production` for production builds:
```
REACT_APP_WS_URL=ws://your-domain.com/ws
REACT_APP_API_URL=http://your-domain.com/api
```

## 📦 Deployment

### Option 1: Docker Compose (Recommended)
```bash
docker-compose up -d
```

### Option 2: Cloud Deployment
1. Build Docker images
2. Push to container registry
3. Deploy to Kubernetes/ECS/Cloud Run
4. Configure environment variables
5. Set up managed PostgreSQL and Kafka

### Environment Variables for Production
```bash
# Backend
DB_HOST=<your-db-host>
DB_PASSWORD=<secure-password>
KAFKA_BROKER=<kafka-broker>

# Frontend
REACT_APP_WS_URL=wss://<your-domain>/ws
REACT_APP_API_URL=https://<your-domain>/api
```

## 🧪 Testing

### Backend Tests
```bash
cd backend
go test ./...
```

### Frontend Tests
```bash
cd frontend
npm test
```

## 🐛 Troubleshooting

### WebSocket Connection Issues
- Ensure backend is running on port 8080
- Check firewall settings
- For production, use WSS (secure WebSocket)

### Database Connection Errors
- Verify PostgreSQL is running
- Check connection string
- Ensure database `fourinarow` exists

### Kafka Issues
- Wait for Kafka to fully start (can take 30-60 seconds)
- Check Zookeeper is healthy
- Verify topic creation

## 📝 License

This project is provided as-is for educational purposes.

## 👥 Contributing

This is an assignment submission, but suggestions are welcome!

## 🎯 Assignment Requirements Checklist

- ✅ Real-time multiplayer with WebSockets
- ✅ Player matchmaking with 10-second bot fallback
- ✅ Competitive bot (strategic, not random)
- ✅ Reconnection support (30-second window)
- ✅ Game state persistence (PostgreSQL)
- ✅ Leaderboard with player stats
- ✅ Kafka analytics integration
- ✅ Analytics consumer service
- ✅ React frontend
- ✅ Docker Compose setup
- ✅ **GoLang backend** (preferred over Node.js)
- ✅ Comprehensive README

## 📧 Contact

For questions about this implementation, please create an issue on GitHub.

---

**Built with ❤️ using GoLang, React, PostgreSQL, and Kafka**

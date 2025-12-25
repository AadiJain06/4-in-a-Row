# 4 in a Row - Real-Time Multiplayer Game

A real-time multiplayer implementation of the classic **Connect Four** (4 in a Row) game built with Go backend and vanilla JavaScript frontend. Features include player matchmaking, competitive AI bot, WebSocket-based real-time gameplay, game analytics via Kafka, and persistent leaderboard.

## 🎮 Table of Contents

- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Architecture](#-architecture)
- [Project Structure](#-project-structure)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [How to Play](#-how-to-play)
- [Configuration](#-configuration)
- [API Documentation](#-api-documentation)
- [Running Locally](#-running-locally)
- [Deployment](#-deployment)
- [Development](#-development)
- [Testing](#-testing)
- [Troubleshooting](#-troubleshooting)

## ✨ Features

### Core Gameplay
- **Real-time multiplayer** - Play against other players or a competitive bot
- **WebSocket communication** - Instant updates for both players
- **7×6 game board** - Standard Connect Four grid
- **Win detection** - Automatic detection of horizontal, vertical, and diagonal wins
- **Draw detection** - Game ends in draw when board is full

### Matchmaking & Bot
- **Smart matchmaking** - Automatic pairing of waiting players
- **Competitive bot** - Strategic AI opponent that:
  - Takes winning moves when available
  - Blocks opponent's winning moves
  - Prefers center columns for strategic advantage
- **10-second bot fallback** - Bot joins automatically if no opponent found

### Reconnection & Reliability
- **30-second reconnection window** - Players can rejoin games after disconnection
- **Game state persistence** - Completed games saved to PostgreSQL
- **Automatic forfeit** - Games forfeited if player doesn't reconnect in time

### Analytics & Leaderboard
- **Kafka integration** - Real-time game event streaming
- **Analytics consumer** - Tracks game duration, wins, games per day/hour, user metrics
- **Leaderboard** - Tracks and displays player wins
- **In-memory fallback** - Works without database/Kafka for development

## 🛠 Tech Stack

### Backend
- **Go 1.21+** - Main backend language
- **Gin** - HTTP web framework
- **Gorilla WebSocket** - WebSocket implementation
- **PostgreSQL** - Database for game persistence (optional)
- **Kafka** - Message queue for analytics (optional)
- **pgx/v5** - PostgreSQL driver

### Frontend
- **Vanilla JavaScript** - No frameworks, pure JS
- **HTML5/CSS3** - Modern web standards
- **WebSocket API** - Real-time communication

### Analytics
- **Kafka Go Client** - Event streaming
- **Go** - Analytics consumer service

## 🏗 Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Browser   │◄───────►│   Backend    │◄───────►│  PostgreSQL │
│  (Frontend) │ WebSocket│   (Go/Gin)   │         │  (Optional) │
└─────────────┘         └──────────────┘         └─────────────┘
                               │
                               │ Kafka Events
                               ▼
                        ┌──────────────┐
                        │   Analytics  │
                        │   Consumer   │
                        └──────────────┘
```

### Key Components

1. **Game Manager** - Handles game state, matchmaking, and bot integration
2. **WebSocket Server** - Manages real-time connections and message routing
3. **Board Logic** - Win detection and move validation
4. **Bot AI** - Strategic decision-making for AI opponent
5. **Storage Layer** - PostgreSQL integration for persistence
6. **Analytics Producer** - Kafka event publishing
7. **Analytics Consumer** - Event processing and metrics tracking

## 📁 Project Structure

```
Emittr/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go          # Application entry point
│   ├── internal/
│   │   ├── analytics/
│   │   │   └── producer.go      # Kafka producer
│   │   ├── game/
│   │   │   ├── board.go         # Board logic & win detection
│   │   │   ├── bot.go           # Bot AI strategy
│   │   │   └── manager.go       # Game state management
│   │   ├── server/
│   │   │   └── server.go        # HTTP/WebSocket server
│   │   └── storage/
│   │       └── storage.go       # PostgreSQL storage layer
│   ├── go.mod
│   └── go.sum
├── frontend/
│   └── index.html               # Single-page frontend
├── analytics/
│   ├── consumer.go              # Kafka analytics consumer
│   └── go.mod
└── README.md
```

## 📋 Prerequisites

- **Go 1.21+** - [Download](https://go.dev/dl/)
- **PostgreSQL** (optional) - For persistent storage
- **Kafka** (optional) - For analytics event streaming
- **Modern web browser** - Chrome, Firefox, Safari, or Edge

## 🚀 Installation

### 1. Clone the Repository

```bash
git clone <repository-url>
cd Emittr
```

### 2. Install Backend Dependencies

```bash
cd backend
go mod download
```

### 3. Install Analytics Consumer Dependencies (Optional)

```bash
cd ../analytics
go mod download
```

## 🎯 How to Play

### Game Rules

**Objective**: Connect 4 of your discs in a row (horizontal, vertical, or diagonal) before your opponent.

**How to Play**:
1. Enter your username and click "Connect"
2. Wait for an opponent (or wait 10 seconds for bot to join)
3. Click on any column to drop your disc
4. Discs fall to the lowest available space in that column
5. First player to connect 4 discs wins!

**Winning Conditions**:
- ✅ **Horizontal**: 4 discs in a row left-to-right
- ✅ **Vertical**: 4 discs stacked top-to-bottom
- ✅ **Diagonal**: 4 discs diagonally (both directions)

**Your Color**: Red discs  
**Opponent Color**: Yellow discs

### Playing Against Bot

1. Enter username and connect
2. Wait 10 seconds - bot will automatically join
3. Play your moves - bot responds immediately
4. Bot uses strategic moves (blocks wins, takes wins, prefers center)

### Playing Against Another Player

1. Open two browser tabs/windows
2. Enter different usernames in each
3. Both click "Connect"
4. You'll be matched immediately
5. Take turns making moves

## ⚙️ Configuration

### Backend Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Server address and port |
| `BOT_DELAY` | `10` | Seconds to wait before bot joins |
| `RECONNECT_WINDOW` | `30` | Seconds before forfeiting disconnected players |
| `POSTGRES_URL` | - | PostgreSQL connection string (optional) |
| `KAFKA_BROKERS` | - | Kafka broker addresses (optional) |
| `KAFKA_TOPIC` | `game-events` | Kafka topic name |

### Example Configuration

```bash
export ADDR=:8080
export BOT_DELAY=10
export RECONNECT_WINDOW=30
export POSTGRES_URL="postgres://user:password@localhost:5432/emittr"
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="game-events"
```

### Analytics Consumer Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKER` | `localhost:9092` | Kafka broker address |
| `KAFKA_TOPIC` | `game-events` | Kafka topic name |

## 🏃 Running Locally

### Step 1: Start Backend Server

```bash
cd backend
go run ./cmd/server
```

You should see:
```
server listening on :8080
```

### Step 2: Open Frontend

**Option A: Direct File**
- Open `frontend/index.html` in your browser
- Works if backend is on `localhost:8080`

**Option B: Local Server (Recommended)**
```bash
cd frontend
python -m http.server 3000
# Or with Node.js:
npx http-server . -p 3000
```
Then open: `http://localhost:3000`

### Step 3: (Optional) Start Analytics Consumer

```bash
cd analytics
go run consumer.go
```

## 📡 API Documentation

### HTTP Endpoints

#### Health Check
```
GET /health
```
**Response:**
```json
{
  "status": "ok"
}
```

#### Leaderboard
```
GET /leaderboard
```
**Response:**
```json
[
  {
    "username": "player1",
    "wins": 5
  },
  {
    "username": "player2",
    "wins": 3
  }
]
```

### WebSocket Endpoint

#### Connect to Game
```
GET /ws?username=YOUR_USERNAME&gameId=GAME_ID
```

**Query Parameters:**
- `username` (required) - Your username
- `gameId` (optional) - Game ID to rejoin existing game

**Client → Server Messages:**

```json
{
  "type": "move",
  "column": 3
}
```

**Server → Client Messages:**

**Waiting for Opponent:**
```json
{
  "type": "waiting",
  "message": "waiting for opponent"
}
```

**Game Initialized:**
```json
{
  "type": "init",
  "gameId": "uuid-here",
  "board": [[0,0,0,...], ...],
  "turn": 1,
  "you": "player1",
  "slot": 1,
  "opponent": "bot",
  "status": "active",
  "winner": null,
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**Game State Update:**
```json
{
  "type": "state",
  "board": [[0,0,0,...], ...],
  "turn": 2,
  "status": "active",
  "winner": null
}
```

**Game Finished:**
```json
{
  "type": "state",
  "board": [[1,1,1,1,...], ...],
  "turn": 1,
  "status": "finished",
  "winner": "player1"
}
```

**Error:**
```json
{
  "type": "error",
  "message": "not your turn"
}
```

## 🚢 Deployment

### Backend Deployment

#### Option 1: Build Binary and Deploy

```bash
cd backend
go build -o server ./cmd/server
# Upload server binary to your server
./server
```

#### Option 2: Docker

Create `backend/Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

Build and run:
```bash
docker build -t emittr-backend ./backend
docker run -p 8080:8080 -e ADDR=:8080 emittr-backend
```

#### Option 3: Platform Services

**Railway:**
1. Connect GitHub repo
2. Set root directory: `backend`
3. Set start command: `go run ./cmd/server`
4. Add environment variables
5. Deploy

**Render:**
1. New Web Service
2. Build: `cd backend && go build -o server ./cmd/server`
3. Start: `./server`
4. Configure environment variables

### Frontend Deployment

#### Update Backend URLs

Before deploying, update `frontend/index.html`:

```javascript
// Change from localhost to your backend URL
const url = `wss://your-backend-url.com/ws?...`;
const res = await fetch(`https://your-backend-url.com/leaderboard`);
```

#### Deploy to Netlify/Vercel

1. Upload `frontend` folder or connect GitHub repo
2. Set publish directory: `frontend`
3. Deploy

### Database Setup (Optional)

If using PostgreSQL:

```sql
CREATE DATABASE emittr;

-- Tables are created automatically by the application
-- The server will create the 'games' table on startup
```

### Kafka Setup (Optional)

1. Install and start Kafka
2. Create topic: `game-events`
3. Configure `KAFKA_BROKERS` environment variable
4. Start analytics consumer

## 💻 Development

### Project Setup

```bash
# Clone repository
git clone <repo-url>
cd Emittr

# Install backend dependencies
cd backend
go mod download

# Install analytics dependencies
cd ../analytics
go mod download
```

### Running in Development Mode

```bash
# Terminal 1: Backend
cd backend
go run ./cmd/server

# Terminal 2: Frontend (if using local server)
cd frontend
python -m http.server 3000

# Terminal 3: Analytics Consumer (optional)
cd analytics
go run consumer.go
```

### Code Structure

- **`backend/cmd/server/main.go`** - Application entry point, configuration
- **`backend/internal/server/server.go`** - HTTP/WebSocket handlers
- **`backend/internal/game/manager.go`** - Game lifecycle management
- **`backend/internal/game/board.go`** - Board logic and win detection
- **`backend/internal/game/bot.go`** - Bot AI implementation
- **`backend/internal/storage/storage.go`** - Database persistence
- **`backend/internal/analytics/producer.go`** - Kafka event publishing
- **`frontend/index.html`** - Single-page frontend application
- **`analytics/consumer.go`** - Analytics event consumer

## 🧪 Testing

### Manual Testing

1. **Two Player Game:**
   - Open two browser tabs
   - Use different usernames
   - Verify real-time updates

2. **Bot Game:**
   - Connect with username
   - Wait 10 seconds
   - Verify bot joins and plays

3. **Reconnection:**
   - Start a game
   - Close browser tab
   - Reopen and reconnect with same username
   - Verify game state restored

4. **Leaderboard:**
   - Play several games
   - Check leaderboard updates
   - Verify win counts

### API Testing

```bash
# Health check
curl http://localhost:8080/health

# Leaderboard
curl http://localhost:8080/leaderboard
```

## 🔧 Troubleshooting

### Backend Won't Start

**Issue**: Port already in use
```bash
# Change port
export ADDR=:8081
```

**Issue**: Go not found
```bash
# Verify Go installation
go version

# Add Go to PATH if needed
export PATH=$PATH:/usr/local/go/bin
```

### Frontend Can't Connect

**Issue**: WebSocket connection fails
- Verify backend is running on port 8080
- Check browser console for errors
- Ensure CORS is configured if using different ports

**Issue**: Leaderboard shows "Loading..."
- Check backend is running
- Verify `/leaderboard` endpoint works: `curl http://localhost:8080/leaderboard`
- Check browser console for errors

### Database Issues

**Issue**: PostgreSQL connection fails
- Verify PostgreSQL is running
- Check connection string format
- Ensure database exists
- Server will fall back to in-memory storage if DB unavailable

### Kafka Issues

**Issue**: Analytics not working
- Verify Kafka is running
- Check broker address
- Ensure topic exists
- Analytics is optional - game works without it

## 📝 Notes

- **No Authentication**: Usernames are trusted (no validation)
- **In-Memory State**: Active games stored in memory (lost on restart)
- **Persistence**: Only completed games are saved to database
- **Graceful Degradation**: Works without PostgreSQL/Kafka
- **Single Frontend File**: All frontend code in one HTML file

## 🎯 Future Enhancements

- User authentication and authorization
- Room-based matchmaking
- Spectator mode
- Game replay/history
- Advanced bot difficulty levels
- Mobile-responsive improvements
- Real-time chat
- Tournament mode

## 📄 License

[Add your license here]

## 👥 Contributing

[Add contribution guidelines if applicable]

---

**Built with ❤️ using Go and vanilla JavaScript**

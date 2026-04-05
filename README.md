# 🔮 Omnigram

**Telegram Mini App for semantic search across your chats.**

Omnigram connects to your Telegram account via MTProto, enriches conversations (groups, channels, 1-1 chats) into a vector database, and lets you search by meaning — not just keywords — using local embeddings and Google Gemini as the LLM brain.

AES-256-GCM encryption is used for telegram token and message encryption with frontend and backend secrets. If you clean your frontend local storage you will not be able to decrypt your messages.

---

## ✨ Features

- **Telegram MTProto Authentication** — Log in with phone number + code (2FA supported)
- **Dialog Discovery** — Browse your recent chats, groups, channels, and bots
- **Chat Enrichment** — Download full message history and vectorize it with context windows
- **Semantic Search** — Find messages by meaning using vector similarity (pgvector)
- **AI-Powered Analysis** — Gemini 2.5 Flash analyzes search results and provides concise answers
- **End-to-End Encryption** — All stored message data is AES-256-GCM encrypted at rest
- **Local Embeddings** — Runs `nomic-embed-text` via Ollama, no data leaves your server
- **Direct Message Links** — Search results link back to the original Telegram messages
- **Multi-Session Support** — Each session is independently encrypted and isolated

---

## 🏗 Architecture

```
┌──────────────────────────────────────────────────────┐
│                    Docker Compose                    │
├──────────────┬──────────────┬──────────┬─────────────┤
│   Nginx      │  Go API      │ Postgres │   Ollama    │
│  (Frontend)  │  (Backend)   │ pgvector │ (Embeddings)│
│  React+Vite  │  Chi Router  │          │ nomic-embed │
│  Port 8080   │  Port 8080   │ Port 5432│ Port 11434  │
│              │  (internal)  │          │             │
└──────────────┴──────────────┴──────────┴─────────────┘
                       │
                       ▼
              Google Gemini API
             (Search Analysis)
```

| Component | Technology |
|-----------|-----------|
| Frontend | React 19 + TypeScript + Vite |
| Backend | Go 1.25 + Chi router + gotd/td (MTProto) |
| Database | PostgreSQL 15 + pgvector |
| Embeddings | Ollama + nomic-embed-text (local) |
| LLM | Google Gemini 2.5 Flash |
| Auth | JWT + AES-256-GCM session encryption |
| Reverse Proxy | Nginx |

---

## 📋 Prerequisites

- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- **Telegram API credentials** — Get `API_ID` and `API_HASH` from [my.telegram.org](https://my.telegram.org/)
- **Google Gemini API key** — Get it from [AI Studio](https://aistudio.google.com/app/api-keys)

---

## 🚀 Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/scroller123/omnigram.git
cd omnigram
```

### 2. Configure environment

```bash
cp .env.example .env
```

Edit `.env` and fill in the required values:

```env
# Telegram credentials (required)
API_ID=12345678
API_HASH=your_api_hash_here

# Gemini LLM key (required for search analysis)
GEMINI_API_KEY=your_gemini_key_here

# Security — generate with: openssl rand -hex 32
ENCRYPTION_SECRET=your_64_hex_char_secret
JWT_SECRET=your_jwt_secret

# Database (defaults work out of the box)
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=omnigram
```

### 3. Start with Docker Compose

```bash
docker compose up --build -d
```

This starts all four services:
- **Nginx + Frontend** → `http://localhost:8080`
- **Go API** → internal, proxied via Nginx
- **PostgreSQL + pgvector** → `localhost:35533`
- **Ollama** → auto-pulls and configures `nomic-embed-text`

### 4. Open the app

Navigate to **http://localhost:8080** in your browser.

---

## 📖 How to Use

### Step 1 — Log In

1. Open the app and enter your **phone number** (international format, e.g. `+1234567890`)
2. Enter the **verification code** sent to your Telegram
3. If you have **2FA** enabled, enter your cloud password

### Step 2 — Browse Dialogs

After login, the **Dialogs Table** shows your recent conversations:
- 👥 Groups
- 📢 Channels / Supergroups
- 👤 Private chats (1-1)
- 🤖 Bots

### Step 3 — Enrich a Chat

Click the **"Enrich"** button next to any dialog. This will:

1. Download the full message history from Telegram
2. Build **5-message context windows** for each message (includes author + timestamp)
3. Generate **vector embeddings** locally via Ollama
4. Encrypt and store everything in PostgreSQL with pgvector

> [!NOTE]
> Enrichment runs in the background. You can track progress in real time via the enrichment tasks panel. Large chats may take several minutes.

### Step 4 — Search by Meaning

Once a chat is enriched, use the **search bar** to ask questions in natural language:

- *"When did we discuss the deployment schedule?"*
- *"What did Alex say about the budget?"*
- *"Find messages about the API redesign"*

**What happens under the hood:**

1. Your query is embedded into a vector using the same model
2. pgvector finds the **top 100 most similar** message contexts
3. Matched messages are decrypted and sent to **Gemini 2.5 Flash**
4. Gemini analyzes the messages and returns a concise answer
5. You also see the raw matched messages with links to Telegram

---

## 🛠 Local Development

### Backend (Go API)

```bash
cd server
go run main.go
```

Requires a running PostgreSQL and Ollama instance. The backend listens on `:8080` and auto-loads `.env` from the project root.

**Swagger UI** is available at `http://localhost:8080/swagger/` when running locally.

### Frontend (React + Vite)

```bash
cd mini-app
npm install
npm run dev
```

The dev server starts on `http://localhost:5173` and proxies API calls to the backend.

---

## 🔌 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/send-code` | Send login code to phone |
| `POST` | `/api/auth/verify-code` | Verify code and get JWT |
| `POST` | `/api/auth/verify-password` | Submit 2FA password |
| `GET`  | `/api/auth/me` | Get current user info |
| `POST` | `/api/auth/logout` | Log out and destroy session |
| `GET`  | `/api/dialogs` | List recent Telegram dialogs |
| `GET`  | `/api/channels` | List grabbed channels |
| `GET`  | `/api/channels/{id}/messages` | Get messages for a channel |
| `POST` | `/api/grab` | Grab channel by username |
| `POST` | `/api/grab_by_id` | Grab chat by ID |
| `POST` | `/api/grab_invite` | Grab chat by invite link hash |
| `POST` | `/api/enrich` | Start enrichment for a dialog |
| `GET`  | `/api/enrich/tasks` | Get enrichment task progress |
| `POST` | `/api/search` | Semantic search across messages |

> All endpoints (except auth) require `Authorization: Bearer <token>` and `X-Session-ID` headers.

---

## 🗂 Project Structure

```
omnigram/
├── docker-compose.yml        # Full stack orchestration
├── .env.example              # Environment template
├── server/                   # Go backend
│   ├── main.go               # Entry point
│   ├── api/                  # HTTP handlers + router
│   ├── ai/                   # Gemini client + Ollama embeddings
│   ├── auth/                 # JWT, AES encryption, middleware
│   ├── db/                   # PostgreSQL + pgvector repository
│   ├── tg/                   # Telegram MTProto client + message grabber
│   ├── docs/                 # Swagger auto-generated docs
│   └── Dockerfile
├── mini-app/                 # React frontend
│   ├── src/
│   │   ├── App.tsx           # Main app with auth flow
│   │   ├── components/       # LoginForm, DialogsTable, etc.
│   │   ├── api/              # API client functions
│   │   └── telegram/         # Telegram-specific utilities
│   ├── nginx/app.conf        # Nginx reverse proxy config
│   └── Dockerfile
└── ollama/                   # Embedding model setup
    ├── Modelfile             # nomic-embed-text config
    └── entrypoint.sh         # Auto-pull + model creation
```

---

## 🔐 Security

- **Message encryption** — All message content is encrypted with AES-256-GCM before storage. Each session derives its own encryption key from the base `ENCRYPTION_SECRET`.
- **Session isolation** — Multi-session architecture ensures data from different sessions is cryptographically separated.
- **JWT authentication** — All API requests are authenticated via signed JWT tokens.
- **No plaintext storage** — Raw message text is never stored in the database; only encrypted blobs and vector embeddings.

---

## 📝 License

This project is for personal / educational use.

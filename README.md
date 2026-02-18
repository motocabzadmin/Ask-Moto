# Ask Moto

A conversational Q&A service for the Moto ride-hailing app. Ask Moto answers user questions about Moto's features, pricing modes, and functionality using a knowledge base retrieval system.

## Features

- **Intent Classification**: Rule-based detection of user intents (pricing modes, launch info, customization, etc.)
- **Knowledge Base Retrieval**: Keyword-based semantic search over markdown documentation
- **Safe Default Responses**: Canonical answers for high-confidence intents
- **Analytics Logging**: Query and feedback tracking for continuous improvement
- **REST API**: Simple HTTP endpoints for chat, feedback, and analytics

## Architecture

This project follows **Hexagonal Architecture** (Ports and Adapters), ensuring clean separation between business logic and infrastructure concerns.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Primary Adapters                         │
│                      (Driving / Input)                          │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    HTTP Handler                          │   │
│  │                   /api/chat, etc.                        │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Input Ports                             │
│              (ChatService, FeedbackService, etc.)               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                            Core                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    Domain Models                         │   │
│  │         Question, Response, Intent, Chunk, etc.          │   │
│  └─────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                  Application Services                    │   │
│  │           ChatService, FeedbackService, etc.             │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Output Ports                             │
│       (KnowledgeBaseRepository, Retriever, Logger, etc.)        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Secondary Adapters                         │
│                      (Driven / Output)                          │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │
│  │  FileStore │ │  Keyword   │ │   File     │ │   Intent   │   │
│  │    (KB)    │ │  Retriever │ │   Logger   │ │ Classifier │   │
│  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
ask-moto/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Environment configuration loader
│   ├── core/
│   │   ├── domain/
│   │   │   └── models.go           # Domain entities and value objects
│   │   ├── ports/
│   │   │   ├── input.go            # Input port interfaces (use cases)
│   │   │   └── output.go           # Output port interfaces (repositories)
│   │   └── services/
│   │       ├── chat.go             # Chat service implementation
│   │       ├── feedback.go         # Feedback service implementation
│   │       └── analytics.go        # Analytics service implementation
│   └── adapters/
│       ├── primary/
│       │   └── http/
│       │       └── handler.go      # HTTP REST API handler
│       └── secondary/
│           ├── knowledgebase/
│           │   └── filestore.go    # Markdown file-based KB storage
│           ├── retrieval/
│           │   └── keyword.go      # Keyword-based retrieval
│           ├── logging/
│           │   └── filelogger.go   # JSONL file logger
│           └── intent/
│               └── classifier.go   # Rule-based intent classifier
├── data/
│   ├── kb/                         # Knowledge base markdown files
│   └── logs/                       # Analytics logs (JSONL)
├── .env                            # Environment configuration
├── .env.example                    # Configuration template
├── go.mod
└── go.sum
```

## Getting Started

### Prerequisites

- Go 1.21 or higher

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/moto/ask-moto.git
   cd ask-moto
   ```

2. Copy the environment configuration:
   ```bash
   cp .env.example .env
   ```

3. Build the application:
   ```bash
   go build -o bin/ask-moto ./cmd/server
   ```

4. Run the server:
   ```bash
   ./bin/ask-moto
   ```

The server starts on port 8080 by default.

### Docker

#### Using Docker Compose (Recommended)

```bash
# Build and run
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down
```

#### Using Docker directly

```bash
# Build the image
docker build -t ask-moto .

# Run the container
docker run -d \
  --name ask-moto \
  -p 8080:8080 \
  -v $(pwd)/data/logs:/app/data/logs \
  -v $(pwd)/data/kb:/app/data/kb:ro \
  ask-moto

# View logs
docker logs -f ask-moto

# Stop and remove
docker stop ask-moto && docker rm ask-moto
```

#### Environment Variables with Docker

Override configuration using environment variables:

```bash
docker run -d \
  --name ask-moto \
  -p 3000:3000 \
  -e PORT=3000 \
  -e TOP_K=10 \
  -e SAFE_DEFAULT_CONFIDENCE=0.8 \
  -v $(pwd)/data/logs:/app/data/logs \
  -v $(pwd)/data/kb:/app/data/kb:ro \
  ask-moto
```

Or with Docker Compose, create a `.env` file and it will be automatically loaded.

## API Documentation

Interactive API documentation is available via Swagger UI:

- **Swagger UI**: http://localhost:8080/swagger/index.html

The Swagger documentation is auto-generated using [swag](https://github.com/swaggo/swag). To regenerate after modifying API annotations:

```bash
swag init -g cmd/server/main.go -o docs
```

The generated files are in the `docs/` directory.

## API Endpoints

### POST /api/chat

Send a question to Ask Moto.

**Request:**
```json
{
  "message": "What is Moto?",
  "user_id": "optional-user-id"
}
```

**Response:**
```json
{
  "response": "Moto is a ride-hailing app that helps riders request rides, choose pricing modes, and helps drivers receive ride orders.",
  "answer_type": "kb_answer",
  "confidence": 1.0,
  "response_id": "uuid-here"
}
```

### POST /api/feedback

Submit feedback on a response.

**Request:**
```json
{
  "response_id": "uuid-from-chat-response",
  "rating": 2,
  "comment": "Very helpful!"
}
```

- `rating`: 1 = thumbs down, 2 = thumbs up

**Response:**
```json
{
  "status": "ok"
}
```

### GET /api/stats

Get analytics statistics.

**Response:**
```json
{
  "total_queries": 100,
  "answered_queries": 85,
  "unknown_queries": 15,
  "answer_rate": 0.85,
  "total_feedback": 50,
  "positive_feedback": 45,
  "negative_feedback": 5
}
```

### GET /health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy"
}
```

## Configuration

All configuration is managed through environment variables. Copy `.env.example` to `.env` and adjust as needed.

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `KB_PATH` | `data/kb` | Path to knowledge base directory |
| `KB_VERSION` | `v1` | Knowledge base version identifier |
| `LOGS_PATH` | `data/logs` | Path to analytics logs directory |
| `TOP_K` | `5` | Number of chunks to retrieve |
| `MIN_RETRIEVAL_SCORE` | `1.0` | Minimum score threshold for retrieval |
| `INTENT_MIN_CONFIDENCE` | `0.3` | Minimum confidence for intent detection |
| `INTENT_MATCH_THRESHOLD` | `0.7` | Token overlap threshold for intent matching |
| `SAFE_DEFAULT_CONFIDENCE` | `0.7` | Confidence threshold for safe default responses |
| `QUESTION_OVERLAP_THRESHOLD` | `0.7` | Token overlap for Q/A matching |
| `SENTENCE_RELEVANCE_THRESHOLD` | `0.4` | Sentence relevance threshold |
| `KEYWORD_BOOST_HIGH` | `3.0` | Boost for high-priority keywords |
| `KEYWORD_BOOST_MEDIUM` | `2.5` | Boost for medium-priority keywords |
| `KEYWORD_BOOST_LOW` | `1.5` | Boost for low-priority keywords |

## Knowledge Base

The knowledge base consists of markdown files in `data/kb/`. Each file represents a module:

- `overview.md` - General information about Moto
- `pricing-modes.md` - Instant, Flex, and Auto pricing modes
- `customization.md` - Ride customization options
- `account.md` - Account management
- `saved-places.md` - Saved places feature
- `ride-request.md` - How to request rides

### Adding Content

Create markdown files with Q&A format:

```markdown
# Module Title

## Section

**Q: Your question here? A: Your answer here.**

Or use multi-line format:

Q: Another question?
A: The answer spans
multiple lines if needed.
```

## Supported Intents

| Intent | Description |
|--------|-------------|
| `launch_date` | When is Moto launching? |
| `launch_location` | Where is Moto available? |
| `what_is_moto` | What is Moto? |
| `pricing_instant` | Instant pricing mode |
| `pricing_flex` | Flex pricing mode |
| `pricing_auto` | Auto pricing mode |
| `auto_bidding` | Clarification about Auto (not bidding) |
| `customization` | Customization options |
| `saved_places` | Saved places feature |
| `request_ride` | How to request a ride |

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/ask-moto ./cmd/server
```

### Adding New Adapters

The hexagonal architecture makes it easy to swap implementations:

1. **New Primary Adapter** (e.g., gRPC): Implement handlers that call the input ports in `internal/core/ports/input.go`

2. **New Secondary Adapter** (e.g., PostgreSQL): Implement the output port interfaces in `internal/core/ports/output.go`

3. **Wire up in main.go**: Inject the new adapter implementations

## License

MIT License

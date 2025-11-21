# 🎬 Letterboxle 🍿 (backend)

![github actions status](https://github.com/pedroaguia8/Letterboxle-backend/actions/workflows/ci.yml/badge.svg)

A daily movie guessing game inspired by Wordle. Each day, players try to identify a mystery movie using progressively revealed hints — from tagline to genre to director to cast.


**[Play now at letterboxle.pedroaguia8.dev](https://letterboxle.pedroaguia8.dev)**

<div style="text-align: center;">
  <img src="docs/images/Inception_gameplay.png" width="50%" alt="Inception (2010)">
</div>

---

## Motivation

I built Letterboxle to sharpen my full-stack development skills while creating something I'd actually want to use every day. The project gave me hands-on experience with Go backend development, PostgreSQL, external API integration (TMDB), and deploying a production application with CI/CD pipelines.

---

## 🚀 Quick Start

### Play Online

Head to **[letterboxle.pedroaguia8.dev](https://letterboxle.pedroaguia8.dev)** and start guessing!

### How to Play

1. A new mystery movie is revealed each day
2. You get **6 guesses** to identify the film
3. With each wrong guess (or skip), a new hint is revealed:
    - **Guess 1:** Tagline
    - **Guess 2:** Genre
    - **Guess 3:** Director
    - **Guess 4:** Actor 1
    - **Guess 5:** Actor 2
    - **Guess 6:** Year
4. Search for movies by title and select your guess
5. Share your results with friends!

---

## 📖 Usage

### Tech Stack

**Backend (this repo):**
- **Go** — HTTP standard library server 
- **PostgreSQL** — Movie database with daily puzzle scheduling
- **sqlc** — Type-safe SQL query generation
- **goose** — Database migrations
- **TMDB API** — Movie poster fetching
- **Docker** — Containerized deployment
- **GitHub Actions** — CI/CD pipeline with automated deployment

**Frontend**:
- **React** — Component-based UI
- **Vite** — Build tooling

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /api/movie_of_the_day/today` | Returns today's movie puzzle with hints |
| `GET /api/movies?search_query={query}` | Searches movie database for autocomplete |

### Architecture Highlights

- **Background worker** fetches and caches movie posters from TMDB every 4 hours
- **Fuzzy search** with PostgreSQL `ILIKE` patterns for flexible movie matching
- **Graceful shutdown** ensures clean database disconnection on deploy
- **Goose migrations** for database schema versioning

---

## 🤝 Contributing

### Prerequisites

- Go 1.25+
- PostgreSQL
- TMDB API key ([get one free](https://www.themoviedb.org/settings/api))

### Clone and Setup

```bash
git clone https://github.com/pedroaguia8/Letterboxle-backend.git
cd Letterboxle-backend
```

### Install Dependencies

```bash
go mod download
```

### Create the Database
```bash
# Connect to PostgreSQL and create the database
psql -U postgres
CREATE DATABASE letterboxle;
\q
```

### Configure Environment

Create a `.env` file:

```bash
DB_URL=postgres://user:password@localhost:5432/letterboxle?sslmode=disable
TMDB_API_KEY=your_tmdb_api_key
PLATFORM=dev
PORT=8080
```

### Run Database Migrations

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir sql/schema postgres "$DB_URL" up
```

### Build and Run

```bash
go build -o app
./app
```

### Run with Docker

```bash
docker compose up --build
```

### Run Tests

```bash
go test ./... -cover
```

### Code Quality

```bash
# Formatting
go fmt ./...

# Linting
staticcheck ./...

# Security
gosec ./...
```

### Submit a Pull Request

Fork the repository and open a pull request to the `main` branch. All PRs run through CI checks for tests, formatting, linting, and security scanning.


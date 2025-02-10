# UpGist 📤

Self-hosted Gist uploader with SSH auth. Built with Go + HTMX for maximum performance.

## ✨ Features

- 🚀 Ultra-lightweight (~15MB Docker image)
- 🔒 SSH key authentication
- 📁 Multiple file uploads
- ⚡️ Pure HTMX frontend, no JS
- 🔄 Progress indicators

## 🏃 Quick Start

1. Create a new gist:
   - Go to https://gist.github.com/
   - Create a new gist (can be empty)
   - Copy the SSH clone URL (e.g., `git@gist.github.com:abc123.git`)
   - Make sure your SSH key is added to GitHub

2. Configure environment:
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

3. Run UpGist:
   ```bash
   docker compose up -d
   ```

Access at `http://localhost:3000`

## 🔧 Environment

Required:
- `GIST_URL`: The SSH clone URL of your gist
- `GITHUB_USERNAME`: Your GitHub username (needed for raw file URLs)

Optional:
- `GIT_USER`: Git user name for commits (default: "UpGist")
- `GIT_EMAIL`: Git user email for commits (default: "upgist@local")
- `GIT_COMMIT_MESSAGE`: Custom commit message (default: "Add files via UpGist")
- `UPGIST_LOGGING`: Enable debug logging

## 💻 Development

```bash
# Copy and configure environment
cp .env.example .env
# Edit .env with your settings

# Run locally with Go
go run main.go

# Build binary
go build -o upgist

# Run with all options
go run main.go

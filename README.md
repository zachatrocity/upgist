# UpGist 📤

Self-hosted Gist uploader with SSH auth. Built with Rust + HTMX for maximum performance.

## ✨ Features

- 🚀 Ultra-lightweight (~15MB Docker image)
- 🔒 SSH key authentication
- 📁 Multiple file uploads
- ⚡️ Pure HTMX frontend, no JS
- 🔄 Progress indicators

## 🏃 Quick Start

```bash
# Update existing gist
GIST_ID=your_gist_id docker compose up -d

# Create new gists
docker compose up -d
```

Access at `http://localhost:3000`

## 🔧 Environment

- `GIST_ID`: (Optional) Existing gist ID to update
- `RUST_LOG`: (Optional) Log level, defaults to "info"

## 💻 Development

```bash
docker compose -f docker-compose.dev.yml up
```

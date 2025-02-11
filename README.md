# UpGist 📤

Self-hosted GitHub Gist file uploader with SSH auth built with Go + HTMX.

## ✨ Features

- 🚀 Ultra-lightweight
- 🔒 SSH key authentication
- 📁 Multiple file uploads
- ⚡️ Pure HTMX frontend

## 🏃 Quick Start

1. Create a new gist:
   - Go to https://gist.github.com/ and make a new gist
   - Copy the SSH clone URL (e.g., `git@gist.github.com:abc123.git`)
   - Make sure your SSH key is added to GitHub and `gist.github.com` is in your `known_hosts` file.
     - `ssh-keyscan gist.github.com >> ~/.ssh/known_hosts`

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

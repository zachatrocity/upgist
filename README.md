# UpGist 📤

<p align="center">
  <img src="https://gist.githubusercontent.com/zachatrocity/e0246929ef65bb738bcf7a74c42b1bbf/raw/86e098e82f2d30bc731bffe60d9e364ca4c4f60b/upgist.png" alt="upgist logo">
</p>

Self-hosted GitHub Gist file uploader with SSH auth built with Go + HTMX.

## ✨ Features

- 🚀 Ultra-lightweight
- 🔒 SSH key authentication
- 📁 Multiple file uploads
- ⚡️ Pure HTMX frontend
- 🌘 css-scope-inline

## 🏃 Quick Start

1. Create a new gist:
   - Go to https://gist.github.com/ and make a new gist
   - Copy the SSH clone URL (e.g., `git@gist.github.com:abc123.git`)
   - Make sure your SSH key is added to GitHub and `gist.github.com` is in your `known_hosts` file.
     - `ssh-keyscan gist.github.com >> ~/.ssh/known_hosts`

2. `docker-compose.yml`
   ```docker
   services:
      upgist:
         image: ghcr.io/zachatrocity/upgist:main
         ports:
            - "3000:3000"
         volumes:
            # Mount SSH keys for container's internal SSH agent
            # Note: gist.github.com needs to be in known_hosts
            - ~/.ssh:/root/.ssh:ro
         environment:
            - GIST_URL=change_me
            - GITHUB_USERNAME=change_me
            # see .env.example for all vars
    init: true

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

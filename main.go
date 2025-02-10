package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Environment variables and defaults
var (
	gistURL      = os.Getenv("GIST_URL")
	githubUser   = os.Getenv("GITHUB_USERNAME")
	gitUser      = getEnvOrDefault("GIT_USER", "UpGist")
	gitEmail     = getEnvOrDefault("GIT_EMAIL", "upgist@local")
	commitMsg    = getEnvOrDefault("GIT_COMMIT_MESSAGE", "Add files via UpGist")
	debugEnabled = os.Getenv("UPGIST_LOGGING") != ""
)

// Debug logger
type debugLogger struct{ enabled bool }

func (l *debugLogger) Printf(format string, v ...interface{}) {
	if l.enabled {
		log.Printf(format, v...)
	}
}

var debugLog = &debugLogger{enabled: debugEnabled}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func configureGit(dir string) error {
	cmd := exec.Command("git", "config", "user.name", gitUser)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git user name: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", gitEmail)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git user email: %v", err)
	}
	return nil
}

func main() {
	// Verify required environment variables
	if gistURL == "" {
		log.Fatal("GIST_URL environment variable is required")
	}
	if githubUser == "" {
		log.Fatal("GITHUB_USERNAME environment variable is required")
	}

	// Setup routes
	http.HandleFunc("/upload", handleUpload)
	http.Handle("/", http.FileServer(http.Dir("static")))

	// Start server
	addr := ":3000"
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	debugLog.Printf("Received upload request")

	// Create temp directory for git operations
	tempDir, err := os.MkdirTemp("", "upgist-*")
	if err != nil {
		debugLog.Printf("Failed to create temp directory: %v", err)
		http.Error(w, "Failed to create temp directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Clone and configure git repository
	if err := cloneAndConfigureRepo(tempDir); err != nil {
		debugLog.Printf("Git setup failed: %v", err)
		http.Error(w, fmt.Sprintf("Git setup failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Process uploaded files
	uploadedFiles, err := processFiles(tempDir, r)
	if err != nil {
		debugLog.Printf("File processing failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Push changes
	if err := pushChanges(tempDir); err != nil {
		debugLog.Printf("Failed to push changes: %v", err)
		http.Error(w, "Failed to push changes", http.StatusInternalServerError)
		return
	}

	// Get file hashes for raw URLs
	fileHashes, err := getFileHashes(tempDir, uploadedFiles)
	if err != nil {
		debugLog.Printf("Failed to get file hashes: %v", err)
		http.Error(w, "Failed to generate file links", http.StatusInternalServerError)
		return
	}

	// Return success response with file links
	w.Header().Set("Content-Type", "text/html")
	gistWebURL := strings.TrimSuffix(gistURL, ".git")
	gistWebURL = strings.Replace(gistWebURL, "git@gist.github.com:", "https://gist.github.com/", 1)
	gistID := strings.Split(gistURL, ":")[1]
	gistID = strings.TrimSuffix(gistID, ".git")

	response := fmt.Sprintf(`<div class="success">
		Files uploaded successfully! <a href="%s" target="_blank">View Gist</a><br><br>
		Direct links:<br>`, gistWebURL)

	for filename, hash := range fileHashes {
		fileURL := fmt.Sprintf("https://gist.githubusercontent.com/%s/%s/raw/%s/%s",
			githubUser, gistID, hash, filename)
		response += fmt.Sprintf(`<a href="%s" target="_blank">%s</a><br>`, fileURL, filename)
	}

	response += "</div>"
	w.Write([]byte(response))
}

func getFileHashes(dir string, files []string) (map[string]string, error) {
	hashes := make(map[string]string)
	for _, filename := range files {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get commit hash: %v", err)
		}
		hash := strings.TrimSpace(string(output))
		hashes[filename] = hash
	}
	return hashes, nil
}

func cloneAndConfigureRepo(dir string) error {
	cmd := exec.Command("git", "clone", gistURL, ".")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone gist: %v, output: %s", err, string(output))
	}

	debugLog.Printf("Gist cloned successfully")
	return configureGit(dir)
}

func processFiles(dir string, r *http.Request) ([]string, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, fmt.Errorf("failed to parse form: %v", err)
	}

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return nil, fmt.Errorf("no files uploaded")
	}

	debugLog.Printf("Processing %d files", len(files))

	var uploadedFiles []string
	for _, fileHeader := range files {
		if err := saveFile(dir, fileHeader); err != nil {
			return nil, err
		}
		uploadedFiles = append(uploadedFiles, fileHeader.Filename)
	}

	if err := createCommit(dir); err != nil {
		return nil, err
	}

	return uploadedFiles, nil
}

func saveFile(dir string, fileHeader *multipart.FileHeader) error {
	debugLog.Printf("Processing file: %s", fileHeader.Filename)

	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", fileHeader.Filename, err)
	}
	defer file.Close()

	dst, err := os.Create(filepath.Join(dir, fileHeader.Filename))
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", fileHeader.Filename, err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		return fmt.Errorf("failed to save file %s: %v", fileHeader.Filename, err)
	}

	// Stage file
	cmd := exec.Command("git", "add", fileHeader.Filename)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage file %s: %v, output: %s", fileHeader.Filename, err, string(output))
	}

	debugLog.Printf("File saved and staged: %s", fileHeader.Filename)
	return nil
}

func createCommit(dir string) error {
	cmd := exec.Command("git", "commit", "-m", commitMsg)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit: %v, output: %s", err, string(output))
	}
	debugLog.Printf("Files committed successfully")
	return nil
}

func pushChanges(dir string) error {
	// Try pushing to main first
	cmd := exec.Command("git", "push", "origin", "HEAD:main")
	cmd.Dir = dir
	if err := cmd.Run(); err == nil {
		return nil
	}

	// If main fails, try master
	debugLog.Printf("Failed to push to main, trying master...")
	cmd = exec.Command("git", "push", "origin", "HEAD:master")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push to master: %v, output: %s", err, string(output))
	}

	debugLog.Printf("Changes pushed successfully")
	return nil
}

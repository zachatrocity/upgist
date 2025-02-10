package main

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		defaultVal  string
		envVal      string
		expected    string
		shouldUnset bool
	}{
		{
			name:       "returns default when env not set",
			key:        "TEST_KEY_1",
			defaultVal: "default",
			expected:   "default",
		},
		{
			name:       "returns env value when set",
			key:        "TEST_KEY_2",
			defaultVal: "default",
			envVal:     "custom",
			expected:   "custom",
		},
		{
			name:        "handles empty env value",
			key:         "TEST_KEY_3",
			defaultVal:  "default",
			envVal:      "",
			expected:    "default",
			shouldUnset: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
			}
			if tt.shouldUnset {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("getEnvOrDefault(%s, %s) = %s; want %s",
					tt.key, tt.defaultVal, result, tt.expected)
			}

			os.Unsetenv(tt.key)
		})
	}
}

func TestHandleUpload(t *testing.T) {
	// Create a temporary directory for the test git repo
	testRepoDir, err := os.MkdirTemp("", "test-repo-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testRepoDir)

	// Initialize git repo
	if err := executeCommand("git", "init", testRepoDir); err != nil {
		t.Fatal(err)
	}

	// Configure git for the test repo
	if err := configureGit(testRepoDir); err != nil {
		t.Fatal(err)
	}

	// Create a bare clone to act as remote
	remoteDir, err := os.MkdirTemp("", "remote-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(remoteDir)

	if err := executeCommand("git", "init", "--bare", remoteDir); err != nil {
		t.Fatal(err)
	}

	// Set the remote
	cmd := exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = testRepoDir
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	// Save original env vars
	originalGistURL := os.Getenv("GIST_URL")
	originalGithubUser := os.Getenv("GITHUB_USERNAME")

	// Set test env vars
	os.Setenv("GIST_URL", remoteDir)
	os.Setenv("GITHUB_USERNAME", "testuser")

	// Restore env vars after test
	defer func() {
		if originalGistURL != "" {
			os.Setenv("GIST_URL", originalGistURL)
		} else {
			os.Unsetenv("GIST_URL")
		}
		if originalGithubUser != "" {
			os.Setenv("GITHUB_USERNAME", originalGithubUser)
		} else {
			os.Unsetenv("GITHUB_USERNAME")
		}
	}()

	// Only test the HTTP aspects, not the git operations
	tests := []struct {
		name         string
		method       string
		setupForm    bool
		expectedCode int
		expectedBody string
	}{
		{
			name:         "rejects non-POST requests",
			method:       http.MethodGet,
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Method not allowed\n",
		},
		{
			name:         "handles missing form",
			method:       http.MethodPost,
			setupForm:    false,
			expectedCode: http.StatusInternalServerError,
			expectedBody: "failed to parse form: request Content-Type isn't multipart/form-data\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.setupForm {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.Close()
				req = httptest.NewRequest(tt.method, "/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req = httptest.NewRequest(tt.method, "/upload", nil)
			}

			rr := httptest.NewRecorder()
			handleUpload(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedCode)
			}

			if rr.Body.String() != tt.expectedBody {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestConfigureGit(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "upgist-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	if err := executeCommand("git", "init"); err != nil {
		t.Fatal(err)
	}

	// Test git configuration
	if err := configureGit(tempDir); err != nil {
		t.Errorf("configureGit() failed: %v", err)
	}

	// Verify configuration
	name, err := getGitConfig(tempDir, "user.name")
	if err != nil {
		t.Fatal(err)
	}
	if name != gitUser {
		t.Errorf("git user.name = %s; want %s", name, gitUser)
	}

	email, err := getGitConfig(tempDir, "user.email")
	if err != nil {
		t.Fatal(err)
	}
	if email != gitEmail {
		t.Errorf("git user.email = %s; want %s", email, gitEmail)
	}
}

// Helper function to execute git commands
func executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// Helper function to get git config values
func getGitConfig(dir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func TestSaveFile(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "upgist-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	if err := executeCommand("git", "init", tempDir); err != nil {
		t.Fatal(err)
	}

	// Create test file content
	content := "test content"
	filename := "test.txt"

	// Create multipart file header
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(part, strings.NewReader(content))
	writer.Close()

	// Parse multipart form
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		t.Fatal(err)
	}

	// Get file header
	file := req.MultipartForm.File["file"][0]

	// Configure git for the test repo
	if err := configureGit(tempDir); err != nil {
		t.Fatal(err)
	}

	// Test saveFile
	if err := saveFile(tempDir, file); err != nil {
		t.Errorf("saveFile() failed: %v", err)
	}

	// Verify file was saved correctly
	savedContent, err := os.ReadFile(filepath.Join(tempDir, filename))
	if err != nil {
		t.Fatal(err)
	}

	if string(savedContent) != content {
		t.Errorf("saved content = %s; want %s", string(savedContent), content)
	}

	// Verify file was staged
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = tempDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "A  "+filename) {
		t.Errorf("file not staged correctly, git status: %s", string(out))
	}
}

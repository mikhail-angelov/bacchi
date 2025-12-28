// Package backup handles the creation and management of backup archives.
package backup

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

// Engine handles the creation and management of backup archives.
type Engine struct {
	TempDir string
}

// NewEngine creates a new backup engine with a temporary directory workspace.
func NewEngine(tempDir string) *Engine {
	return &Engine{TempDir: tempDir}
}

// CreateArchive creates a tar.gz archive of the specified folders, supporting incremental backups with GNU tar.
func (e *Engine) CreateArchive(name string, folders []string, exclude []string, snapshotFile string) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	archiveName := fmt.Sprintf("%s_%s.tar.gz", name, timestamp)
	archivePath := filepath.Join(e.TempDir, archiveName)

	args := []string{"-czf", archivePath}

	if snapshotFile != "" {
		args = append(args, "--listed-incremental", snapshotFile)
	}

	for _, pattern := range exclude {
		args = append(args, "--exclude", pattern)
	}

	args = append(args, folders...)

	cmd := exec.Command("tar", args...) // #nosec G204
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar failed: %w, output: %s", err, string(output))
	}

	return archivePath, nil
}

// Encrypt encrypts a file using GPG symmetric encryption with a passphrase.
func (e *Engine) Encrypt(filePath string, passphrase string) (string, error) {
	encryptedPath := filePath + ".gpg"
	cmd := exec.Command("gpg", "--batch", "--yes", "--passphrase", passphrase, "--symmetric", "--output", encryptedPath, filePath) // #nosec G204
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gpg encryption failed: %w, output: %s", err, string(output))
	}
	return encryptedPath, nil
}

// Decrypt decrypts a GPG-encrypted file using a symmetric passphrase.
func (e *Engine) Decrypt(filePath string, passphrase string) (string, error) {
	decryptedPath := filePath[:len(filePath)-4] // remove .gpg
	cmd := exec.Command("gpg", "--batch", "--yes", "--passphrase", passphrase, "--decrypt", "--output", decryptedPath, filePath) // #nosec G204
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gpg decryption failed: %w, output: %s", err, string(output))
	}
	return decryptedPath, nil
}

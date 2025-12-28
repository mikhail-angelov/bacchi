// Package main is the entry point for the backup-service CLI.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	"time"

	"github.com/mikhail-angelov/backup-service/internal/backup"
	"github.com/mikhail-angelov/backup-service/internal/config"
	"github.com/mikhail-angelov/backup-service/internal/retention"
	"github.com/mikhail-angelov/backup-service/internal/s3"
	"github.com/mikhail-angelov/backup-service/internal/telegram"
	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	var rootCmd = &cobra.Command{Use: "backup-service"}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file (default is config.yaml)")

	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(restoreCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func backupCmd() *cobra.Command {
	var full bool
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Perform an immediate backup and rotate old ones",
		Run: func(_ *cobra.Command, _ []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			log.Println("Starting backup process...")
			if err := executeBackup(cfg, full); err != nil {
				log.Fatalf("Backup failed: %v", err)
			}
			log.Println("Backup process completed successfully")
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "Force a full backup")
	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List backups in S3",
		Run: func(_ *cobra.Command, _ []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			ctx := context.Background()
			s3Client, err := s3.NewClient(ctx, cfg.S3.Bucket, cfg.S3.Region, cfg.S3.Endpoint, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, cfg.S3.Prefix)
			if err != nil {
				log.Fatalf("failed to create S3 client: %v", err)
			}

			backups, err := s3Client.ListBackups(ctx)
			if err != nil {
				log.Fatalf("failed to list backups: %v", err)
			}

			fmt.Println("Available backups:")
			for _, b := range backups {
				fmt.Println(b)
			}
		},
	}
}

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore [backup-key] [target-dir]",
		Short: "Restore a backup from S3 (applies Full + all Incrementals up to the key)",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			key := args[0]
			targetDir := args[1]

			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			ctx := context.Background()
			s3Client, err := s3.NewClient(ctx, cfg.S3.Bucket, cfg.S3.Region, cfg.S3.Endpoint, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, cfg.S3.Prefix)
			if err != nil {
				log.Fatalf("failed to create S3 client: %v", err)
			}

			allBackups, err := s3Client.ListBackups(ctx)
			if err != nil {
				log.Fatalf("failed to list backups: %v", err)
			}

			name, targetTs := getBackupNameAndTimestamp(key)
			if name == "" || targetTs == "" {
				log.Fatalf("failed to parse backup key: %s", key)
			}

			var chain []string
			var lastFull string
			for _, b := range allBackups {
				bName, bTs := getBackupNameAndTimestamp(b)
				if bName != name || bTs > targetTs {
					continue
				}
				if strings.Contains(b, ".full.") {
					lastFull = b
					chain = []string{b} // Start new chain from this full backup
				} else if lastFull != "" {
					chain = append(chain, b)
				}
			}

			if len(chain) == 0 {
				log.Fatalf("could not find a valid backup chain for %s", key)
			}

			// Ensure the chain ends at target key (handles cases where later backups exist)
			finalChain := []string{}
			for _, b := range chain {
				finalChain = append(finalChain, b)
				if b == key {
					break
				}
			}
			chain = finalChain

			log.Printf("Found backup chain of %d files to restore", len(chain))
			engine := backup.NewEngine(os.TempDir())

			for i, chainKey := range chain {
				log.Printf("[%d/%d] Restoring %s...", i+1, len(chain), chainKey)
				tempPath := filepath.Join(os.TempDir(), filepath.Base(chainKey))
				if err := s3Client.DownloadFile(ctx, chainKey, tempPath); err != nil {
					log.Fatalf("failed to download %s: %v", chainKey, err)
				}

				extractPath := tempPath
				if filepath.Ext(tempPath) == ".gpg" {
					decryptedPath, err := engine.Decrypt(tempPath, cfg.Encryption.Passphrase)
					if err != nil {
						log.Fatalf("failed to decrypt %s: %v", tempPath, err)
					}
					_ = os.Remove(tempPath)
					extractPath = decryptedPath
				}

				untarCmd := exec.Command("tar", "-xzf", extractPath, "-C", targetDir) // #nosec G204
				if output, err := untarCmd.CombinedOutput(); err != nil {
					log.Fatalf("failed to extract: %v, output: %s", err, string(output))
				}
				_ = os.Remove(extractPath)
			}

			log.Println("Restore completed successfully")
		},
	}
}

func getBackupNameAndTimestamp(key string) (string, string) {
	base := filepath.Base(key)
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return "", ""
	}
	name := parts[0]
	rest := parts[1]
	tsParts := strings.Split(rest, ".")
	timestamp := tsParts[0]
	return name, timestamp
}

func executeBackup(cfg *config.Config, forceFull bool) error {
	ctx := context.Background()
	engine := backup.NewEngine(os.TempDir())
	s3Client, err := s3.NewClient(ctx, cfg.S3.Bucket, cfg.S3.Region, cfg.S3.Endpoint, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, cfg.S3.Prefix)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}

	existingBackups, err := s3Client.ListBackups(ctx)
	if err != nil {
		log.Printf("Warning: failed to list existing backups, will assume no full backup exists: %v", err)
	}

	retentionManager := retention.NewManager(s3Client, cfg.Retention.Daily, cfg.Retention.Monthly)
	tgClient := telegram.NewClient(cfg.Telegram.BotToken, cfg.Telegram.ChatID)

	var errs []error
	for _, b := range cfg.Backups {
		isFull := forceFull
		if !isFull {
			currentMonth := time.Now().Format("200601")
			foundFullThisMonth := false
			for _, key := range existingBackups {
				if strings.Contains(key, "/"+b.Name+"_") && strings.Contains(key, ".full.") && strings.Contains(key, "_"+currentMonth) {
					foundFullThisMonth = true
					break
				}
			}
			if !foundFullThisMonth {
				log.Printf("No full backup found for %s in %s, forcing full backup", b.Name, currentMonth)
				isFull = true
			}
		}

		log.Printf("Backing up %s (%s)...", b.Name, map[bool]string{true: "FULL", false: "INCREMENTAL"}[isFull])
		snapshotFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.snar", b.Name))
		archivePath, backupType, err := engine.CreateArchive(b.Name, b.Folders, b.Exclude, snapshotFile, isFull)
		if err != nil {
			errs = append(errs, fmt.Errorf("backup %s failed: %w", b.Name, err))
			continue
		}

		uploadPath := archivePath
		if cfg.Encryption.Enabled {
			log.Printf("Encrypting %s...", archivePath)
			encryptedPath, err := engine.Encrypt(archivePath, cfg.Encryption.Passphrase)
			if err != nil {
				errs = append(errs, fmt.Errorf("encryption of %s failed: %w", b.Name, err))
				_ = os.Remove(archivePath)
				continue
			}
			_ = os.Remove(archivePath)
			uploadPath = encryptedPath
		}

		log.Printf("Uploading %s to S3...", uploadPath)
		if err := s3Client.UploadFile(ctx, uploadPath); err != nil {
			errs = append(errs, fmt.Errorf("upload %s failed: %w", b.Name, err))
			_ = os.Remove(uploadPath)
			continue
		}

		_ = os.Remove(uploadPath)
		log.Printf("Backup %s (%s) completed", b.Name, backupType)
	}

	log.Println("Running retention rotation...")
	if err := retentionManager.Rotate(ctx); err != nil {
		errs = append(errs, fmt.Errorf("retention failed: %w", err))
	}

	if cfg.Telegram.Enabled {
		if len(errs) > 0 {
			msg := "❌ Backup Failed:\n"
			for _, e := range errs {
				msg += fmt.Sprintf("- %v\n", e)
			}
			_ = tgClient.SendMessage(msg)
		} else {
			_ = tgClient.SendMessage("✅ Backup completed successfully")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("completed with errors: %v", errs)
	}

	return nil
}

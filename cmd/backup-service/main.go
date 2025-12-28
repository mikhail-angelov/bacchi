// Package main is the entry point for the backup-service CLI.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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
	return &cobra.Command{
		Use:   "backup",
		Short: "Perform an immediate backup and rotate old ones",
		Run: func(_ *cobra.Command, _ []string) {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				log.Fatalf("failed to load config: %v", err)
			}

			log.Println("Starting backup process...")
			if err := executeBackup(cfg); err != nil {
				log.Fatalf("Backup failed: %v", err)
			}
			log.Println("Backup process completed successfully")
		},
	}
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
		Short: "Restore a backup from S3",
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

			tempPath := filepath.Join(os.TempDir(), filepath.Base(key))
			log.Printf("Downloading %s to %s...", key, tempPath)
			if err := s3Client.DownloadFile(ctx, key, tempPath); err != nil {
				log.Fatalf("failed to download: %v", err)
			}

			extractPath := tempPath
			if filepath.Ext(tempPath) == ".gpg" {
				if !cfg.Encryption.Enabled {
					log.Fatal("backup is encrypted but encryption is not enabled in config")
				}
				log.Printf("Decrypting %s...", tempPath)
				engine := backup.NewEngine(os.TempDir())
				decryptedPath, err := engine.Decrypt(tempPath, cfg.Encryption.Passphrase)
				if err != nil {
					log.Fatalf("failed to decrypt: %v", err)
				}
				_ = os.Remove(tempPath)
				extractPath = decryptedPath
			}

			log.Printf("Extracting to %s...", targetDir)
			if err := os.MkdirAll(targetDir, 0o750); err != nil {
				log.Fatalf("failed to create target dir: %v", err)
			}

			untarCmd := exec.Command("tar", "-xzf", extractPath, "-C", targetDir) // #nosec G204
			if output, err := untarCmd.CombinedOutput(); err != nil {
				log.Fatalf("failed to extract: %v, output: %s", err, string(output))
			}
			_ = os.Remove(extractPath)

			log.Println("Restore completed successfully")
		},
	}
}

func executeBackup(cfg *config.Config) error {
	ctx := context.Background()
	engine := backup.NewEngine(os.TempDir())
	s3Client, err := s3.NewClient(ctx, cfg.S3.Bucket, cfg.S3.Region, cfg.S3.Endpoint, cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, cfg.S3.Prefix)
	if err != nil {
		return fmt.Errorf("failed to create S3 client: %w", err)
	}
	retentionManager := retention.NewManager(s3Client, cfg.Retention.Daily, cfg.Retention.Monthly)
	tgClient := telegram.NewClient(cfg.Telegram.BotToken, cfg.Telegram.ChatID)

	var errs []error
	for _, b := range cfg.Backups {
		log.Printf("Backing up %s...", b.Name)
		snapshotFile := filepath.Join(os.TempDir(), fmt.Sprintf("%s.snar", b.Name))
		archivePath, err := engine.CreateArchive(b.Name, b.Folders, b.Exclude, snapshotFile)
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

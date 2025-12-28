package retention

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mikhail-angelov/backup-service/internal/s3"
)

// Manager handles the retention and rotation of backups.
type Manager struct {
	s3Client *s3.Client
	daily    int
	monthly  int
}

// NewManager creates a new retention manager.
func NewManager(s3Client *s3.Client, daily, monthly int) *Manager {
	return &Manager{
		s3Client: s3Client,
		daily:    daily,
		monthly:  monthly,
	}
}

// Rotate performs backup rotation based on the configured daily and monthly retention policies.
func (m *Manager) Rotate(ctx context.Context) error {
	keys, err := m.s3Client.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups for rotation: %w", err)
	}

	// Filter and sort backups by date (assuming filename contains timestamp)
	// Example filename: backup_20251228075027.tar.gz
	backups := make([]string, 0)
	for _, key := range keys {
		if strings.HasSuffix(key, ".tar.gz") {
			backups = append(backups, key)
		}
	}
	sort.Strings(backups)

	if len(backups) == 0 {
		return nil
	}

	// Logic for 10 daily / 1 monthly
	// Simple approach:
	// 1. Identify monthly candidates (first backup of each month)
	// 2. Identify daily candidates (last 10 backups)
	// 3. Keep those, delete others

	toKeep := make(map[string]bool)

	// Monthly: keep the latest one (user specified 1 monthly)
	// We can find the latest monthly by grouping by YYYYMM
	months := make(map[string]string)
	for _, b := range backups {
		// Extract date: backup_20251228075027.tar.gz
		parts := strings.Split(b, "_")
		if len(parts) < 2 {
			continue
		}
		datePart := parts[len(parts)-1]
		if len(datePart) < 6 {
			continue
		}
		monthKey := datePart[:6] // YYYYMM
		months[monthKey] = b
	}

	// Keep only the latest month's first/last backup?
	// The user said "1 monthly". I'll keep the latest one from the previous month if available,
	// or just the latest one that qualifies as monthly.
	// Actually, let's just keep the last backup of each month, and then limit to 1.
	monthKeys := make([]string, 0, len(months))
	for k := range months {
		monthKeys = append(monthKeys, k)
	}
	sort.Strings(monthKeys)
	if len(monthKeys) > 0 {
		// Keep the latest monthly
		toKeep[months[monthKeys[len(monthKeys)-1]]] = true
	}

	// Daily: keep last 10
	start := len(backups) - m.daily
	if start < 0 {
		start = 0
	}
	for i := start; i < len(backups); i++ {
		toKeep[backups[i]] = true
	}

	// Delete others
	for _, b := range backups {
		if !toKeep[b] {
			fmt.Printf("Rotating out old backup: %s\n", b)
			if err := m.s3Client.DeleteFile(ctx, b); err != nil {
				return fmt.Errorf("failed to delete old backup %s: %w", b, err)
			}
		}
	}

	return nil
}

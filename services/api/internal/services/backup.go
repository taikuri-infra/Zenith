package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// BackupService orchestrates real database backups via K8s Jobs and S3 storage.
type BackupService struct {
	k8sClient  k8sclient.Client
	backupRepo ports.BackupRepository
	dbRepo     ports.DatabaseRepository
	dbSvc      *DatabaseService
	s3         ports.ObjectStorage
	s3Bucket   string
	namespace  string // K8s namespace for backup jobs
}

// NewBackupService creates a new BackupService.
func NewBackupService(
	k8sClient k8sclient.Client,
	backupRepo ports.BackupRepository,
	dbRepo ports.DatabaseRepository,
	dbSvc *DatabaseService,
	s3 ports.ObjectStorage,
	s3Bucket string,
	namespace string,
) *BackupService {
	return &BackupService{
		k8sClient:  k8sClient,
		backupRepo: backupRepo,
		dbRepo:     dbRepo,
		dbSvc:      dbSvc,
		s3:         s3,
		s3Bucket:   s3Bucket,
		namespace:  namespace,
	}
}

// BackupStorageKey returns the S3 key for a backup file.
func BackupStorageKey(databaseID, backupID string) string {
	return fmt.Sprintf("backups/%s/%s.sql.gz", databaseID, backupID)
}

// TriggerBackup creates a K8s Job to run pg_dump and upload to S3.
func (s *BackupService) TriggerBackup(ctx context.Context, backup *entities.DatabaseBackup, db *entities.UserDatabase) error {
	storageKey := BackupStorageKey(db.ID, backup.ID)

	// Get database password from K8s secret
	password := ""
	if s.dbSvc != nil {
		pw, err := s.dbSvc.GetDatabasePassword(ctx, db.ID)
		if err != nil {
			slog.Error("backup: failed to get db password", "db_id", db.ID, "error", err)
			s.backupRepo.UpdateBackupStatus(ctx, backup.ID, entities.BackupStatusFailed, 0, "failed to get database credentials")
			return fmt.Errorf("failed to get database credentials: %w", err)
		}
		password = pw
	}

	if db.Host == "" || password == "" {
		s.backupRepo.UpdateBackupStatus(ctx, backup.ID, entities.BackupStatusFailed, 0, "database not provisioned or missing credentials")
		return fmt.Errorf("database not provisioned or missing credentials")
	}

	jobName := fmt.Sprintf("backup-%s", backup.ID[:8])

	// Build the K8s Job manifest for pg_dump → gzip → S3 upload
	job := &k8sclient.JobObject{
		Name:      jobName,
		Namespace: s.namespace,
		Labels: map[string]string{
			"zenith.dev/backup-id":   backup.ID,
			"zenith.dev/database-id": db.ID,
			"zenith.dev/managed-by":  "zenith",
			"zenith.dev/job-type":    "backup",
		},
		Spec: s.buildBackupJobSpec(db, password, storageKey, jobName),
	}

	// Update status to running
	if err := s.backupRepo.UpdateBackupStatus(ctx, backup.ID, entities.BackupStatusRunning, 0, ""); err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	// Create the K8s Job
	if err := s.k8sClient.CreateJob(ctx, job); err != nil {
		s.backupRepo.UpdateBackupStatus(ctx, backup.ID, entities.BackupStatusFailed, 0, fmt.Sprintf("failed to create backup job: %v", err))
		return fmt.Errorf("failed to create backup job: %w", err)
	}

	// Watch job completion in background
	go s.watchBackupJob(backup.ID, db.ID, jobName, storageKey)

	return nil
}

// TriggerRestore creates a K8s Job to download from S3 and run pg_restore.
func (s *BackupService) TriggerRestore(ctx context.Context, backup *entities.DatabaseBackup, db *entities.UserDatabase) error {
	storageKey := backup.StorageKey
	if storageKey == "" {
		storageKey = BackupStorageKey(db.ID, backup.ID)
	}

	password := ""
	if s.dbSvc != nil {
		pw, err := s.dbSvc.GetDatabasePassword(ctx, db.ID)
		if err != nil {
			return fmt.Errorf("failed to get database credentials: %w", err)
		}
		password = pw
	}

	if db.Host == "" || password == "" {
		return fmt.Errorf("database not provisioned or missing credentials")
	}

	jobName := fmt.Sprintf("restore-%s", backup.ID[:8])

	job := &k8sclient.JobObject{
		Name:      jobName,
		Namespace: s.namespace,
		Labels: map[string]string{
			"zenith.dev/backup-id":   backup.ID,
			"zenith.dev/database-id": db.ID,
			"zenith.dev/managed-by":  "zenith",
			"zenith.dev/job-type":    "restore",
		},
		Spec: s.buildRestoreJobSpec(db, password, storageKey, jobName),
	}

	// Mark database as provisioning during restore
	s.dbRepo.UpdateDatabaseStatus(ctx, db.ID, entities.DatabaseStatusProvisioning)

	if err := s.k8sClient.CreateJob(ctx, job); err != nil {
		s.dbRepo.UpdateDatabaseStatus(ctx, db.ID, entities.DatabaseStatusReady)
		return fmt.Errorf("failed to create restore job: %w", err)
	}

	// Watch job completion in background
	go s.watchRestoreJob(backup.ID, db.ID, jobName)

	return nil
}

// GenerateDownloadURL creates a presigned S3 URL for downloading a backup.
func (s *BackupService) GenerateDownloadURL(ctx context.Context, backup *entities.DatabaseBackup) (string, error) {
	storageKey := backup.StorageKey
	if storageKey == "" {
		storageKey = BackupStorageKey(backup.DatabaseID, backup.ID)
	}

	url, err := s.s3.GeneratePresignedDownloadURL(ctx, s.s3Bucket, storageKey, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}
	return url, nil
}

// buildBackupJobSpec creates the K8s Job spec for pg_dump → S3.
func (s *BackupService) buildBackupJobSpec(db *entities.UserDatabase, password, storageKey, jobName string) map[string]interface{} {
	// Shell script: pg_dump | gzip | aws s3 cp
	script := fmt.Sprintf(`set -e
echo "Starting backup: %s on %s"
export PGPASSWORD="%s"
pg_dump -h "%s" -p %d -U "%s" -d "%s" --no-owner --no-privileges | gzip | \
  aws s3 cp - "s3://%s/%s" \
  --endpoint-url "$AWS_ENDPOINT_URL" \
  --region fsn1
SIZE=$(aws s3api head-object --bucket "%s" --key "%s" --endpoint-url "$AWS_ENDPOINT_URL" --region fsn1 --query ContentLength --output text 2>/dev/null || echo "0")
echo "BACKUP_SIZE_BYTES=$SIZE"
echo "Backup complete"`,
		db.DBName, db.Host,
		password,
		db.Host, db.Port, db.DBUser, db.DBName,
		s.s3Bucket, storageKey,
		s.s3Bucket, storageKey,
	)

	return map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      jobName,
			"namespace": s.namespace,
		},
		"spec": map[string]interface{}{
			"ttlSecondsAfterFinished": 3600,
			"backoffLimit":            2,
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"restartPolicy": "Never",
					"containers": []map[string]interface{}{
						{
							"name":  "pg-backup",
							"image": "ghcr.io/cloudnative-pg/postgresql:16.6",
							"command": []string{
								"/bin/sh", "-c",
								// Install aws-cli, then run backup
								fmt.Sprintf(`apk add --no-cache aws-cli >/dev/null 2>&1 && %s`, script),
							},
							"env": []map[string]interface{}{
								{"name": "AWS_ACCESS_KEY_ID", "valueFrom": map[string]interface{}{
									"secretKeyRef": map[string]interface{}{
										"name": "cnpg-s3-credentials",
										"key":  "ACCESS_KEY_ID",
									},
								}},
								{"name": "AWS_SECRET_ACCESS_KEY", "valueFrom": map[string]interface{}{
									"secretKeyRef": map[string]interface{}{
										"name": "cnpg-s3-credentials",
										"key":  "ACCESS_SECRET_KEY",
									},
								}},
								{"name": "AWS_ENDPOINT_URL", "value": "https://fsn1.your-objectstorage.com"},
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "128Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "500m",
									"memory": "512Mi",
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildRestoreJobSpec creates the K8s Job spec for S3 → pg_restore.
func (s *BackupService) buildRestoreJobSpec(db *entities.UserDatabase, password, storageKey, jobName string) map[string]interface{} {
	script := fmt.Sprintf(`set -e
echo "Starting restore: %s on %s"
export PGPASSWORD="%s"
aws s3 cp "s3://%s/%s" - \
  --endpoint-url "$AWS_ENDPOINT_URL" \
  --region fsn1 | gunzip | \
  psql -h "%s" -p %d -U "%s" -d "%s" --no-password
echo "Restore complete"`,
		db.DBName, db.Host,
		password,
		s.s3Bucket, storageKey,
		db.Host, db.Port, db.DBUser, db.DBName,
	)

	return map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      jobName,
			"namespace": s.namespace,
		},
		"spec": map[string]interface{}{
			"ttlSecondsAfterFinished": 3600,
			"backoffLimit":            1,
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"restartPolicy": "Never",
					"containers": []map[string]interface{}{
						{
							"name":  "pg-restore",
							"image": "ghcr.io/cloudnative-pg/postgresql:16.6",
							"command": []string{
								"/bin/sh", "-c",
								fmt.Sprintf(`apk add --no-cache aws-cli >/dev/null 2>&1 && %s`, script),
							},
							"env": []map[string]interface{}{
								{"name": "AWS_ACCESS_KEY_ID", "valueFrom": map[string]interface{}{
									"secretKeyRef": map[string]interface{}{
										"name": "cnpg-s3-credentials",
										"key":  "ACCESS_KEY_ID",
									},
								}},
								{"name": "AWS_SECRET_ACCESS_KEY", "valueFrom": map[string]interface{}{
									"secretKeyRef": map[string]interface{}{
										"name": "cnpg-s3-credentials",
										"key":  "ACCESS_SECRET_KEY",
									},
								}},
								{"name": "AWS_ENDPOINT_URL", "value": "https://fsn1.your-objectstorage.com"},
							},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    "100m",
									"memory": "128Mi",
								},
								"limits": map[string]interface{}{
									"cpu":    "500m",
									"memory": "512Mi",
								},
							},
						},
					},
				},
			},
		},
	}
}

// watchBackupJob polls the K8s Job until completion or failure.
func (s *BackupService) watchBackupJob(backupID, dbID, jobName, storageKey string) {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-timeout:
			slog.Error("backup job timed out", "backup_id", backupID, "job", jobName)
			s.backupRepo.UpdateBackupStatus(ctx, backupID, entities.BackupStatusFailed, 0, "backup job timed out after 10 minutes")
			s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
			return

		case <-ticker.C:
			job, err := s.k8sClient.GetJob(ctx, s.namespace, jobName)
			if err != nil {
				continue // Job may not be visible yet
			}

			if job.Succeeded > 0 {
				// Backup succeeded — update status with storage key and estimated size
				sizeMB := 1 // minimum
				// Try to get actual size from S3 metadata
				result, listErr := s.s3.ListObjects(ctx, s.s3Bucket, storageKey, "", 1)
				if listErr == nil && len(result.Objects) > 0 {
					sizeMB = int(result.Objects[0].Size / (1024 * 1024))
					if sizeMB < 1 {
						sizeMB = 1
					}
				}

				// Update storage_key in the backup record
				s.backupRepo.UpdateBackupStatus(ctx, backupID, entities.BackupStatusCompleted, sizeMB, "")
				// Also set storage_key
				s.setBackupStorageKey(ctx, backupID, storageKey)

				slog.Info("backup completed", "backup_id", backupID, "size_mb", sizeMB)
				s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
				return
			}

			if job.Failed > 0 {
				slog.Error("backup job failed", "backup_id", backupID, "job", jobName)
				s.backupRepo.UpdateBackupStatus(ctx, backupID, entities.BackupStatusFailed, 0, "backup job failed")
				s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
				return
			}
		}
	}
}

// watchRestoreJob polls the K8s Job until completion or failure.
func (s *BackupService) watchRestoreJob(backupID, dbID, jobName string) {
	ctx := context.Background()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(15 * time.Minute)

	for {
		select {
		case <-timeout:
			slog.Error("restore job timed out", "backup_id", backupID, "job", jobName)
			s.dbRepo.UpdateDatabaseStatus(ctx, dbID, entities.DatabaseStatusReady)
			s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
			return

		case <-ticker.C:
			job, err := s.k8sClient.GetJob(ctx, s.namespace, jobName)
			if err != nil {
				continue
			}

			if job.Succeeded > 0 {
				slog.Info("restore completed", "backup_id", backupID, "db_id", dbID)
				s.dbRepo.UpdateDatabaseStatus(ctx, dbID, entities.DatabaseStatusReady)
				s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
				return
			}

			if job.Failed > 0 {
				slog.Error("restore job failed", "backup_id", backupID, "job", jobName)
				s.dbRepo.UpdateDatabaseStatus(ctx, dbID, entities.DatabaseStatusReady)
				s.k8sClient.DeleteJob(ctx, s.namespace, jobName)
				return
			}
		}
	}
}

// setBackupStorageKey updates the storage_key field on a backup record.
func (s *BackupService) setBackupStorageKey(ctx context.Context, backupID, storageKey string) {
	// Direct SQL update since the repository interface doesn't have a dedicated method
	type storageKeySetter interface {
		SetStorageKey(ctx context.Context, id, key string) error
	}
	if setter, ok := s.backupRepo.(storageKeySetter); ok {
		if err := setter.SetStorageKey(ctx, backupID, storageKey); err != nil {
			slog.Error("failed to set backup storage key", "backup_id", backupID, "error", err)
		}
	}
}

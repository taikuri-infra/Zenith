package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

const databaseFinalizer = "zenith.dev/database-cleanup"

type DatabaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Hetzner  *hetzner.Client
}

func NewDatabaseReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder, h *hetzner.Client) *DatabaseReconciler {
	return &DatabaseReconciler{Client: c, Scheme: s, Recorder: r, Hetzner: h}
}

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var db zenithv1.Database
	if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !db.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&db, databaseFinalizer) {
			logger.Info("Cleaning up database resources", "name", db.Name)

			// Delete Hetzner volume if it exists
			if db.Status.HetznerVolumeID != "" && r.Hetzner.IsConfigured() {
				volID, err := hetzner.ParseVolumeID(db.Status.HetznerVolumeID)
				if err == nil {
					if err := r.Hetzner.DeleteVolume(ctx, volID); err != nil {
						logger.Error(err, "Failed to delete Hetzner volume", "volumeID", db.Status.HetznerVolumeID)
						return ctrl.Result{}, err
					}
					r.Recorder.Event(&db, corev1.EventTypeNormal, "VolumeDeleted", "Deleted Hetzner volume")
				}
			}

			controllerutil.RemoveFinalizer(&db, databaseFinalizer)
			if err := r.Update(ctx, &db); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&db, databaseFinalizer) {
		controllerutil.AddFinalizer(&db, databaseFinalizer)
		if err := r.Update(ctx, &db); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Set initial status
	if db.Status.Phase == "" {
		db.Status.Phase = "Provisioning"
		if err := r.Status().Update(ctx, &db); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&db, corev1.EventTypeNormal, "Provisioning", "Starting database provisioning")
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       db.Name,
		"app.kubernetes.io/component":  "database",
		"app.kubernetes.io/managed-by": "zenith-operator",
		"zenith.dev/database":          db.Name,
		"zenith.dev/engine":            db.Spec.Engine,
	}

	// Determine engine-specific settings
	port, image, dataDir, envVars := engineConfig(db.Spec.Engine, db.Spec.Version)

	// Step 1: Create Hetzner Volume if configured and not yet created
	if db.Status.HetznerVolumeID == "" && r.Hetzner.IsConfigured() {
		sizeGB := int(db.Spec.Storage.Value() / (1024 * 1024 * 1024))
		if sizeGB < 10 {
			sizeGB = 10
		}

		vol, err := r.Hetzner.CreateVolume(ctx, fmt.Sprintf("zenith-db-%s-%s", db.Namespace, db.Name), sizeGB, "fsn1")
		if err != nil {
			r.Recorder.Eventf(&db, corev1.EventTypeWarning, "VolumeCreationFailed", "Failed to create volume: %v", err)
			db.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &db)
			return ctrl.Result{}, err
		}

		db.Status.HetznerVolumeID = fmt.Sprintf("%d", vol.ID)
		r.Recorder.Eventf(&db, corev1.EventTypeNormal, "VolumeCreated", "Created Hetzner volume %d", vol.ID)

		if err := r.Status().Update(ctx, &db); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 2: CreateOrUpdate PersistentVolume and PersistentVolumeClaim
	// If Hetzner volume exists, create PV using Hetzner CSI driver
	pvcName := fmt.Sprintf("%s-data", db.Name)

	if db.Status.HetznerVolumeID != "" {
		pvName := fmt.Sprintf("zenith-db-%s-%s", db.Namespace, db.Name)
		storageSize := db.Spec.Storage.DeepCopy()

		pv := &corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
			},
		}
		pvResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, pv, func() error {
			pv.Labels = labels
			fsType := "ext4"
			pv.Spec = corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: storageSize,
				},
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					CSI: &corev1.CSIPersistentVolumeSource{
						Driver:       "csi.hetzner.cloud",
						VolumeHandle: db.Status.HetznerVolumeID,
						FSType:       fsType,
					},
				},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				StorageClassName:              "hcloud-volumes",
			}
			return nil
		})
		if err != nil {
			r.Recorder.Eventf(&db, corev1.EventTypeWarning, "PVFailed", "Failed to ensure PersistentVolume: %v", err)
			return ctrl.Result{}, err
		}
		if pvResult != controllerutil.OperationResultNone {
			logger.Info("PersistentVolume reconciled", "operation", pvResult)
		}

		// CreateOrUpdate PVC
		storageClassName := "hcloud-volumes"
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: db.Namespace,
			},
		}
		pvcResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
			pvc.Labels = labels
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageSize,
					},
				},
				VolumeName:       pvName,
				StorageClassName: &storageClassName,
			}
			return ctrl.SetControllerReference(&db, pvc, r.Scheme)
		})
		if err != nil {
			r.Recorder.Eventf(&db, corev1.EventTypeWarning, "PVCFailed", "Failed to ensure PersistentVolumeClaim: %v", err)
			return ctrl.Result{}, err
		}
		if pvcResult != controllerutil.OperationResultNone {
			logger.Info("PersistentVolumeClaim reconciled", "operation", pvcResult)
		}
	} else {
		// No Hetzner volume: create a PVC with default storage class
		storageSize := db.Spec.Storage.DeepCopy()
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: db.Namespace,
			},
		}
		pvcResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
			pvc.Labels = labels
			pvc.Spec = corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: storageSize,
					},
				},
			}
			return ctrl.SetControllerReference(&db, pvc, r.Scheme)
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		if pvcResult != controllerutil.OperationResultNone {
			logger.Info("PersistentVolumeClaim reconciled (default SC)", "operation", pvcResult)
		}
	}

	// Step 3: CreateOrUpdate Secret with credentials
	// Only generate password on first creation
	secretName := fmt.Sprintf("%s-conn", db.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: db.Namespace,
		},
	}
	secretResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels

		// Preserve existing password if secret already exists
		existingPassword := ""
		if secret.Data != nil {
			if pw, ok := secret.Data["password"]; ok {
				existingPassword = string(pw)
			}
		}

		password := existingPassword
		if password == "" {
			password = generatePassword(24)
		}

		username := dbUsername(db.Spec.Engine)
		host := fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
		connString := buildConnectionString(db.Spec.Engine, username, password, host, port, db.Name)

		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			"username":          []byte(username),
			"password":          []byte(password),
			"host":              []byte(host),
			"port":              []byte(strconv.Itoa(int(port))),
			"database":          []byte(db.Name),
			"connection-string": []byte(connString),
		}
		return ctrl.SetControllerReference(&db, secret, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&db, corev1.EventTypeWarning, "SecretFailed", "Failed to ensure connection secret: %v", err)
		return ctrl.Result{}, err
	}
	if secretResult != controllerutil.OperationResultNone {
		logger.Info("Connection secret reconciled", "operation", secretResult)
	}

	// Read back the password from the secret for env var injection
	if err := r.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		return ctrl.Result{}, err
	}
	password := string(secret.Data["password"])

	// Step 4: CreateOrUpdate StatefulSet
	replicas := db.Spec.Replicas
	if replicas < 1 {
		replicas = 1
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
	}
	stsResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
		sts.Labels = labels

		// Build env vars for the container
		containerEnv := buildDBEnvVars(db.Spec.Engine, password, envVars)

		// Resource requirements
		resources := corev1.ResourceRequirements{}
		if db.Spec.Resources != nil {
			resources = corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    db.Spec.Resources.CPU,
					corev1.ResourceMemory: db.Spec.Resources.Memory,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewMilliQuantity(db.Spec.Resources.CPU.MilliValue()/2, resource.DecimalSI),
					corev1.ResourceMemory: *resource.NewQuantity(db.Spec.Resources.Memory.Value()/2, resource.BinarySI),
				},
			}
		}

		sts.Spec = appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: db.Name,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:      db.Spec.Engine,
							Image:     image,
							Ports:     []corev1.ContainerPort{{ContainerPort: port, Protocol: corev1.ProtocolTCP}},
							Env:       containerEnv,
							Resources: resources,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: dataDir,
								},
							},
							ReadinessProbe: buildDBReadinessProbe(db.Spec.Engine, port),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(&db, sts, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&db, corev1.EventTypeWarning, "StatefulSetFailed", "Failed to ensure StatefulSet: %v", err)
		return ctrl.Result{}, err
	}
	if stsResult != controllerutil.OperationResultNone {
		logger.Info("StatefulSet reconciled", "operation", stsResult)
		r.Recorder.Eventf(&db, corev1.EventTypeNormal, "StatefulSetReady", "StatefulSet %s", stsResult)
	}

	// Step 5: CreateOrUpdate headless Service for StatefulSet
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
	}
	svcResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Selector:  labels,
			ClusterIP: corev1.ClusterIPNone, // headless
			Ports: []corev1.ServicePort{
				{
					Name:       db.Spec.Engine,
					Port:       port,
					TargetPort: intstr.FromInt32(port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		return ctrl.SetControllerReference(&db, svc, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&db, corev1.EventTypeWarning, "ServiceFailed", "Failed to ensure headless Service: %v", err)
		return ctrl.Result{}, err
	}
	if svcResult != controllerutil.OperationResultNone {
		logger.Info("Headless Service reconciled", "operation", svcResult)
	}

	// Step 6: If backup enabled, CreateOrUpdate CronJob
	if db.Spec.Backup != nil && db.Spec.Backup.Enabled {
		if err := r.reconcileBackupCronJob(ctx, &db, labels, port, secretName); err != nil {
			r.Recorder.Eventf(&db, corev1.EventTypeWarning, "BackupCronJobFailed", "Failed to ensure backup CronJob: %v", err)
			return ctrl.Result{}, err
		}
	}

	// Update status
	host := fmt.Sprintf("%s.%s.svc.cluster.local", db.Name, db.Namespace)
	connString := buildConnectionString(db.Spec.Engine, dbUsername(db.Spec.Engine), string(secret.Data["password"]), host, port, db.Name)

	db.Status.Phase = "Ready"
	db.Status.Host = host
	db.Status.Port = port
	db.Status.SecretName = secretName
	db.Status.ConnectionString = connString

	if err := r.Status().Update(ctx, &db); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&db, corev1.EventTypeNormal, "Ready", "Database is ready")
	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) reconcileBackupCronJob(ctx context.Context, db *zenithv1.Database, labels map[string]string, port int32, secretName string) error {
	logger := log.FromContext(ctx)

	schedule := "0 2 * * *"
	if db.Spec.Backup.Schedule != "" {
		schedule = db.Spec.Backup.Schedule
	}

	backupCmd := backupCommand(db.Spec.Engine, db.Name, port)
	_, image, _, _ := engineConfig(db.Spec.Engine, db.Spec.Version)

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-backup", db.Name),
			Namespace: db.Namespace,
		},
	}

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, cronJob, func() error {
		cronJob.Labels = labels
		retentionDays := int32(db.Spec.Backup.RetentionDays)
		if retentionDays <= 0 {
			retentionDays = 7
		}
		successLimit := int32(3)
		failedLimit := int32(1)

		cronJob.Spec = batchv1.CronJobSpec{
			Schedule:                   schedule,
			SuccessfulJobsHistoryLimit: &successLimit,
			FailedJobsHistoryLimit:     &failedLimit,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Containers: []corev1.Container{
								{
									Name:    "backup",
									Image:   image,
									Command: []string{"/bin/sh", "-c", backupCmd},
									Env: []corev1.EnvVar{
										{
											Name: "DB_PASSWORD",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  "password",
												},
											},
										},
										{
											Name: "DB_HOST",
											ValueFrom: &corev1.EnvVarSource{
												SecretKeyRef: &corev1.SecretKeySelector{
													LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
													Key:                  "host",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(db, cronJob, r.Scheme)
	})
	if err != nil {
		return err
	}
	if result != controllerutil.OperationResultNone {
		logger.Info("Backup CronJob reconciled", "operation", result)
	}
	return nil
}

// engineConfig returns port, image, data directory, and env var names for each engine.
func engineConfig(engine, version string) (port int32, image, dataDir string, envVars map[string]string) {
	switch engine {
	case "postgresql":
		return 5432, fmt.Sprintf("postgres:%s", version), "/var/lib/postgresql/data", map[string]string{
			"passwordEnv": "POSTGRES_PASSWORD",
			"userEnv":     "POSTGRES_USER",
			"dbEnv":       "POSTGRES_DB",
		}
	case "mysql":
		return 3306, fmt.Sprintf("mysql:%s", version), "/var/lib/mysql", map[string]string{
			"passwordEnv": "MYSQL_ROOT_PASSWORD",
			"dbEnv":       "MYSQL_DATABASE",
		}
	case "mongodb":
		return 27017, fmt.Sprintf("mongo:%s", version), "/data/db", map[string]string{
			"passwordEnv": "MONGO_INITDB_ROOT_PASSWORD",
			"userEnv":     "MONGO_INITDB_ROOT_USERNAME",
		}
	case "redis":
		return 6379, fmt.Sprintf("redis:%s", version), "/data", map[string]string{}
	default:
		return 5432, fmt.Sprintf("postgres:%s", version), "/var/lib/postgresql/data", map[string]string{
			"passwordEnv": "POSTGRES_PASSWORD",
		}
	}
}

func dbUsername(engine string) string {
	switch engine {
	case "postgresql":
		return "postgres"
	case "mysql":
		return "root"
	case "mongodb":
		return "admin"
	case "redis":
		return ""
	default:
		return "admin"
	}
}

func buildConnectionString(engine, username, password, host string, port int32, dbName string) string {
	switch engine {
	case "postgresql":
		return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", username, password, host, port, dbName)
	case "mysql":
		return fmt.Sprintf("mysql://%s:%s@%s:%d/%s", username, password, host, port, dbName)
	case "mongodb":
		return fmt.Sprintf("mongodb://%s:%s@%s:%d/%s", username, password, host, port, dbName)
	case "redis":
		if password != "" {
			return fmt.Sprintf("redis://:%s@%s:%d", password, host, port)
		}
		return fmt.Sprintf("redis://%s:%d", host, port)
	default:
		return fmt.Sprintf("%s://%s:%d", engine, host, port)
	}
}

func buildDBEnvVars(engine, password string, envVarNames map[string]string) []corev1.EnvVar {
	var envs []corev1.EnvVar

	if pwEnv, ok := envVarNames["passwordEnv"]; ok && pwEnv != "" {
		envs = append(envs, corev1.EnvVar{Name: pwEnv, Value: password})
	}
	if userEnv, ok := envVarNames["userEnv"]; ok && userEnv != "" {
		envs = append(envs, corev1.EnvVar{Name: userEnv, Value: dbUsername(engine)})
	}
	if dbEnv, ok := envVarNames["dbEnv"]; ok && dbEnv != "" {
		envs = append(envs, corev1.EnvVar{Name: dbEnv, Value: "zenith"})
	}

	return envs
}

func buildDBReadinessProbe(engine string, port int32) *corev1.Probe {
	switch engine {
	case "postgresql":
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"pg_isready", "-U", "postgres"},
				},
			},
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			InitialDelaySeconds: 15,
		}
	case "mysql":
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"mysqladmin", "ping", "-h", "localhost"},
				},
			},
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			InitialDelaySeconds: 15,
		}
	case "mongodb":
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"mongosh", "--eval", "db.adminCommand('ping')"},
				},
			},
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			InitialDelaySeconds: 15,
		}
	case "redis":
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"redis-cli", "ping"},
				},
			},
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
			InitialDelaySeconds: 5,
		}
	default:
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt32(port),
				},
			},
			PeriodSeconds:       10,
			InitialDelaySeconds: 15,
		}
	}
}

func backupCommand(engine, dbName string, port int32) string {
	switch engine {
	case "postgresql":
		return fmt.Sprintf("PGPASSWORD=$DB_PASSWORD pg_dump -h $DB_HOST -U postgres -d %s > /tmp/backup-$(date +%%Y%%m%%d%%H%%M%%S).sql", dbName)
	case "mysql":
		return fmt.Sprintf("mysqldump -h $DB_HOST -u root -p$DB_PASSWORD %s > /tmp/backup-$(date +%%Y%%m%%d%%H%%M%%S).sql", dbName)
	case "mongodb":
		return "mongodump --host $DB_HOST --username admin --password $DB_PASSWORD --out /tmp/backup-$(date +%Y%m%d%H%M%S)"
	case "redis":
		return "redis-cli -h $DB_HOST BGSAVE && sleep 5 && cp /data/dump.rdb /tmp/backup-$(date +%Y%m%d%H%M%S).rdb"
	default:
		return "echo 'Unsupported engine for backup'"
	}
}

// generatePassword generates a cryptographically random hex password.
func generatePassword(length int) string {
	bytes := make([]byte, length/2+1)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback - should never happen
		return "changeme-fallback-password"
	}
	return hex.EncodeToString(bytes)[:length]
}

func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Database{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&batchv1.CronJob{}).
		Named("database").
		Complete(r)
}


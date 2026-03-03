package v1alpha1

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ============================================================================
// Project DeepCopy Tests
// ============================================================================

func TestProjectDeepCopy(t *testing.T) {
	proj := &Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: ProjectSpec{
			DisplayName: "Test Project",
			Owner:       "user@example.com",
			Plan:        "pro",
			Region:      "fsn1",
			ResourceQuota: &ResourceQuota{
				MaxApps:      10,
				MaxDatabases: 5,
				MaxStorageGB: 100,
			},
		},
		Status: ProjectStatus{
			Phase:     "Active",
			Namespace: "zenith-test-project",
		},
	}

	copied := proj.DeepCopy()
	if copied.Spec.DisplayName != proj.Spec.DisplayName {
		t.Errorf("DeepCopy failed: DisplayName mismatch")
	}
	if copied.Spec.ResourceQuota.MaxApps != 10 {
		t.Errorf("DeepCopy failed: ResourceQuota.MaxApps = %d, want 10", copied.Spec.ResourceQuota.MaxApps)
	}

	// Ensure independence
	copied.Spec.ResourceQuota.MaxApps = 20
	if proj.Spec.ResourceQuota.MaxApps == 20 {
		t.Error("DeepCopy is not independent: modifying copy affected original")
	}
}

func TestProjectImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &Project{}
	var _ runtime.Object = &ProjectList{}
}

func TestProjectDeepCopyInto(t *testing.T) {
	proj := &Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: ProjectSpec{
			DisplayName: "Test Project",
			Owner:       "user@example.com",
			Plan:        "pro",
			Region:      "fsn1",
			ResourceQuota: &ResourceQuota{
				MaxApps:      10,
				MaxDatabases: 5,
				MaxStorageGB: 100,
			},
		},
		Status: ProjectStatus{
			Phase:         "Active",
			Namespace:     "zenith-test-project",
			AppCount:      3,
			DatabaseCount: 2,
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	out := &Project{}
	proj.DeepCopyInto(out)

	if out.Spec.DisplayName != "Test Project" {
		t.Errorf("DeepCopyInto failed: DisplayName = %q", out.Spec.DisplayName)
	}
	if out.Spec.ResourceQuota == nil {
		t.Fatal("DeepCopyInto failed: ResourceQuota is nil")
	}
	if out.Spec.ResourceQuota.MaxApps != 10 {
		t.Errorf("DeepCopyInto failed: MaxApps = %d", out.Spec.ResourceQuota.MaxApps)
	}
	if out.Status.AppCount != 3 {
		t.Errorf("DeepCopyInto failed: AppCount = %d", out.Status.AppCount)
	}
	if len(out.Status.Conditions) != 1 {
		t.Errorf("DeepCopyInto failed: Conditions length = %d", len(out.Status.Conditions))
	}

	// Ensure pointer independence
	out.Spec.ResourceQuota.MaxApps = 99
	if proj.Spec.ResourceQuota.MaxApps == 99 {
		t.Error("DeepCopyInto is not independent for ResourceQuota pointer")
	}

	// Ensure conditions slice independence
	out.Status.Conditions[0].Status = metav1.ConditionFalse
	if proj.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Error("DeepCopyInto is not independent for Conditions slice")
	}
}

func TestProjectDeepCopyObject(t *testing.T) {
	proj := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       ProjectSpec{DisplayName: "Test"},
	}

	obj := proj.DeepCopyObject()
	copied, ok := obj.(*Project)
	if !ok {
		t.Fatal("DeepCopyObject did not return *Project")
	}
	if copied.Spec.DisplayName != "Test" {
		t.Errorf("DeepCopyObject failed: DisplayName = %q", copied.Spec.DisplayName)
	}
}

func TestProjectDeepCopyWithNilQuota(t *testing.T) {
	proj := &Project{
		ObjectMeta: metav1.ObjectMeta{Name: "no-quota"},
		Spec: ProjectSpec{
			DisplayName:   "No Quota",
			Owner:         "test@test.com",
			ResourceQuota: nil,
		},
	}

	copied := proj.DeepCopy()
	if copied.Spec.ResourceQuota != nil {
		t.Error("DeepCopy should preserve nil ResourceQuota")
	}
}

func TestProjectListDeepCopy(t *testing.T) {
	list := &ProjectList{
		Items: []Project{
			{ObjectMeta: metav1.ObjectMeta{Name: "p1"}, Spec: ProjectSpec{DisplayName: "P1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "p2"}, Spec: ProjectSpec{DisplayName: "P2"}},
		},
	}

	copied := list.DeepCopy()
	if len(copied.Items) != 2 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
	if copied.Items[0].Name != "p1" {
		t.Errorf("DeepCopy failed: Items[0].Name = %q", copied.Items[0].Name)
	}

	// Ensure independence
	copied.Items[0].Spec.DisplayName = "Modified"
	if list.Items[0].Spec.DisplayName == "Modified" {
		t.Error("DeepCopy is not independent for Items slice")
	}
}

func TestProjectListDeepCopyObject(t *testing.T) {
	list := &ProjectList{
		Items: []Project{{ObjectMeta: metav1.ObjectMeta{Name: "p1"}}},
	}
	obj := list.DeepCopyObject()
	_, ok := obj.(*ProjectList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *ProjectList")
	}
}

func TestProjectSpecDeepCopy(t *testing.T) {
	spec := &ProjectSpec{
		DisplayName: "Test",
		Owner:       "user@test.com",
		Plan:        "pro",
		Region:      "fsn1",
		ResourceQuota: &ResourceQuota{
			MaxApps:      10,
			MaxDatabases: 5,
			MaxStorageGB: 100,
		},
	}

	copied := spec.DeepCopy()
	if copied.DisplayName != "Test" {
		t.Errorf("DeepCopy failed: DisplayName = %q", copied.DisplayName)
	}
	copied.ResourceQuota.MaxApps = 99
	if spec.ResourceQuota.MaxApps == 99 {
		t.Error("DeepCopy is not independent for ResourceQuota")
	}
}

func TestProjectSpecDeepCopyNil(t *testing.T) {
	var spec *ProjectSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil ProjectSpec should return nil")
	}
}

func TestProjectStatusDeepCopy(t *testing.T) {
	status := &ProjectStatus{
		Phase:     "Active",
		Namespace: "ns",
		Conditions: []metav1.Condition{
			{Type: "Ready", Status: metav1.ConditionTrue},
		},
		AppCount:      3,
		DatabaseCount: 1,
	}

	copied := status.DeepCopy()
	if copied.Phase != "Active" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
	if len(copied.Conditions) != 1 {
		t.Fatalf("DeepCopy failed: Conditions length = %d", len(copied.Conditions))
	}
	copied.Conditions[0].Status = metav1.ConditionFalse
	if status.Conditions[0].Status != metav1.ConditionTrue {
		t.Error("DeepCopy is not independent for Conditions")
	}
}

func TestProjectStatusDeepCopyNil(t *testing.T) {
	var status *ProjectStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil ProjectStatus should return nil")
	}
}

func TestResourceQuotaDeepCopy(t *testing.T) {
	rq := &ResourceQuota{MaxApps: 10, MaxDatabases: 5, MaxStorageGB: 100}
	copied := rq.DeepCopy()
	if copied.MaxApps != 10 {
		t.Errorf("DeepCopy failed: MaxApps = %d", copied.MaxApps)
	}
	copied.MaxApps = 99
	if rq.MaxApps == 99 {
		t.Error("DeepCopy is not independent for ResourceQuota")
	}
}

func TestResourceQuotaDeepCopyNil(t *testing.T) {
	var rq *ResourceQuota
	if rq.DeepCopy() != nil {
		t.Error("DeepCopy of nil ResourceQuota should return nil")
	}
}

// ============================================================================
// App DeepCopy Tests
// ============================================================================

func TestAppDeepCopy(t *testing.T) {
	replicas := int32(3)
	app := &App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "zenith-proj",
		},
		Spec: AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
		},
		Status: AppStatus{
			Phase:         "Running",
			ReadyReplicas: 3,
			URL:           "https://my-app.example.com",
		},
	}

	copied := app.DeepCopy()
	if copied.Spec.Image != "nginx:latest" {
		t.Errorf("DeepCopy failed: Image mismatch")
	}
	if *copied.Spec.Replicas != 3 {
		t.Errorf("DeepCopy failed: Replicas = %d, want 3", *copied.Spec.Replicas)
	}

	// Ensure pointer independence
	*copied.Spec.Replicas = 5
	if *app.Spec.Replicas != 3 {
		t.Error("DeepCopy is not independent for Replicas pointer")
	}
}

func TestAppImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &App{}
	var _ runtime.Object = &AppList{}
}

func TestAppDeepCopyWithAllFields(t *testing.T) {
	replicas := int32(2)
	now := metav1.NewTime(time.Now())
	app := &App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "full-app",
			Namespace: "zenith-test",
		},
		Spec: AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
			Domain:   "app.example.com",
			Env: []corev1.EnvVar{
				{Name: "FOO", Value: "bar"},
				{Name: "BAZ", Value: "qux"},
			},
			Resources: &AppResources{
				CPU:    resource.MustParse("500m"),
				Memory: resource.MustParse("256Mi"),
			},
			HealthCheck: &HealthCheck{
				Path:            "/health",
				Port:            8081,
				IntervalSeconds: 15,
			},
			AutoScale: &AutoScale{
				MinReplicas:      2,
				MaxReplicas:      10,
				TargetCPUPercent: 80,
			},
			BuildSource: &BuildSource{
				GitURL:     "https://github.com/example/repo.git",
				Branch:     "main",
				Dockerfile: "Dockerfile",
			},
		},
		Status: AppStatus{
			Phase:          "Running",
			ReadyReplicas:  2,
			URL:            "https://app.example.com",
			InternalURL:    "http://full-app.zenith-test.svc.cluster.local:8080",
			CurrentImage:   "nginx:latest",
			LastDeployedAt: &now,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
		},
	}

	copied := app.DeepCopy()

	// Verify all sub-struct fields
	if copied.Spec.Domain != "app.example.com" {
		t.Errorf("DeepCopy failed: Domain = %q", copied.Spec.Domain)
	}
	if len(copied.Spec.Env) != 2 {
		t.Fatalf("DeepCopy failed: Env length = %d", len(copied.Spec.Env))
	}
	if copied.Spec.Resources == nil {
		t.Fatal("DeepCopy failed: Resources is nil")
	}
	if copied.Spec.HealthCheck == nil {
		t.Fatal("DeepCopy failed: HealthCheck is nil")
	}
	if copied.Spec.HealthCheck.Path != "/health" {
		t.Errorf("DeepCopy failed: HealthCheck.Path = %q", copied.Spec.HealthCheck.Path)
	}
	if copied.Spec.HealthCheck.Port != 8081 {
		t.Errorf("DeepCopy failed: HealthCheck.Port = %d", copied.Spec.HealthCheck.Port)
	}
	if copied.Spec.AutoScale == nil {
		t.Fatal("DeepCopy failed: AutoScale is nil")
	}
	if copied.Spec.AutoScale.MinReplicas != 2 {
		t.Errorf("DeepCopy failed: AutoScale.MinReplicas = %d", copied.Spec.AutoScale.MinReplicas)
	}
	if copied.Spec.BuildSource == nil {
		t.Fatal("DeepCopy failed: BuildSource is nil")
	}
	if copied.Spec.BuildSource.GitURL != "https://github.com/example/repo.git" {
		t.Errorf("DeepCopy failed: BuildSource.GitURL = %q", copied.Spec.BuildSource.GitURL)
	}

	// Verify status
	if copied.Status.LastDeployedAt == nil {
		t.Fatal("DeepCopy failed: LastDeployedAt is nil")
	}
	if len(copied.Status.Conditions) != 1 {
		t.Fatalf("DeepCopy failed: Conditions length = %d", len(copied.Status.Conditions))
	}

	// Ensure pointer independence for HealthCheck
	copied.Spec.HealthCheck.Path = "/changed"
	if app.Spec.HealthCheck.Path == "/changed" {
		t.Error("DeepCopy is not independent for HealthCheck pointer")
	}

	// Ensure pointer independence for AutoScale
	copied.Spec.AutoScale.MaxReplicas = 99
	if app.Spec.AutoScale.MaxReplicas == 99 {
		t.Error("DeepCopy is not independent for AutoScale pointer")
	}

	// Ensure pointer independence for BuildSource
	copied.Spec.BuildSource.Branch = "develop"
	if app.Spec.BuildSource.Branch == "develop" {
		t.Error("DeepCopy is not independent for BuildSource pointer")
	}

	// Ensure pointer independence for Resources
	copied.Spec.Resources.CPU = resource.MustParse("1")
	if app.Spec.Resources.CPU.Cmp(resource.MustParse("500m")) != 0 {
		t.Error("DeepCopy is not independent for Resources pointer")
	}

	// Ensure Env slice independence
	copied.Spec.Env[0].Value = "changed"
	if app.Spec.Env[0].Value == "changed" {
		t.Error("DeepCopy is not independent for Env slice")
	}

	// Ensure LastDeployedAt independence
	newTime := metav1.NewTime(time.Now().Add(time.Hour))
	copied.Status.LastDeployedAt = &newTime
	if app.Status.LastDeployedAt.Equal(&newTime) {
		t.Error("DeepCopy is not independent for LastDeployedAt pointer")
	}
}

func TestAppDeepCopyWithNilOptionalFields(t *testing.T) {
	app := &App{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-app"},
		Spec: AppSpec{
			Image:       "nginx:latest",
			Replicas:    nil,
			Port:        8080,
			Env:         nil,
			Resources:   nil,
			HealthCheck: nil,
			AutoScale:   nil,
			BuildSource: nil,
		},
		Status: AppStatus{
			Phase:          "Running",
			LastDeployedAt: nil,
			Conditions:     nil,
		},
	}

	copied := app.DeepCopy()
	if copied.Spec.Replicas != nil {
		t.Error("DeepCopy should preserve nil Replicas")
	}
	if copied.Spec.Env != nil {
		t.Error("DeepCopy should preserve nil Env")
	}
	if copied.Spec.Resources != nil {
		t.Error("DeepCopy should preserve nil Resources")
	}
	if copied.Spec.HealthCheck != nil {
		t.Error("DeepCopy should preserve nil HealthCheck")
	}
	if copied.Spec.AutoScale != nil {
		t.Error("DeepCopy should preserve nil AutoScale")
	}
	if copied.Spec.BuildSource != nil {
		t.Error("DeepCopy should preserve nil BuildSource")
	}
	if copied.Status.LastDeployedAt != nil {
		t.Error("DeepCopy should preserve nil LastDeployedAt")
	}
	if copied.Status.Conditions != nil {
		t.Error("DeepCopy should preserve nil Conditions")
	}
}

func TestAppDeepCopyInto(t *testing.T) {
	replicas := int32(1)
	app := &App{
		ObjectMeta: metav1.ObjectMeta{Name: "test-app", Namespace: "ns"},
		Spec:       AppSpec{Image: "nginx", Replicas: &replicas, Port: 80},
	}

	out := &App{}
	app.DeepCopyInto(out)
	if out.Spec.Image != "nginx" {
		t.Errorf("DeepCopyInto failed: Image = %q", out.Spec.Image)
	}
	if out.Spec.Replicas == nil || *out.Spec.Replicas != 1 {
		t.Error("DeepCopyInto failed: Replicas mismatch")
	}

	// Independence
	*out.Spec.Replicas = 5
	if *app.Spec.Replicas != 1 {
		t.Error("DeepCopyInto is not independent")
	}
}

func TestAppDeepCopyObject(t *testing.T) {
	app := &App{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       AppSpec{Image: "nginx"},
	}
	obj := app.DeepCopyObject()
	_, ok := obj.(*App)
	if !ok {
		t.Fatal("DeepCopyObject did not return *App")
	}
}

func TestAppListDeepCopy(t *testing.T) {
	list := &AppList{
		Items: []App{
			{ObjectMeta: metav1.ObjectMeta{Name: "a1"}, Spec: AppSpec{Image: "img1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "a2"}, Spec: AppSpec{Image: "img2"}},
		},
	}

	copied := list.DeepCopy()
	if len(copied.Items) != 2 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}

	copied.Items[0].Spec.Image = "modified"
	if list.Items[0].Spec.Image == "modified" {
		t.Error("DeepCopy is not independent for Items")
	}
}

func TestAppListDeepCopyObject(t *testing.T) {
	list := &AppList{Items: []App{{ObjectMeta: metav1.ObjectMeta{Name: "a1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*AppList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *AppList")
	}
}

func TestAppSpecDeepCopy(t *testing.T) {
	replicas := int32(2)
	spec := &AppSpec{
		Image:    "nginx",
		Replicas: &replicas,
		Port:     8080,
		Env:      []corev1.EnvVar{{Name: "A", Value: "B"}},
	}

	copied := spec.DeepCopy()
	if copied.Image != "nginx" {
		t.Errorf("DeepCopy failed: Image = %q", copied.Image)
	}
	*copied.Replicas = 99
	if *spec.Replicas != 2 {
		t.Error("DeepCopy is not independent for Replicas")
	}
}

func TestAppSpecDeepCopyNil(t *testing.T) {
	var spec *AppSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil AppSpec should return nil")
	}
}

func TestAppStatusDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now())
	status := &AppStatus{
		Phase:          "Running",
		ReadyReplicas:  2,
		URL:            "https://example.com",
		InternalURL:    "http://svc:8080",
		CurrentImage:   "nginx",
		LastDeployedAt: &now,
		Conditions: []metav1.Condition{
			{Type: "Ready", Status: metav1.ConditionTrue},
		},
	}

	copied := status.DeepCopy()
	if copied.Phase != "Running" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
	if copied.LastDeployedAt == nil {
		t.Fatal("DeepCopy failed: LastDeployedAt is nil")
	}
}

func TestAppStatusDeepCopyNil(t *testing.T) {
	var status *AppStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil AppStatus should return nil")
	}
}

func TestAppResourcesDeepCopyInto(t *testing.T) {
	res := &AppResources{
		CPU:    resource.MustParse("500m"),
		Memory: resource.MustParse("256Mi"),
	}

	out := &AppResources{}
	res.DeepCopyInto(out)

	if out.CPU.Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("DeepCopyInto failed: CPU = %s", out.CPU.String())
	}
	if out.Memory.Cmp(resource.MustParse("256Mi")) != 0 {
		t.Errorf("DeepCopyInto failed: Memory = %s", out.Memory.String())
	}
}

// ============================================================================
// Database DeepCopy Tests
// ============================================================================

func TestDatabaseDeepCopy(t *testing.T) {
	db := &Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-db",
			Namespace: "zenith-proj",
		},
		Spec: DatabaseSpec{
			Engine:   "postgresql",
			Version:  "16",
			Storage:  resource.MustParse("20Gi"),
			Replicas: 1,
			Parameters: map[string]string{
				"max_connections": "200",
			},
		},
	}

	copied := db.DeepCopy()
	if copied.Spec.Engine != "postgresql" {
		t.Errorf("DeepCopy failed: Engine mismatch")
	}

	// Ensure map independence
	copied.Spec.Parameters["max_connections"] = "500"
	if db.Spec.Parameters["max_connections"] != "200" {
		t.Error("DeepCopy is not independent for Parameters map")
	}
}

func TestDatabaseImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &Database{}
	var _ runtime.Object = &DatabaseList{}
}

func TestDatabaseDeepCopyWithAllFields(t *testing.T) {
	now := metav1.NewTime(time.Now())
	db := &Database{
		ObjectMeta: metav1.ObjectMeta{Name: "full-db", Namespace: "ns"},
		Spec: DatabaseSpec{
			Engine:   "postgresql",
			Version:  "16",
			Storage:  resource.MustParse("50Gi"),
			Replicas: 3,
			Backup: &BackupConfig{
				Enabled:       true,
				Schedule:      "0 3 * * *",
				RetentionDays: 14,
			},
			Resources: &DatabaseResources{
				CPU:    resource.MustParse("1"),
				Memory: resource.MustParse("2Gi"),
			},
			Parameters: map[string]string{
				"max_connections": "500",
				"shared_buffers":  "1GB",
			},
		},
		Status: DatabaseStatus{
			Phase:            "Ready",
			ConnectionString: "postgresql://...",
			Host:             "db.svc.local",
			Port:             5432,
			HetznerVolumeID:  "12345",
			SecretName:       "full-db-conn",
			StorageUsed:      "10Gi",
			LastBackupTime:   &now,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
		},
	}

	copied := db.DeepCopy()

	// Verify Backup
	if copied.Spec.Backup == nil {
		t.Fatal("DeepCopy failed: Backup is nil")
	}
	if copied.Spec.Backup.Schedule != "0 3 * * *" {
		t.Errorf("DeepCopy failed: Backup.Schedule = %q", copied.Spec.Backup.Schedule)
	}
	copied.Spec.Backup.RetentionDays = 30
	if db.Spec.Backup.RetentionDays == 30 {
		t.Error("DeepCopy is not independent for Backup pointer")
	}

	// Verify Resources
	if copied.Spec.Resources == nil {
		t.Fatal("DeepCopy failed: Resources is nil")
	}
	if copied.Spec.Resources.CPU.Cmp(resource.MustParse("1")) != 0 {
		t.Errorf("DeepCopy failed: Resources.CPU = %s", copied.Spec.Resources.CPU.String())
	}

	// Verify Status fields
	if copied.Status.LastBackupTime == nil {
		t.Fatal("DeepCopy failed: LastBackupTime is nil")
	}
	if len(copied.Status.Conditions) != 1 {
		t.Fatalf("DeepCopy failed: Conditions length = %d", len(copied.Status.Conditions))
	}
}

func TestDatabaseDeepCopyWithNilOptionalFields(t *testing.T) {
	db := &Database{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-db"},
		Spec: DatabaseSpec{
			Engine:     "redis",
			Version:    "7.2",
			Storage:    resource.MustParse("5Gi"),
			Backup:     nil,
			Resources:  nil,
			Parameters: nil,
		},
		Status: DatabaseStatus{
			Phase:          "Ready",
			LastBackupTime: nil,
			Conditions:     nil,
		},
	}

	copied := db.DeepCopy()
	if copied.Spec.Backup != nil {
		t.Error("DeepCopy should preserve nil Backup")
	}
	if copied.Spec.Resources != nil {
		t.Error("DeepCopy should preserve nil Resources")
	}
	if copied.Spec.Parameters != nil {
		t.Error("DeepCopy should preserve nil Parameters")
	}
	if copied.Status.LastBackupTime != nil {
		t.Error("DeepCopy should preserve nil LastBackupTime")
	}
}

func TestDatabaseDeepCopyInto(t *testing.T) {
	db := &Database{
		ObjectMeta: metav1.ObjectMeta{Name: "db1"},
		Spec:       DatabaseSpec{Engine: "postgresql", Version: "16", Storage: resource.MustParse("10Gi")},
	}
	out := &Database{}
	db.DeepCopyInto(out)
	if out.Spec.Engine != "postgresql" {
		t.Errorf("DeepCopyInto failed: Engine = %q", out.Spec.Engine)
	}
}

func TestDatabaseDeepCopyObject(t *testing.T) {
	db := &Database{ObjectMeta: metav1.ObjectMeta{Name: "db"}}
	obj := db.DeepCopyObject()
	_, ok := obj.(*Database)
	if !ok {
		t.Fatal("DeepCopyObject did not return *Database")
	}
}

func TestDatabaseListDeepCopy(t *testing.T) {
	list := &DatabaseList{
		Items: []Database{
			{ObjectMeta: metav1.ObjectMeta{Name: "db1"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "db2"}},
		},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 2 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestDatabaseListDeepCopyObject(t *testing.T) {
	list := &DatabaseList{Items: []Database{{ObjectMeta: metav1.ObjectMeta{Name: "db1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*DatabaseList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *DatabaseList")
	}
}

func TestDatabaseSpecDeepCopy(t *testing.T) {
	spec := &DatabaseSpec{
		Engine:   "postgresql",
		Version:  "16",
		Storage:  resource.MustParse("10Gi"),
		Replicas: 1,
		Parameters: map[string]string{
			"max_connections": "200",
		},
	}
	copied := spec.DeepCopy()
	copied.Parameters["new_key"] = "value"
	if _, ok := spec.Parameters["new_key"]; ok {
		t.Error("DeepCopy is not independent for Parameters map")
	}
}

func TestDatabaseSpecDeepCopyNil(t *testing.T) {
	var spec *DatabaseSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil DatabaseSpec should return nil")
	}
}

func TestDatabaseStatusDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now())
	status := &DatabaseStatus{
		Phase:          "Ready",
		Host:           "host",
		Port:           5432,
		LastBackupTime: &now,
		Conditions:     []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Ready" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
	if copied.LastBackupTime == nil {
		t.Fatal("DeepCopy failed: LastBackupTime is nil")
	}
}

func TestDatabaseStatusDeepCopyNil(t *testing.T) {
	var status *DatabaseStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil DatabaseStatus should return nil")
	}
}

func TestDatabaseResourcesDeepCopyInto(t *testing.T) {
	res := &DatabaseResources{
		CPU:    resource.MustParse("2"),
		Memory: resource.MustParse("4Gi"),
	}
	out := &DatabaseResources{}
	res.DeepCopyInto(out)
	if out.CPU.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("DeepCopyInto failed: CPU = %s", out.CPU.String())
	}
}

// ============================================================================
// StorageBucket DeepCopy Tests
// ============================================================================

func TestStorageBucketDeepCopy(t *testing.T) {
	sb := &StorageBucket{
		ObjectMeta: metav1.ObjectMeta{Name: "my-bucket"},
		Spec: StorageBucketSpec{
			Access:     "private",
			Versioning: true,
			LifecycleRules: []LifecycleRule{
				{Prefix: "logs/", ExpirationDays: 30},
			},
		},
	}

	copied := sb.DeepCopy()
	if copied.Spec.Access != "private" {
		t.Errorf("DeepCopy failed: Access mismatch")
	}
	if len(copied.Spec.LifecycleRules) != 1 {
		t.Errorf("DeepCopy failed: LifecycleRules length mismatch")
	}
}

func TestStorageBucketImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &StorageBucket{}
	var _ runtime.Object = &StorageBucketList{}
}

func TestStorageBucketDeepCopyWithAllFields(t *testing.T) {
	sb := &StorageBucket{
		ObjectMeta: metav1.ObjectMeta{Name: "full-bucket", Namespace: "ns"},
		Spec: StorageBucketSpec{
			Access:     "public-read",
			Versioning: true,
			Region:     "nbg1",
			LifecycleRules: []LifecycleRule{
				{Prefix: "logs/", ExpirationDays: 30, TransitionDays: 7},
				{Prefix: "tmp/", ExpirationDays: 1},
			},
			CORSRules: []CORSRule{
				{
					AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
					AllowedMethods: []string{"GET", "PUT"},
					AllowedHeaders: []string{"Content-Type", "Authorization"},
					MaxAgeSeconds:  3600,
				},
			},
		},
		Status: StorageBucketStatus{
			Phase:       "Ready",
			Endpoint:    "https://fsn1.your-objectstorage.com",
			BucketName:  "zenith-ns-full-bucket",
			SecretName:  "full-bucket-s3-credentials",
			SizeBytes:   1024000,
			ObjectCount: 42,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
		},
	}

	copied := sb.DeepCopy()

	// Verify LifecycleRules independence
	if len(copied.Spec.LifecycleRules) != 2 {
		t.Fatalf("DeepCopy failed: LifecycleRules length = %d", len(copied.Spec.LifecycleRules))
	}
	copied.Spec.LifecycleRules[0].ExpirationDays = 99
	if sb.Spec.LifecycleRules[0].ExpirationDays == 99 {
		t.Error("DeepCopy is not independent for LifecycleRules")
	}

	// Verify CORSRules independence
	if len(copied.Spec.CORSRules) != 1 {
		t.Fatalf("DeepCopy failed: CORSRules length = %d", len(copied.Spec.CORSRules))
	}
	if len(copied.Spec.CORSRules[0].AllowedOrigins) != 2 {
		t.Fatalf("DeepCopy failed: AllowedOrigins length = %d", len(copied.Spec.CORSRules[0].AllowedOrigins))
	}
	copied.Spec.CORSRules[0].AllowedOrigins[0] = "https://changed.com"
	if sb.Spec.CORSRules[0].AllowedOrigins[0] == "https://changed.com" {
		t.Error("DeepCopy is not independent for CORSRules.AllowedOrigins")
	}

	// Verify status
	if copied.Status.SizeBytes != 1024000 {
		t.Errorf("DeepCopy failed: SizeBytes = %d", copied.Status.SizeBytes)
	}
	if copied.Status.ObjectCount != 42 {
		t.Errorf("DeepCopy failed: ObjectCount = %d", copied.Status.ObjectCount)
	}
}

func TestStorageBucketDeepCopyWithNilOptionalFields(t *testing.T) {
	sb := &StorageBucket{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-bucket"},
		Spec: StorageBucketSpec{
			Access:         "private",
			LifecycleRules: nil,
			CORSRules:      nil,
		},
		Status: StorageBucketStatus{
			Conditions: nil,
		},
	}

	copied := sb.DeepCopy()
	if copied.Spec.LifecycleRules != nil {
		t.Error("DeepCopy should preserve nil LifecycleRules")
	}
	if copied.Spec.CORSRules != nil {
		t.Error("DeepCopy should preserve nil CORSRules")
	}
}

func TestStorageBucketDeepCopyInto(t *testing.T) {
	sb := &StorageBucket{
		ObjectMeta: metav1.ObjectMeta{Name: "sb1"},
		Spec:       StorageBucketSpec{Access: "private", Region: "fsn1"},
	}
	out := &StorageBucket{}
	sb.DeepCopyInto(out)
	if out.Spec.Access != "private" {
		t.Errorf("DeepCopyInto failed: Access = %q", out.Spec.Access)
	}
}

func TestStorageBucketDeepCopyObject(t *testing.T) {
	sb := &StorageBucket{ObjectMeta: metav1.ObjectMeta{Name: "sb"}}
	obj := sb.DeepCopyObject()
	_, ok := obj.(*StorageBucket)
	if !ok {
		t.Fatal("DeepCopyObject did not return *StorageBucket")
	}
}

func TestStorageBucketListDeepCopy(t *testing.T) {
	list := &StorageBucketList{
		Items: []StorageBucket{
			{ObjectMeta: metav1.ObjectMeta{Name: "sb1"}},
		},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestStorageBucketListDeepCopyObject(t *testing.T) {
	list := &StorageBucketList{Items: []StorageBucket{{ObjectMeta: metav1.ObjectMeta{Name: "sb1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*StorageBucketList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *StorageBucketList")
	}
}

func TestStorageBucketSpecDeepCopy(t *testing.T) {
	spec := &StorageBucketSpec{
		Access: "private",
		LifecycleRules: []LifecycleRule{
			{Prefix: "logs/", ExpirationDays: 30},
		},
	}
	copied := spec.DeepCopy()
	copied.LifecycleRules = append(copied.LifecycleRules, LifecycleRule{Prefix: "new/"})
	if len(spec.LifecycleRules) != 1 {
		t.Error("DeepCopy is not independent for LifecycleRules")
	}
}

func TestStorageBucketSpecDeepCopyNil(t *testing.T) {
	var spec *StorageBucketSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil StorageBucketSpec should return nil")
	}
}

func TestStorageBucketStatusDeepCopy(t *testing.T) {
	status := &StorageBucketStatus{
		Phase:      "Ready",
		Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Ready" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
}

func TestStorageBucketStatusDeepCopyNil(t *testing.T) {
	var status *StorageBucketStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil StorageBucketStatus should return nil")
	}
}

func TestCORSRuleDeepCopyInto(t *testing.T) {
	rule := &CORSRule{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAgeSeconds:  600,
	}
	out := &CORSRule{}
	rule.DeepCopyInto(out)

	if len(out.AllowedOrigins) != 1 {
		t.Fatalf("DeepCopyInto failed: AllowedOrigins length = %d", len(out.AllowedOrigins))
	}
	out.AllowedOrigins[0] = "changed"
	if rule.AllowedOrigins[0] == "changed" {
		t.Error("DeepCopyInto is not independent for AllowedOrigins")
	}

	out.AllowedMethods[0] = "DELETE"
	if rule.AllowedMethods[0] == "DELETE" {
		t.Error("DeepCopyInto is not independent for AllowedMethods")
	}

	out.AllowedHeaders[0] = "X-Custom"
	if rule.AllowedHeaders[0] == "X-Custom" {
		t.Error("DeepCopyInto is not independent for AllowedHeaders")
	}
}

func TestCORSRuleDeepCopyIntoNilSlices(t *testing.T) {
	rule := &CORSRule{
		AllowedOrigins: nil,
		AllowedMethods: nil,
		AllowedHeaders: nil,
	}
	out := &CORSRule{}
	rule.DeepCopyInto(out)

	if out.AllowedOrigins != nil {
		t.Error("DeepCopyInto should preserve nil AllowedOrigins")
	}
	if out.AllowedMethods != nil {
		t.Error("DeepCopyInto should preserve nil AllowedMethods")
	}
	if out.AllowedHeaders != nil {
		t.Error("DeepCopyInto should preserve nil AllowedHeaders")
	}
}

// ============================================================================
// Domain DeepCopy Tests
// ============================================================================

func TestDomainDeepCopy(t *testing.T) {
	dom := &Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "my-domain"},
		Spec: DomainSpec{
			Domain: "app.example.com",
			AppRef: "my-app",
			SSL: &SSLConfig{
				Enabled: true,
				Issuer:  "letsencrypt-prod",
			},
		},
	}

	copied := dom.DeepCopy()
	if copied.Spec.Domain != "app.example.com" {
		t.Errorf("DeepCopy failed: Domain mismatch")
	}

	copied.Spec.SSL.Issuer = "letsencrypt-staging"
	if dom.Spec.SSL.Issuer != "letsencrypt-prod" {
		t.Error("DeepCopy is not independent for SSL pointer")
	}
}

func TestDomainImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &Domain{}
	var _ runtime.Object = &DomainList{}
}

func TestDomainDeepCopyWithAllFields(t *testing.T) {
	now := metav1.NewTime(time.Now())
	dom := &Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "full-domain", Namespace: "ns"},
		Spec: DomainSpec{
			Domain: "full.example.com",
			AppRef: "web-app",
			SSL: &SSLConfig{
				Enabled: true,
				Issuer:  "letsencrypt-prod",
			},
			DNS: &DNSConfig{
				AutoConfigure: true,
				Type:          "CNAME",
			},
		},
		Status: DomainStatus{
			Phase:             "Active",
			SSLReady:          true,
			DNSConfigured:     true,
			CertificateExpiry: &now,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
		},
	}

	copied := dom.DeepCopy()

	// Verify DNS pointer independence
	if copied.Spec.DNS == nil {
		t.Fatal("DeepCopy failed: DNS is nil")
	}
	copied.Spec.DNS.Type = "A"
	if dom.Spec.DNS.Type == "A" {
		t.Error("DeepCopy is not independent for DNS pointer")
	}

	// Verify SSL pointer independence
	copied.Spec.SSL.Enabled = false
	if !dom.Spec.SSL.Enabled {
		t.Error("DeepCopy is not independent for SSL pointer")
	}

	// Verify CertificateExpiry independence
	if copied.Status.CertificateExpiry == nil {
		t.Fatal("DeepCopy failed: CertificateExpiry is nil")
	}
}

func TestDomainDeepCopyWithNilOptionalFields(t *testing.T) {
	dom := &Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-domain"},
		Spec: DomainSpec{
			Domain: "example.com",
			AppRef: "app",
			SSL:    nil,
			DNS:    nil,
		},
		Status: DomainStatus{
			CertificateExpiry: nil,
			Conditions:        nil,
		},
	}

	copied := dom.DeepCopy()
	if copied.Spec.SSL != nil {
		t.Error("DeepCopy should preserve nil SSL")
	}
	if copied.Spec.DNS != nil {
		t.Error("DeepCopy should preserve nil DNS")
	}
	if copied.Status.CertificateExpiry != nil {
		t.Error("DeepCopy should preserve nil CertificateExpiry")
	}
}

func TestDomainDeepCopyInto(t *testing.T) {
	dom := &Domain{
		ObjectMeta: metav1.ObjectMeta{Name: "dom1"},
		Spec:       DomainSpec{Domain: "example.com", AppRef: "app"},
	}
	out := &Domain{}
	dom.DeepCopyInto(out)
	if out.Spec.Domain != "example.com" {
		t.Errorf("DeepCopyInto failed: Domain = %q", out.Spec.Domain)
	}
}

func TestDomainDeepCopyObject(t *testing.T) {
	dom := &Domain{ObjectMeta: metav1.ObjectMeta{Name: "dom"}}
	obj := dom.DeepCopyObject()
	_, ok := obj.(*Domain)
	if !ok {
		t.Fatal("DeepCopyObject did not return *Domain")
	}
}

func TestDomainListDeepCopy(t *testing.T) {
	list := &DomainList{
		Items: []Domain{{ObjectMeta: metav1.ObjectMeta{Name: "d1"}}},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestDomainListDeepCopyObject(t *testing.T) {
	list := &DomainList{Items: []Domain{{ObjectMeta: metav1.ObjectMeta{Name: "d1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*DomainList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *DomainList")
	}
}

func TestDomainSpecDeepCopy(t *testing.T) {
	spec := &DomainSpec{
		Domain: "example.com",
		AppRef: "app",
		SSL:    &SSLConfig{Enabled: true, Issuer: "letsencrypt-prod"},
		DNS:    &DNSConfig{AutoConfigure: true, Type: "A"},
	}
	copied := spec.DeepCopy()
	copied.SSL.Issuer = "changed"
	if spec.SSL.Issuer == "changed" {
		t.Error("DeepCopy is not independent for SSL")
	}
	copied.DNS.Type = "CNAME"
	if spec.DNS.Type == "CNAME" {
		t.Error("DeepCopy is not independent for DNS")
	}
}

func TestDomainSpecDeepCopyNil(t *testing.T) {
	var spec *DomainSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil DomainSpec should return nil")
	}
}

func TestDomainStatusDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now())
	status := &DomainStatus{
		Phase:             "Active",
		SSLReady:          true,
		DNSConfigured:     true,
		CertificateExpiry: &now,
		Conditions:        []metav1.Condition{{Type: "Ready"}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Active" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
}

func TestDomainStatusDeepCopyNil(t *testing.T) {
	var status *DomainStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil DomainStatus should return nil")
	}
}

// ============================================================================
// AuthRealm DeepCopy Tests
// ============================================================================

func TestAuthRealmDeepCopy(t *testing.T) {
	realm := &AuthRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "my-realm"},
		Spec: AuthRealmSpec{
			DisplayName: "My Realm",
			Providers: []IdentityProvider{
				{Name: "google", Type: "google", ClientID: "xxx"},
			},
			Clients: []AuthClient{
				{Name: "web-app", Type: "public", RedirectURIs: []string{"https://app.example.com/callback"}},
			},
		},
	}

	copied := realm.DeepCopy()
	if copied.Spec.DisplayName != "My Realm" {
		t.Errorf("DeepCopy failed: DisplayName mismatch")
	}
	if len(copied.Spec.Providers) != 1 {
		t.Errorf("DeepCopy failed: Providers length mismatch")
	}
}

func TestAuthRealmImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &AuthRealm{}
	var _ runtime.Object = &AuthRealmList{}
}

func TestAuthRealmDeepCopyWithAllFields(t *testing.T) {
	realm := &AuthRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "full-realm", Namespace: "ns"},
		Spec: AuthRealmSpec{
			DisplayName: "Full Realm",
			Providers: []IdentityProvider{
				{
					Name:     "google",
					Type:     "google",
					ClientID: "google-id",
					ClientSecretRef: &SecretKeyRef{
						Name: "google-secret",
						Key:  "client-secret",
					},
					Enabled: true,
					Config: map[string]string{
						"hostedDomain": "example.com",
					},
				},
				{
					Name:     "github",
					Type:     "github",
					ClientID: "github-id",
					Enabled:  true,
				},
			},
			Clients: []AuthClient{
				{
					Name:         "web-app",
					Type:         "public",
					RedirectURIs: []string{"https://app.example.com/callback", "http://localhost:3000/callback"},
					Scopes:       []string{"openid", "profile", "email"},
				},
				{
					Name:         "api",
					Type:         "confidential",
					RedirectURIs: []string{"https://api.example.com/callback"},
					Scopes:       []string{"api:read", "api:write"},
				},
			},
			Settings: &RealmSettings{
				MFARequired:    true,
				SessionTimeout: "12h",
				PasswordPolicy: &PasswordPolicy{
					MinLength:        12,
					RequireUppercase: true,
					RequireNumbers:   true,
					RequireSpecial:   true,
				},
			},
		},
		Status: AuthRealmStatus{
			Phase:       "Ready",
			Endpoint:    "https://auth.zenith.dev/realms/full-realm/.well-known/openid-configuration",
			UserCount:   150,
			ClientCount: 2,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue},
			},
		},
	}

	copied := realm.DeepCopy()

	// Verify Providers deep copy with ClientSecretRef
	if len(copied.Spec.Providers) != 2 {
		t.Fatalf("DeepCopy failed: Providers length = %d", len(copied.Spec.Providers))
	}
	if copied.Spec.Providers[0].ClientSecretRef == nil {
		t.Fatal("DeepCopy failed: ClientSecretRef is nil")
	}
	copied.Spec.Providers[0].ClientSecretRef.Name = "changed"
	if realm.Spec.Providers[0].ClientSecretRef.Name == "changed" {
		t.Error("DeepCopy is not independent for ClientSecretRef pointer")
	}

	// Verify Provider Config map independence
	copied.Spec.Providers[0].Config["hostedDomain"] = "changed.com"
	if realm.Spec.Providers[0].Config["hostedDomain"] == "changed.com" {
		t.Error("DeepCopy is not independent for Provider Config map")
	}

	// Verify Clients deep copy
	if len(copied.Spec.Clients) != 2 {
		t.Fatalf("DeepCopy failed: Clients length = %d", len(copied.Spec.Clients))
	}
	copied.Spec.Clients[0].RedirectURIs[0] = "https://changed.com"
	if realm.Spec.Clients[0].RedirectURIs[0] == "https://changed.com" {
		t.Error("DeepCopy is not independent for Client RedirectURIs")
	}

	copied.Spec.Clients[0].Scopes[0] = "changed"
	if realm.Spec.Clients[0].Scopes[0] == "changed" {
		t.Error("DeepCopy is not independent for Client Scopes")
	}

	// Verify Settings deep copy
	if copied.Spec.Settings == nil {
		t.Fatal("DeepCopy failed: Settings is nil")
	}
	if !copied.Spec.Settings.MFARequired {
		t.Error("DeepCopy failed: MFARequired mismatch")
	}
	if copied.Spec.Settings.PasswordPolicy == nil {
		t.Fatal("DeepCopy failed: PasswordPolicy is nil")
	}
	copied.Spec.Settings.PasswordPolicy.MinLength = 99
	if realm.Spec.Settings.PasswordPolicy.MinLength == 99 {
		t.Error("DeepCopy is not independent for PasswordPolicy pointer")
	}
}

func TestAuthRealmDeepCopyWithNilOptionalFields(t *testing.T) {
	realm := &AuthRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-realm"},
		Spec: AuthRealmSpec{
			DisplayName: "Minimal",
			Providers:   nil,
			Clients:     nil,
			Settings:    nil,
		},
	}

	copied := realm.DeepCopy()
	if copied.Spec.Providers != nil {
		t.Error("DeepCopy should preserve nil Providers")
	}
	if copied.Spec.Clients != nil {
		t.Error("DeepCopy should preserve nil Clients")
	}
	if copied.Spec.Settings != nil {
		t.Error("DeepCopy should preserve nil Settings")
	}
}

func TestAuthRealmDeepCopyInto(t *testing.T) {
	realm := &AuthRealm{
		ObjectMeta: metav1.ObjectMeta{Name: "realm1"},
		Spec:       AuthRealmSpec{DisplayName: "Realm 1"},
	}
	out := &AuthRealm{}
	realm.DeepCopyInto(out)
	if out.Spec.DisplayName != "Realm 1" {
		t.Errorf("DeepCopyInto failed: DisplayName = %q", out.Spec.DisplayName)
	}
}

func TestAuthRealmDeepCopyObject(t *testing.T) {
	realm := &AuthRealm{ObjectMeta: metav1.ObjectMeta{Name: "realm"}}
	obj := realm.DeepCopyObject()
	_, ok := obj.(*AuthRealm)
	if !ok {
		t.Fatal("DeepCopyObject did not return *AuthRealm")
	}
}

func TestAuthRealmListDeepCopy(t *testing.T) {
	list := &AuthRealmList{
		Items: []AuthRealm{{ObjectMeta: metav1.ObjectMeta{Name: "r1"}}},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestAuthRealmListDeepCopyObject(t *testing.T) {
	list := &AuthRealmList{Items: []AuthRealm{{ObjectMeta: metav1.ObjectMeta{Name: "r1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*AuthRealmList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *AuthRealmList")
	}
}

func TestAuthRealmSpecDeepCopy(t *testing.T) {
	spec := &AuthRealmSpec{
		DisplayName: "Test",
		Providers:   []IdentityProvider{{Name: "p1", Type: "google"}},
		Clients:     []AuthClient{{Name: "c1", Type: "public"}},
		Settings:    &RealmSettings{MFARequired: true},
	}
	copied := spec.DeepCopy()
	copied.Settings.MFARequired = false
	if !spec.Settings.MFARequired {
		t.Error("DeepCopy is not independent for Settings")
	}
}

func TestAuthRealmSpecDeepCopyNil(t *testing.T) {
	var spec *AuthRealmSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil AuthRealmSpec should return nil")
	}
}

func TestAuthRealmStatusDeepCopy(t *testing.T) {
	status := &AuthRealmStatus{
		Phase:       "Ready",
		Endpoint:    "https://auth.zenith.dev",
		UserCount:   10,
		ClientCount: 2,
		Conditions:  []metav1.Condition{{Type: "Ready"}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Ready" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
}

func TestAuthRealmStatusDeepCopyNil(t *testing.T) {
	var status *AuthRealmStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil AuthRealmStatus should return nil")
	}
}

func TestIdentityProviderDeepCopyInto(t *testing.T) {
	provider := &IdentityProvider{
		Name:     "google",
		Type:     "google",
		ClientID: "xxx",
		ClientSecretRef: &SecretKeyRef{
			Name: "secret",
			Key:  "key",
		},
		Enabled: true,
		Config: map[string]string{
			"hostedDomain": "example.com",
		},
	}
	out := &IdentityProvider{}
	provider.DeepCopyInto(out)

	if out.ClientSecretRef == nil {
		t.Fatal("DeepCopyInto failed: ClientSecretRef is nil")
	}
	out.ClientSecretRef.Name = "changed"
	if provider.ClientSecretRef.Name == "changed" {
		t.Error("DeepCopyInto is not independent for ClientSecretRef")
	}

	out.Config["hostedDomain"] = "changed.com"
	if provider.Config["hostedDomain"] == "changed.com" {
		t.Error("DeepCopyInto is not independent for Config")
	}
}

func TestIdentityProviderDeepCopyIntoNilFields(t *testing.T) {
	provider := &IdentityProvider{
		Name:            "simple",
		Type:            "oidc",
		ClientSecretRef: nil,
		Config:          nil,
	}
	out := &IdentityProvider{}
	provider.DeepCopyInto(out)
	if out.ClientSecretRef != nil {
		t.Error("DeepCopyInto should preserve nil ClientSecretRef")
	}
	if out.Config != nil {
		t.Error("DeepCopyInto should preserve nil Config")
	}
}

func TestAuthClientDeepCopyInto(t *testing.T) {
	client := &AuthClient{
		Name:         "web-app",
		Type:         "public",
		RedirectURIs: []string{"https://app.example.com/callback"},
		Scopes:       []string{"openid", "profile"},
	}
	out := &AuthClient{}
	client.DeepCopyInto(out)

	out.RedirectURIs[0] = "changed"
	if client.RedirectURIs[0] == "changed" {
		t.Error("DeepCopyInto is not independent for RedirectURIs")
	}

	out.Scopes[0] = "changed"
	if client.Scopes[0] == "changed" {
		t.Error("DeepCopyInto is not independent for Scopes")
	}
}

func TestAuthClientDeepCopyIntoNilSlices(t *testing.T) {
	client := &AuthClient{
		Name:         "minimal",
		RedirectURIs: nil,
		Scopes:       nil,
	}
	out := &AuthClient{}
	client.DeepCopyInto(out)
	if out.RedirectURIs != nil {
		t.Error("DeepCopyInto should preserve nil RedirectURIs")
	}
	if out.Scopes != nil {
		t.Error("DeepCopyInto should preserve nil Scopes")
	}
}

func TestRealmSettingsDeepCopyInto(t *testing.T) {
	settings := &RealmSettings{
		MFARequired:    true,
		SessionTimeout: "24h",
		PasswordPolicy: &PasswordPolicy{
			MinLength:        12,
			RequireUppercase: true,
			RequireNumbers:   true,
			RequireSpecial:   false,
		},
	}
	out := &RealmSettings{}
	settings.DeepCopyInto(out)

	if out.PasswordPolicy == nil {
		t.Fatal("DeepCopyInto failed: PasswordPolicy is nil")
	}
	out.PasswordPolicy.MinLength = 99
	if settings.PasswordPolicy.MinLength == 99 {
		t.Error("DeepCopyInto is not independent for PasswordPolicy")
	}
}

func TestRealmSettingsDeepCopyIntoNilPasswordPolicy(t *testing.T) {
	settings := &RealmSettings{
		MFARequired:    false,
		SessionTimeout: "24h",
		PasswordPolicy: nil,
	}
	out := &RealmSettings{}
	settings.DeepCopyInto(out)
	if out.PasswordPolicy != nil {
		t.Error("DeepCopyInto should preserve nil PasswordPolicy")
	}
}

// ============================================================================
// GitSync DeepCopy Tests
// ============================================================================

func TestGitSyncDeepCopy(t *testing.T) {
	gs := &GitSync{
		ObjectMeta: metav1.ObjectMeta{Name: "my-sync", Namespace: "ns"},
		Spec: GitSyncSpec{
			RepoURL:  "https://github.com/example/repo.git",
			Branch:   "main",
			Path:     "/manifests",
			Interval: "10m",
			SecretRef: &SecretKeyRef{
				Name: "git-credentials",
				Key:  "token",
			},
			AutoSync:       true,
			PruneResources: false,
		},
		Status: GitSyncStatus{
			Phase:           "Synced",
			LastCommitHash:  "abc123",
			SyncedResources: 5,
			Message:         "OK",
		},
	}

	copied := gs.DeepCopy()
	if copied.Spec.RepoURL != "https://github.com/example/repo.git" {
		t.Errorf("DeepCopy failed: RepoURL = %q", copied.Spec.RepoURL)
	}
	if copied.Spec.SecretRef == nil {
		t.Fatal("DeepCopy failed: SecretRef is nil")
	}

	// Verify SecretRef independence
	copied.Spec.SecretRef.Name = "changed"
	if gs.Spec.SecretRef.Name == "changed" {
		t.Error("DeepCopy is not independent for SecretRef pointer")
	}
}

func TestGitSyncDeepCopyWithNilOptionalFields(t *testing.T) {
	gs := &GitSync{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-sync"},
		Spec: GitSyncSpec{
			RepoURL:   "https://github.com/example/repo.git",
			SecretRef: nil,
		},
		Status: GitSyncStatus{
			LastSyncTime: nil,
			Conditions:   nil,
		},
	}

	copied := gs.DeepCopy()
	if copied.Spec.SecretRef != nil {
		t.Error("DeepCopy should preserve nil SecretRef")
	}
	if copied.Status.LastSyncTime != nil {
		t.Error("DeepCopy should preserve nil LastSyncTime")
	}
}

func TestGitSyncDeepCopyInto(t *testing.T) {
	gs := &GitSync{
		ObjectMeta: metav1.ObjectMeta{Name: "gs1"},
		Spec:       GitSyncSpec{RepoURL: "https://github.com/test/repo.git"},
	}
	out := &GitSync{}
	gs.DeepCopyInto(out)
	if out.Spec.RepoURL != "https://github.com/test/repo.git" {
		t.Errorf("DeepCopyInto failed: RepoURL = %q", out.Spec.RepoURL)
	}
}

func TestGitSyncDeepCopyObject(t *testing.T) {
	gs := &GitSync{ObjectMeta: metav1.ObjectMeta{Name: "gs"}}
	obj := gs.DeepCopyObject()
	_, ok := obj.(*GitSync)
	if !ok {
		t.Fatal("DeepCopyObject did not return *GitSync")
	}
}

func TestGitSyncListDeepCopy(t *testing.T) {
	list := &GitSyncList{
		Items: []GitSync{{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}}},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestGitSyncListDeepCopyObject(t *testing.T) {
	list := &GitSyncList{Items: []GitSync{{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*GitSyncList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *GitSyncList")
	}
}

func TestGitSyncSpecDeepCopy(t *testing.T) {
	spec := &GitSyncSpec{
		RepoURL:   "https://example.com/repo",
		Branch:    "main",
		SecretRef: &SecretKeyRef{Name: "secret", Key: "key"},
	}
	copied := spec.DeepCopy()
	copied.SecretRef.Name = "changed"
	if spec.SecretRef.Name == "changed" {
		t.Error("DeepCopy is not independent for SecretRef")
	}
}

func TestGitSyncSpecDeepCopyNil(t *testing.T) {
	var spec *GitSyncSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil GitSyncSpec should return nil")
	}
}

func TestGitSyncStatusDeepCopy(t *testing.T) {
	now := metav1.NewTime(time.Now())
	status := &GitSyncStatus{
		Phase:          "Synced",
		LastSyncTime:   &now,
		LastCommitHash: "abc123",
		Conditions:     []metav1.Condition{{Type: "Ready"}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Synced" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
	if copied.LastSyncTime == nil {
		t.Fatal("DeepCopy failed: LastSyncTime is nil")
	}
}

func TestGitSyncStatusDeepCopyNil(t *testing.T) {
	var status *GitSyncStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil GitSyncStatus should return nil")
	}
}

// ============================================================================
// CrossplaneResource DeepCopy Tests
// ============================================================================

func TestCrossplaneResourceDeepCopy(t *testing.T) {
	cr := &CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-bucket",
			Namespace: "zenith-test",
		},
		Spec: CrossplaneResourceSpec{
			Provider:          "aws",
			ResourceKind:      "Bucket",
			ProviderConfigRef: "default",
			DeletionPolicy:    "Delete",
			Config: map[string]string{
				"region": "eu-central-1",
				"acl":    "private",
			},
			WriteConnectionSecretToRef: &SecretKeyRef{
				Name: "bucket-conn",
				Key:  "endpoint",
			},
		},
		Status: CrossplaneResourceStatus{
			Phase:                  "Ready",
			CrossplaneResourceName: "zenith-test-my-bucket",
			CrossplaneReady:        true,
		},
	}

	copied := cr.DeepCopy()
	if copied.Spec.Provider != "aws" {
		t.Errorf("DeepCopy failed: Provider mismatch")
	}
	if copied.Spec.ResourceKind != "Bucket" {
		t.Errorf("DeepCopy failed: ResourceKind mismatch")
	}
	if copied.Spec.Config["region"] != "eu-central-1" {
		t.Errorf("DeepCopy failed: Config region mismatch")
	}

	// Ensure map independence
	copied.Spec.Config["region"] = "us-east-1"
	if cr.Spec.Config["region"] != "eu-central-1" {
		t.Error("DeepCopy is not independent for Config map")
	}

	// Ensure pointer independence
	copied.Spec.WriteConnectionSecretToRef.Name = "other-secret"
	if cr.Spec.WriteConnectionSecretToRef.Name != "bucket-conn" {
		t.Error("DeepCopy is not independent for WriteConnectionSecretToRef pointer")
	}
}

func TestCrossplaneResourceImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &CrossplaneResource{}
	var _ runtime.Object = &CrossplaneResourceList{}
}

func TestCrossplaneResourceDeepCopyWithNilOptionalFields(t *testing.T) {
	cr := &CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal-cr"},
		Spec: CrossplaneResourceSpec{
			Provider:                   "aws",
			ResourceKind:               "Bucket",
			Config:                     nil,
			WriteConnectionSecretToRef: nil,
		},
		Status: CrossplaneResourceStatus{
			Conditions: nil,
		},
	}

	copied := cr.DeepCopy()
	if copied.Spec.Config != nil {
		t.Error("DeepCopy should preserve nil Config")
	}
	if copied.Spec.WriteConnectionSecretToRef != nil {
		t.Error("DeepCopy should preserve nil WriteConnectionSecretToRef")
	}
	if copied.Status.Conditions != nil {
		t.Error("DeepCopy should preserve nil Conditions")
	}
}

func TestCrossplaneResourceDeepCopyInto(t *testing.T) {
	cr := &CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{Name: "cr1"},
		Spec: CrossplaneResourceSpec{
			Provider:     "aws",
			ResourceKind: "Bucket",
		},
	}
	out := &CrossplaneResource{}
	cr.DeepCopyInto(out)
	if out.Spec.Provider != "aws" {
		t.Errorf("DeepCopyInto failed: Provider = %q", out.Spec.Provider)
	}
}

func TestCrossplaneResourceDeepCopyObject(t *testing.T) {
	cr := &CrossplaneResource{ObjectMeta: metav1.ObjectMeta{Name: "cr"}}
	obj := cr.DeepCopyObject()
	_, ok := obj.(*CrossplaneResource)
	if !ok {
		t.Fatal("DeepCopyObject did not return *CrossplaneResource")
	}
}

func TestCrossplaneResourceListDeepCopy(t *testing.T) {
	list := &CrossplaneResourceList{
		Items: []CrossplaneResource{{ObjectMeta: metav1.ObjectMeta{Name: "cr1"}}},
	}
	copied := list.DeepCopy()
	if len(copied.Items) != 1 {
		t.Fatalf("DeepCopy failed: Items length = %d", len(copied.Items))
	}
}

func TestCrossplaneResourceListDeepCopyObject(t *testing.T) {
	list := &CrossplaneResourceList{Items: []CrossplaneResource{{ObjectMeta: metav1.ObjectMeta{Name: "cr1"}}}}
	obj := list.DeepCopyObject()
	_, ok := obj.(*CrossplaneResourceList)
	if !ok {
		t.Fatal("DeepCopyObject did not return *CrossplaneResourceList")
	}
}

func TestCrossplaneResourceSpecDeepCopy(t *testing.T) {
	spec := &CrossplaneResourceSpec{
		Provider:     "aws",
		ResourceKind: "Bucket",
		Config:       map[string]string{"region": "eu-central-1"},
		WriteConnectionSecretToRef: &SecretKeyRef{Name: "s", Key: "k"},
	}
	copied := spec.DeepCopy()
	copied.Config["region"] = "changed"
	if spec.Config["region"] == "changed" {
		t.Error("DeepCopy is not independent for Config")
	}
	copied.WriteConnectionSecretToRef.Name = "changed"
	if spec.WriteConnectionSecretToRef.Name == "changed" {
		t.Error("DeepCopy is not independent for WriteConnectionSecretToRef")
	}
}

func TestCrossplaneResourceSpecDeepCopyNil(t *testing.T) {
	var spec *CrossplaneResourceSpec
	if spec.DeepCopy() != nil {
		t.Error("DeepCopy of nil CrossplaneResourceSpec should return nil")
	}
}

func TestCrossplaneResourceStatusDeepCopy(t *testing.T) {
	status := &CrossplaneResourceStatus{
		Phase:       "Ready",
		Conditions:  []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}},
	}
	copied := status.DeepCopy()
	if copied.Phase != "Ready" {
		t.Errorf("DeepCopy failed: Phase = %q", copied.Phase)
	}
	copied.Conditions[0].Status = metav1.ConditionFalse
	if status.Conditions[0].Status != metav1.ConditionTrue {
		t.Error("DeepCopy is not independent for Conditions")
	}
}

func TestCrossplaneResourceStatusDeepCopyNil(t *testing.T) {
	var status *CrossplaneResourceStatus
	if status.DeepCopy() != nil {
		t.Error("DeepCopy of nil CrossplaneResourceStatus should return nil")
	}
}

// ============================================================================
// Nil DeepCopy Tests (for all root types)
// ============================================================================

func TestNilDeepCopy(t *testing.T) {
	var proj *Project
	if proj.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}

	var app *App
	if app.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}

	var db *Database
	if db.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}

	var sb *StorageBucket
	if sb.DeepCopy() != nil {
		t.Error("DeepCopy of nil StorageBucket should return nil")
	}

	var dom *Domain
	if dom.DeepCopy() != nil {
		t.Error("DeepCopy of nil Domain should return nil")
	}

	var realm *AuthRealm
	if realm.DeepCopy() != nil {
		t.Error("DeepCopy of nil AuthRealm should return nil")
	}

	var gs *GitSync
	if gs.DeepCopy() != nil {
		t.Error("DeepCopy of nil GitSync should return nil")
	}

	var cr *CrossplaneResource
	if cr.DeepCopy() != nil {
		t.Error("DeepCopy of nil CrossplaneResource should return nil")
	}
}

func TestNilListDeepCopy(t *testing.T) {
	var projList *ProjectList
	if projList.DeepCopy() != nil {
		t.Error("DeepCopy of nil ProjectList should return nil")
	}

	var appList *AppList
	if appList.DeepCopy() != nil {
		t.Error("DeepCopy of nil AppList should return nil")
	}

	var dbList *DatabaseList
	if dbList.DeepCopy() != nil {
		t.Error("DeepCopy of nil DatabaseList should return nil")
	}

	var sbList *StorageBucketList
	if sbList.DeepCopy() != nil {
		t.Error("DeepCopy of nil StorageBucketList should return nil")
	}

	var domList *DomainList
	if domList.DeepCopy() != nil {
		t.Error("DeepCopy of nil DomainList should return nil")
	}

	var realmList *AuthRealmList
	if realmList.DeepCopy() != nil {
		t.Error("DeepCopy of nil AuthRealmList should return nil")
	}

	var gsList *GitSyncList
	if gsList.DeepCopy() != nil {
		t.Error("DeepCopy of nil GitSyncList should return nil")
	}

	var crList *CrossplaneResourceList
	if crList.DeepCopy() != nil {
		t.Error("DeepCopy of nil CrossplaneResourceList should return nil")
	}
}

// ============================================================================
// Scheme Registration Tests
// ============================================================================

func TestSchemeRegistration(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("Failed to add to scheme: %v", err)
	}

	// Verify all types are registered
	types := []runtime.Object{
		&Project{}, &ProjectList{},
		&App{}, &AppList{},
		&Database{}, &DatabaseList{},
		&StorageBucket{}, &StorageBucketList{},
		&Domain{}, &DomainList{},
		&AuthRealm{}, &AuthRealmList{},
		&GitSync{}, &GitSyncList{},
		&CrossplaneResource{}, &CrossplaneResourceList{},
	}

	for _, obj := range types {
		gvks, _, err := s.ObjectKinds(obj)
		if err != nil {
			t.Errorf("Type %T not registered in scheme: %v", obj, err)
		}
		if len(gvks) == 0 {
			t.Errorf("No GVK found for type %T", obj)
		}
	}
}

func TestSchemeRegistrationGroupVersion(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("Failed to add to scheme: %v", err)
	}

	// Verify the GroupVersion
	gvks, _, err := s.ObjectKinds(&Project{})
	if err != nil {
		t.Fatalf("Failed to get GVKs: %v", err)
	}
	if len(gvks) == 0 {
		t.Fatal("No GVKs found")
	}
	if gvks[0].Group != "zenith.dev" {
		t.Errorf("Expected group 'zenith.dev', got '%s'", gvks[0].Group)
	}
	if gvks[0].Version != "v1alpha1" {
		t.Errorf("Expected version 'v1alpha1', got '%s'", gvks[0].Version)
	}
}

func TestSchemeRegistrationKindNames(t *testing.T) {
	s := runtime.NewScheme()
	if err := AddToScheme(s); err != nil {
		t.Fatalf("Failed to add to scheme: %v", err)
	}

	expectedKinds := map[runtime.Object]string{
		&Project{}:             "Project",
		&ProjectList{}:         "ProjectList",
		&App{}:                 "App",
		&AppList{}:             "AppList",
		&Database{}:            "Database",
		&DatabaseList{}:        "DatabaseList",
		&StorageBucket{}:       "StorageBucket",
		&StorageBucketList{}:   "StorageBucketList",
		&Domain{}:              "Domain",
		&DomainList{}:          "DomainList",
		&AuthRealm{}:           "AuthRealm",
		&AuthRealmList{}:       "AuthRealmList",
		&GitSync{}:             "GitSync",
		&GitSyncList{}:         "GitSyncList",
		&CrossplaneResource{}:     "CrossplaneResource",
		&CrossplaneResourceList{}: "CrossplaneResourceList",
	}

	for obj, expectedKind := range expectedKinds {
		gvks, _, err := s.ObjectKinds(obj)
		if err != nil {
			t.Errorf("Type %T not registered: %v", obj, err)
			continue
		}
		if gvks[0].Kind != expectedKind {
			t.Errorf("Expected kind '%s' for %T, got '%s'", expectedKind, obj, gvks[0].Kind)
		}
	}
}

func TestGroupVersionValues(t *testing.T) {
	if GroupVersion.Group != "zenith.dev" {
		t.Errorf("Expected group 'zenith.dev', got '%s'", GroupVersion.Group)
	}
	if GroupVersion.Version != "v1alpha1" {
		t.Errorf("Expected version 'v1alpha1', got '%s'", GroupVersion.Version)
	}
}

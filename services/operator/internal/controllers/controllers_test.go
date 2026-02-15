package controllers

import (
	"context"
	"testing"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setupScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	clientgoscheme.AddToScheme(s)
	zenithv1.AddToScheme(s)
	return s
}

func TestProjectReconciler_CreateNamespace(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Test Project",
			Owner:       "user@test.com",
			Plan:        "free",
			Region:      "fsn1",
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(client, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-project"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue")
	}

	// Verify namespace was created
	ns := &corev1.Namespace{}
	err = client.Get(context.Background(), types.NamespacedName{Name: "zenith-test-project"}, ns)
	if err != nil {
		t.Fatalf("Expected namespace to be created: %v", err)
	}

	if ns.Labels["zenith.dev/project"] != "test-project" {
		t.Errorf("Expected project label, got '%s'", ns.Labels["zenith.dev/project"])
	}
}

func TestProjectReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(client, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

func TestAppReconciler_CreateDeploymentAndService(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(2)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
		},
	}

	// Create the namespace first
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(client, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "web-app", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue")
	}
}

func TestDatabaseReconciler_SetPortByEngine(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-db",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "postgresql",
			Version: "16",
			Storage: resource.MustParse("20Gi"),
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("test-token")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(client, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-db", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify status was updated
	updatedDB := &zenithv1.Database{}
	client.Get(context.Background(), types.NamespacedName{Name: "test-db", Namespace: "zenith-test"}, updatedDB)

	if updatedDB.Status.Port != 5432 {
		t.Errorf("Expected PostgreSQL port 5432, got %d", updatedDB.Status.Port)
	}
	if updatedDB.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedDB.Status.Phase)
	}
}

func TestDatabaseReconciler_RedisPort(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "redis",
			Version: "7.2",
			Storage: resource.MustParse("5Gi"),
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("test-token")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(client, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cache", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedDB := &zenithv1.Database{}
	client.Get(context.Background(), types.NamespacedName{Name: "cache", Namespace: "zenith-test"}, updatedDB)

	if updatedDB.Status.Port != 6379 {
		t.Errorf("Expected Redis port 6379, got %d", updatedDB.Status.Port)
	}
}

func TestStorageBucketReconciler(t *testing.T) {
	scheme := setupScheme()

	sb := &zenithv1.StorageBucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-bucket",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.StorageBucketSpec{
			Access: "private",
			Region: "fsn1",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, sb).
		WithStatusSubresource(sb).
		Build()

	hetznerClient := hetzner.NewClient("test-token")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewStorageBucketReconciler(client, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.StorageBucket{}
	client.Get(context.Background(), types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"}, updated)

	if updated.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updated.Status.Phase)
	}
}

func TestDomainReconciler(t *testing.T) {
	scheme := setupScheme()

	domain := &zenithv1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-domain",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DomainSpec{
			Domain: "app.example.com",
			AppRef: "web-app",
			SSL:    &zenithv1.SSLConfig{Enabled: true},
			DNS:    &zenithv1.DNSConfig{AutoConfigure: true},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, domain).
		WithStatusSubresource(domain).
		Build()

	hetznerClient := hetzner.NewClient("test-token")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDomainReconciler(client, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-domain", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.Domain{}
	client.Get(context.Background(), types.NamespacedName{Name: "my-domain", Namespace: "zenith-test"}, updated)

	if updated.Status.Phase != "Active" {
		t.Errorf("Expected phase 'Active', got '%s'", updated.Status.Phase)
	}
	if !updated.Status.SSLReady {
		t.Error("Expected SSL to be ready")
	}
}

func TestAuthRealmReconciler(t *testing.T) {
	scheme := setupScheme()

	realm := &zenithv1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-realm",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AuthRealmSpec{
			DisplayName: "My Realm",
			Clients: []zenithv1.AuthClient{
				{Name: "web-app", Type: "public"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, realm).
		WithStatusSubresource(realm).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(client, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-realm", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.AuthRealm{}
	client.Get(context.Background(), types.NamespacedName{Name: "my-realm", Namespace: "zenith-test"}, updated)

	if updated.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updated.Status.Phase)
	}
	if updated.Status.ClientCount != 1 {
		t.Errorf("Expected 1 client, got %d", updated.Status.ClientCount)
	}
}

func TestGatewayRouteReconciler(t *testing.T) {
	scheme := setupScheme()

	route := &zenithv1.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-route",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GatewayRouteSpec{
			Path:    "/api/v1/users",
			Methods: []string{"GET", "POST"},
			Service: zenithv1.ServiceRef{Name: "user-svc", Port: 8080},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(client, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "api-route", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.GatewayRoute{}
	client.Get(context.Background(), types.NamespacedName{Name: "api-route", Namespace: "zenith-test"}, updated)

	if updated.Status.Phase != "Active" {
		t.Errorf("Expected phase 'Active', got '%s'", updated.Status.Phase)
	}
}

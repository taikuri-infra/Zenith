package v1alpha1

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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

func TestGatewayRouteDeepCopy(t *testing.T) {
	route := &GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "my-route"},
		Spec: GatewayRouteSpec{
			Path:    "/api/v1/users",
			Methods: []string{"GET", "POST"},
			Service: ServiceRef{Name: "user-service", Port: 8080},
			Auth:    &RouteAuth{Enabled: true, Scopes: []string{"read", "write"}},
		},
	}

	copied := route.DeepCopy()
	if copied.Spec.Path != "/api/v1/users" {
		t.Errorf("DeepCopy failed: Path mismatch")
	}

	copied.Spec.Methods = append(copied.Spec.Methods, "DELETE")
	if len(route.Spec.Methods) != 2 {
		t.Error("DeepCopy is not independent for Methods slice")
	}
}

func TestGatewayRouteImplementsRuntimeObject(t *testing.T) {
	var _ runtime.Object = &GatewayRoute{}
	var _ runtime.Object = &GatewayRouteList{}
}

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
}

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
		&GatewayRoute{}, &GatewayRouteList{},
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

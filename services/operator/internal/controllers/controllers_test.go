package controllers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
	_ = clientgoscheme.AddToScheme(s)
	_ = zenithv1.AddToScheme(s)
	return s
}

// ============================================================================
// Project Controller Tests
// ============================================================================

func TestProjectReconciler_CreateNamespaceAndQuota(t *testing.T) {
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

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	// First reconcile adds finalizer
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue")
	}

	// Second reconcile creates resources
	result, err = reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-project"},
	})
	if err != nil {
		t.Fatalf("Second reconcile failed: %v", err)
	}

	// Verify finalizer was added
	updatedProject := &zenithv1.Project{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-project"}, updatedProject); err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}
	if len(updatedProject.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify namespace was created
	ns := &corev1.Namespace{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "zenith-test-project"}, ns); err != nil {
		t.Fatalf("Expected namespace to be created: %v", err)
	}
	if ns.Labels["zenith.dev/project"] != "test-project" {
		t.Errorf("Expected project label, got '%s'", ns.Labels["zenith.dev/project"])
	}
	if ns.Labels["zenith.dev/plan"] != "free" {
		t.Errorf("Expected plan label 'free', got '%s'", ns.Labels["zenith.dev/plan"])
	}

	// Verify ResourceQuota was created
	rq := &corev1.ResourceQuota{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "test-project-quota",
		Namespace: "zenith-test-project",
	}, rq); err != nil {
		t.Fatalf("Expected ResourceQuota to be created: %v", err)
	}
	// Free plan: 2 CPU limit
	cpuLimit := rq.Spec.Hard[corev1.ResourceLimitsCPU]
	if cpuLimit.Cmp(resource.MustParse("2")) != 0 {
		t.Errorf("Expected CPU limit '2', got '%s'", cpuLimit.String())
	}

	// Verify LimitRange was created
	lr := &corev1.LimitRange{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "test-project-limits",
		Namespace: "zenith-test-project",
	}, lr); err != nil {
		t.Fatalf("Expected LimitRange to be created: %v", err)
	}
	if len(lr.Spec.Limits) == 0 {
		t.Error("Expected LimitRange to have limits")
	}
}

func TestProjectReconciler_ProPlanQuotas(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pro-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Pro Project",
			Owner:       "pro@test.com",
			Plan:        "pro",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	// Two reconciles: first adds finalizer, second creates resources
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "pro-project"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "pro-project"},
	})

	rq := &corev1.ResourceQuota{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "pro-project-quota",
		Namespace: "zenith-pro-project",
	}, rq); err != nil {
		t.Fatalf("Expected ResourceQuota: %v", err)
	}

	cpuLimit := rq.Spec.Hard[corev1.ResourceLimitsCPU]
	if cpuLimit.Cmp(resource.MustParse("8")) != 0 {
		t.Errorf("Expected pro CPU limit '8', got '%s'", cpuLimit.String())
	}
	memLimit := rq.Spec.Hard[corev1.ResourceLimitsMemory]
	if memLimit.Cmp(resource.MustParse("16Gi")) != 0 {
		t.Errorf("Expected pro memory limit '16Gi', got '%s'", memLimit.String())
	}
}

func TestProjectReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

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

// ============================================================================
// App Controller Tests
// ============================================================================

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

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	// First reconcile adds finalizer
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "web-app", Namespace: "zenith-test"},
	})

	// Second reconcile creates resources
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "web-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue")
	}

	// Verify finalizer
	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "web-app", Namespace: "zenith-test"}, updatedApp)
	if len(updatedApp.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify deployment
	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "web-app", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected deployment to be created: %v", err)
	}
	if *dep.Spec.Replicas != 2 {
		t.Errorf("Expected 2 replicas, got %d", *dep.Spec.Replicas)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "nginx:latest" {
		t.Errorf("Expected image 'nginx:latest', got '%s'", dep.Spec.Template.Spec.Containers[0].Image)
	}
	// Verify rolling update strategy
	if dep.Spec.Strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
		t.Errorf("Expected RollingUpdate strategy, got %s", dep.Spec.Strategy.Type)
	}
	// Verify owner reference
	if len(dep.OwnerReferences) == 0 {
		t.Error("Expected owner reference on deployment")
	}

	// Verify service
	svc := &corev1.Service{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "web-app", Namespace: "zenith-test"}, svc); err != nil {
		t.Fatalf("Expected service to be created: %v", err)
	}
	if len(svc.OwnerReferences) == 0 {
		t.Error("Expected owner reference on service")
	}
}

func TestAppReconciler_WithDomainCreatesIngress(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
			Domain:   "app.example.com",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "web-app", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "web-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify ingress was created
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "web-app", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected ingress to be created: %v", err)
	}
	if ingress.Spec.Rules[0].Host != "app.example.com" {
		t.Errorf("Expected host 'app.example.com', got '%s'", ingress.Spec.Rules[0].Host)
	}
	if len(ingress.Spec.TLS) == 0 {
		t.Error("Expected TLS section on ingress")
	}
	if ingress.Annotations["cert-manager.io/cluster-issuer"] != "letsencrypt-prod" {
		t.Error("Expected cert-manager annotation on ingress")
	}

	// Verify status URL
	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "web-app", Namespace: "zenith-test"}, updatedApp)
	if updatedApp.Status.URL != "https://app.example.com" {
		t.Errorf("Expected status URL 'https://app.example.com', got '%s'", updatedApp.Status.URL)
	}
}

func TestAppReconciler_WithAutoScaleCreatesHPA(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scalable-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
			AutoScale: &zenithv1.AutoScale{
				MinReplicas:      2,
				MaxReplicas:      10,
				TargetCPUPercent: 80,
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "scalable-app", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "scalable-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify HPA was created
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "scalable-app", Namespace: "zenith-test"}, hpa); err != nil {
		t.Fatalf("Expected HPA to be created: %v", err)
	}
	if hpa.Spec.MaxReplicas != 10 {
		t.Errorf("Expected max replicas 10, got %d", hpa.Spec.MaxReplicas)
	}
	if *hpa.Spec.MinReplicas != 2 {
		t.Errorf("Expected min replicas 2, got %d", *hpa.Spec.MinReplicas)
	}
}

func TestAppReconciler_WithResourceLimits(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "limited-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
			Resources: &zenithv1.AppResources{
				CPU:    resource.MustParse("500m"),
				Memory: resource.MustParse("256Mi"),
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "limited-app", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "limited-app", Namespace: "zenith-test"},
	})

	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "limited-app", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected deployment: %v", err)
	}

	container := dep.Spec.Template.Spec.Containers[0]
	cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
	if cpuLimit.Cmp(resource.MustParse("500m")) != 0 {
		t.Errorf("Expected CPU limit '500m', got '%s'", cpuLimit.String())
	}
}

func TestAppReconciler_WithHealthCheck(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "myapp:1.0",
			Replicas: &replicas,
			Port:     8080,
			HealthCheck: &zenithv1.HealthCheck{
				Path:            "/healthz",
				Port:            9090,
				IntervalSeconds: 15,
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "health-app", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "health-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "health-app", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected deployment: %v", err)
	}

	container := dep.Spec.Template.Spec.Containers[0]

	// Verify readiness probe
	if container.ReadinessProbe == nil {
		t.Fatal("Expected readiness probe to be set")
	}
	if container.ReadinessProbe.HTTPGet.Path != "/healthz" {
		t.Errorf("Expected readiness probe path '/healthz', got '%s'", container.ReadinessProbe.HTTPGet.Path)
	}
	if container.ReadinessProbe.HTTPGet.Port.IntValue() != 9090 {
		t.Errorf("Expected readiness probe port 9090, got %d", container.ReadinessProbe.HTTPGet.Port.IntValue())
	}
	if container.ReadinessProbe.PeriodSeconds != 15 {
		t.Errorf("Expected readiness probe interval 15, got %d", container.ReadinessProbe.PeriodSeconds)
	}

	// Verify liveness probe
	if container.LivenessProbe == nil {
		t.Fatal("Expected liveness probe to be set")
	}
	if container.LivenessProbe.HTTPGet.Path != "/healthz" {
		t.Errorf("Expected liveness probe path '/healthz', got '%s'", container.LivenessProbe.HTTPGet.Path)
	}
	if container.LivenessProbe.HTTPGet.Port.IntValue() != 9090 {
		t.Errorf("Expected liveness probe port 9090, got %d", container.LivenessProbe.HTTPGet.Port.IntValue())
	}
	if container.LivenessProbe.InitialDelaySeconds != 15 {
		t.Errorf("Expected liveness initial delay 15, got %d", container.LivenessProbe.InitialDelaySeconds)
	}
}

func TestAppReconciler_WithoutDomainNoIngress(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-domain-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
			// No Domain set
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-domain-app", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-domain-app", Namespace: "zenith-test"},
	})

	// Verify deployment exists
	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "no-domain-app", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected deployment: %v", err)
	}

	// Verify NO ingress was created
	ingress := &networkingv1.Ingress{}
	err := cl.Get(context.Background(), types.NamespacedName{Name: "no-domain-app", Namespace: "zenith-test"}, ingress)
	if err == nil {
		t.Error("Expected no ingress to be created when domain is not set")
	}

	// Verify status URL is empty
	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "no-domain-app", Namespace: "zenith-test"}, updatedApp)
	if updatedApp.Status.URL != "" {
		t.Errorf("Expected empty status URL without domain, got '%s'", updatedApp.Status.URL)
	}
	// InternalURL should still be set
	if updatedApp.Status.InternalURL == "" {
		t.Error("Expected internal URL to be set")
	}
}

func TestAppReconciler_WithEnvVars(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "env-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "myapp:latest",
			Replicas: &replicas,
			Port:     3000,
			Env: []corev1.EnvVar{
				{Name: "NODE_ENV", Value: "production"},
				{Name: "API_KEY", Value: "secret-123"},
				{Name: "DB_URL", Value: "postgres://localhost:5432/mydb"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "env-app", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "env-app", Namespace: "zenith-test"},
	})

	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "env-app", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected deployment: %v", err)
	}

	container := dep.Spec.Template.Spec.Containers[0]
	if len(container.Env) != 3 {
		t.Fatalf("Expected 3 env vars, got %d", len(container.Env))
	}

	envMap := make(map[string]string)
	for _, e := range container.Env {
		envMap[e.Name] = e.Value
	}
	if envMap["NODE_ENV"] != "production" {
		t.Errorf("Expected NODE_ENV='production', got '%s'", envMap["NODE_ENV"])
	}
	if envMap["API_KEY"] != "secret-123" {
		t.Errorf("Expected API_KEY='secret-123', got '%s'", envMap["API_KEY"])
	}
	if envMap["DB_URL"] != "postgres://localhost:5432/mydb" {
		t.Errorf("Expected DB_URL to be set correctly")
	}
}

func TestAppReconciler_UpdateImageChange(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "update-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image:    "myapp:v1",
			Replicas: &replicas,
			Port:     8080,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	// First: add finalizer, then create resources
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "update-app", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "update-app", Namespace: "zenith-test"},
	})

	// Verify initial image
	dep := &appsv1.Deployment{}
	cl.Get(context.Background(), types.NamespacedName{Name: "update-app", Namespace: "zenith-test"}, dep)
	if dep.Spec.Template.Spec.Containers[0].Image != "myapp:v1" {
		t.Errorf("Expected initial image 'myapp:v1', got '%s'", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Update the app image
	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "update-app", Namespace: "zenith-test"}, updatedApp)
	updatedApp.Spec.Image = "myapp:v2"
	cl.Update(context.Background(), updatedApp)

	// Reconcile again
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "update-app", Namespace: "zenith-test"},
	})

	// Verify updated image
	cl.Get(context.Background(), types.NamespacedName{Name: "update-app", Namespace: "zenith-test"}, dep)
	if dep.Spec.Template.Spec.Containers[0].Image != "myapp:v2" {
		t.Errorf("Expected updated image 'myapp:v2', got '%s'", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify status reflects new image
	cl.Get(context.Background(), types.NamespacedName{Name: "update-app", Namespace: "zenith-test"}, updatedApp)
	if updatedApp.Status.CurrentImage != "myapp:v2" {
		t.Errorf("Expected status current image 'myapp:v2', got '%s'", updatedApp.Status.CurrentImage)
	}
}

func TestAppReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

// ============================================================================
// Database Controller Tests
// ============================================================================

func TestDatabaseReconciler_CreatesStatefulSetAndService(t *testing.T) {
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

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	// Use unconfigured Hetzner client (no token)
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	// First reconcile adds finalizer
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-db", Namespace: "zenith-test"},
	})

	// Second reconcile creates resources
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-db", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer
	updatedDB := &zenithv1.Database{}
	cl.Get(context.Background(), types.NamespacedName{Name: "test-db", Namespace: "zenith-test"}, updatedDB)
	if len(updatedDB.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify StatefulSet was created
	sts := &appsv1.StatefulSet{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-db", Namespace: "zenith-test"}, sts); err != nil {
		t.Fatalf("Expected StatefulSet to be created: %v", err)
	}
	if sts.Spec.Template.Spec.Containers[0].Image != "postgres:16" {
		t.Errorf("Expected image 'postgres:16', got '%s'", sts.Spec.Template.Spec.Containers[0].Image)
	}
	// Check volume mount
	if len(sts.Spec.Template.Spec.Containers[0].VolumeMounts) == 0 {
		t.Error("Expected volume mount on container")
	}
	if sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != "/var/lib/postgresql/data" {
		t.Errorf("Expected mount path '/var/lib/postgresql/data', got '%s'", sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	}
	if len(sts.OwnerReferences) == 0 {
		t.Error("Expected owner reference on StatefulSet")
	}

	// Verify headless Service
	svc := &corev1.Service{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-db", Namespace: "zenith-test"}, svc); err != nil {
		t.Fatalf("Expected headless Service to be created: %v", err)
	}
	if svc.Spec.ClusterIP != corev1.ClusterIPNone {
		t.Error("Expected headless service (ClusterIP=None)")
	}

	// Verify Secret with credentials
	secret := &corev1.Secret{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-db-conn", Namespace: "zenith-test"}, secret); err != nil {
		t.Fatalf("Expected connection Secret to be created: %v", err)
	}
	if string(secret.Data["username"]) != "postgres" {
		t.Errorf("Expected username 'postgres', got '%s'", string(secret.Data["username"]))
	}
	if len(secret.Data["password"]) == 0 {
		t.Error("Expected password to be generated")
	}
	if string(secret.Data["port"]) != "5432" {
		t.Errorf("Expected port '5432', got '%s'", string(secret.Data["port"]))
	}

	// Verify PVC was created
	pvc := &corev1.PersistentVolumeClaim{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-db-data", Namespace: "zenith-test"}, pvc); err != nil {
		t.Fatalf("Expected PVC to be created: %v", err)
	}

	// Verify status
	if updatedDB.Status.Port != 5432 {
		t.Errorf("Expected PostgreSQL port 5432, got %d", updatedDB.Status.Port)
	}
	if updatedDB.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedDB.Status.Phase)
	}
	if updatedDB.Status.SecretName != "test-db-conn" {
		t.Errorf("Expected secretName 'test-db-conn', got '%s'", updatedDB.Status.SecretName)
	}
}

func TestDatabaseReconciler_RedisEngine(t *testing.T) {
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

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cache", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cache", Namespace: "zenith-test"},
	})

	updatedDB := &zenithv1.Database{}
	cl.Get(context.Background(), types.NamespacedName{Name: "cache", Namespace: "zenith-test"}, updatedDB)

	if updatedDB.Status.Port != 6379 {
		t.Errorf("Expected Redis port 6379, got %d", updatedDB.Status.Port)
	}

	// Verify StatefulSet uses Redis image
	sts := &appsv1.StatefulSet{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "cache", Namespace: "zenith-test"}, sts); err != nil {
		t.Fatalf("Expected StatefulSet: %v", err)
	}
	if sts.Spec.Template.Spec.Containers[0].Image != "redis:7.2" {
		t.Errorf("Expected image 'redis:7.2', got '%s'", sts.Spec.Template.Spec.Containers[0].Image)
	}
	if sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != "/data" {
		t.Errorf("Expected mount path '/data', got '%s'", sts.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	}
}

func TestDatabaseReconciler_MySQLEngine(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mydb",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "mysql",
			Version: "8.0",
			Storage: resource.MustParse("10Gi"),
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "mydb", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "mydb", Namespace: "zenith-test"},
	})

	updatedDB := &zenithv1.Database{}
	cl.Get(context.Background(), types.NamespacedName{Name: "mydb", Namespace: "zenith-test"}, updatedDB)

	if updatedDB.Status.Port != 3306 {
		t.Errorf("Expected MySQL port 3306, got %d", updatedDB.Status.Port)
	}

	sts := &appsv1.StatefulSet{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "mydb", Namespace: "zenith-test"}, sts); err != nil {
		t.Fatalf("Expected StatefulSet: %v", err)
	}
	if sts.Spec.Template.Spec.Containers[0].Image != "mysql:8.0" {
		t.Errorf("Expected image 'mysql:8.0', got '%s'", sts.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestDatabaseReconciler_BackupCronJobEnabled(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backup-db",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "postgresql",
			Version: "16",
			Storage: resource.MustParse("20Gi"),
			Backup: &zenithv1.BackupConfig{
				Enabled:       true,
				Schedule:      "0 3 * * *",
				RetentionDays: 14,
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "backup-db", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "backup-db", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify CronJob was created
	cronJob := &batchv1.CronJob{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "backup-db-backup", Namespace: "zenith-test"}, cronJob); err != nil {
		t.Fatalf("Expected backup CronJob to be created: %v", err)
	}
	if cronJob.Spec.Schedule != "0 3 * * *" {
		t.Errorf("Expected schedule '0 3 * * *', got '%s'", cronJob.Spec.Schedule)
	}
	// Verify the backup container uses the correct image
	container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
	if container.Image != "postgres:16" {
		t.Errorf("Expected backup image 'postgres:16', got '%s'", container.Image)
	}
	// Verify env vars reference the secret
	foundDBPassword := false
	for _, env := range container.Env {
		if env.Name == "DB_PASSWORD" && env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
			if env.ValueFrom.SecretKeyRef.Name == "backup-db-conn" {
				foundDBPassword = true
			}
		}
	}
	if !foundDBPassword {
		t.Error("Expected DB_PASSWORD env var referencing connection secret")
	}
	// Verify owner reference
	if len(cronJob.OwnerReferences) == 0 {
		t.Error("Expected owner reference on CronJob")
	}
}

func TestDatabaseReconciler_BackupDisabledNoCronJob(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nobackup-db",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "postgresql",
			Version: "16",
			Storage: resource.MustParse("10Gi"),
			Backup: &zenithv1.BackupConfig{
				Enabled: false,
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nobackup-db", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nobackup-db", Namespace: "zenith-test"},
	})

	// Verify NO CronJob was created
	cronJob := &batchv1.CronJob{}
	err := cl.Get(context.Background(), types.NamespacedName{Name: "nobackup-db-backup", Namespace: "zenith-test"}, cronJob)
	if err == nil {
		t.Error("Expected no CronJob when backup is disabled")
	}
}

func TestDatabaseReconciler_PasswordPreservation(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pw-db",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "postgresql",
			Version: "16",
			Storage: resource.MustParse("10Gi"),
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	// First reconcile: finalizer
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "pw-db", Namespace: "zenith-test"},
	})
	// Second reconcile: creates resources
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "pw-db", Namespace: "zenith-test"},
	})

	// Read the generated password
	secret := &corev1.Secret{}
	cl.Get(context.Background(), types.NamespacedName{Name: "pw-db-conn", Namespace: "zenith-test"}, secret)
	originalPassword := string(secret.Data["password"])
	if originalPassword == "" {
		t.Fatal("Expected password to be generated")
	}

	// Reconcile again (simulating an update)
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "pw-db", Namespace: "zenith-test"},
	})

	// Read password again - it should be preserved
	cl.Get(context.Background(), types.NamespacedName{Name: "pw-db-conn", Namespace: "zenith-test"}, secret)
	newPassword := string(secret.Data["password"])
	if newPassword != originalPassword {
		t.Errorf("Expected password to be preserved across reconciles, got different passwords")
	}
}

func TestDatabaseReconciler_ConnectionStringFormat(t *testing.T) {
	tests := []struct {
		engine       string
		version      string
		expectedPort int32
		expectedUser string
		connPrefix   string
	}{
		{"postgresql", "16", 5432, "postgres", "postgresql://postgres:"},
		{"mysql", "8.0", 3306, "root", "mysql://root:"},
		{"mongodb", "7", 27017, "admin", "mongodb://admin:"},
		{"redis", "7.2", 6379, "", "redis://:"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			scheme := setupScheme()

			db := &zenithv1.Database{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "conn-" + tt.engine,
					Namespace: "zenith-test",
				},
				Spec: zenithv1.DatabaseSpec{
					Engine:  tt.engine,
					Version: tt.version,
					Storage: resource.MustParse("5Gi"),
				},
			}

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(ns, db).
				WithStatusSubresource(db).
				Build()

			hetznerClient := hetzner.NewClient("")
			recorder := record.NewFakeRecorder(10)
			reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

			reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "conn-" + tt.engine, Namespace: "zenith-test"},
			})
			reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "conn-" + tt.engine, Namespace: "zenith-test"},
			})

			// Read connection secret
			secret := &corev1.Secret{}
			if err := cl.Get(context.Background(), types.NamespacedName{
				Name:      "conn-" + tt.engine + "-conn",
				Namespace: "zenith-test",
			}, secret); err != nil {
				t.Fatalf("Expected connection secret: %v", err)
			}

			connString := string(secret.Data["connection-string"])
			if !strings.HasPrefix(connString, tt.connPrefix) {
				t.Errorf("Expected connection string to start with '%s', got '%s'", tt.connPrefix, connString)
			}

			if string(secret.Data["username"]) != tt.expectedUser {
				t.Errorf("Expected username '%s', got '%s'", tt.expectedUser, string(secret.Data["username"]))
			}
		})
	}
}

func TestDatabaseReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

// ============================================================================
// Domain Controller Tests
// ============================================================================

func TestDomainReconciler_CreatesIngressWithTLS(t *testing.T) {
	scheme := setupScheme()

	// Create the referenced app first
	replicas := int32(1)
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

	domain := &zenithv1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-domain",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DomainSpec{
			Domain: "app.example.com",
			AppRef: "web-app",
			SSL:    &zenithv1.SSLConfig{Enabled: true, Issuer: "letsencrypt-prod"},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app, domain).
		WithStatusSubresource(domain, app).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDomainReconciler(cl, scheme, recorder, hetznerClient)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-domain", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-domain", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer
	updatedDomain := &zenithv1.Domain{}
	cl.Get(context.Background(), types.NamespacedName{Name: "my-domain", Namespace: "zenith-test"}, updatedDomain)
	if len(updatedDomain.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify Ingress was created
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "domain-my-domain", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress to be created: %v", err)
	}
	if ingress.Spec.Rules[0].Host != "app.example.com" {
		t.Errorf("Expected host 'app.example.com', got '%s'", ingress.Spec.Rules[0].Host)
	}
	if len(ingress.Spec.TLS) == 0 {
		t.Error("Expected TLS section")
	}
	if ingress.Spec.TLS[0].SecretName != "my-domain-tls" {
		t.Errorf("Expected TLS secret 'my-domain-tls', got '%s'", ingress.Spec.TLS[0].SecretName)
	}
	// Verify backend points to the app service
	backend := ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service
	if backend.Name != "web-app" {
		t.Errorf("Expected backend service 'web-app', got '%s'", backend.Name)
	}
	if backend.Port.Number != 8080 {
		t.Errorf("Expected backend port 8080, got %d", backend.Port.Number)
	}

	// Verify status
	if updatedDomain.Status.Phase != "Active" {
		t.Errorf("Expected phase 'Active', got '%s'", updatedDomain.Status.Phase)
	}
}

func TestDomainReconciler_WithoutSSLNoTLS(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
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

	domain := &zenithv1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-ssl-domain",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DomainSpec{
			Domain: "app.example.com",
			AppRef: "web-app",
			SSL:    &zenithv1.SSLConfig{Enabled: false},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app, domain).
		WithStatusSubresource(domain, app).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDomainReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-ssl-domain", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-ssl-domain", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Ingress was created
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "domain-no-ssl-domain", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress to be created: %v", err)
	}

	// Verify NO TLS section when SSL disabled
	if len(ingress.Spec.TLS) != 0 {
		t.Error("Expected no TLS section when SSL is disabled")
	}

	// Verify no cert-manager annotation
	if _, ok := ingress.Annotations["cert-manager.io/cluster-issuer"]; ok {
		t.Error("Expected no cert-manager annotation when SSL is disabled")
	}
}

func TestDomainReconciler_WithoutDNSAutoConfigure(t *testing.T) {
	scheme := setupScheme()

	replicas := int32(1)
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

	domain := &zenithv1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-dns-domain",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DomainSpec{
			Domain: "manual.example.com",
			AppRef: "web-app",
			DNS:    &zenithv1.DNSConfig{AutoConfigure: false},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app, domain).
		WithStatusSubresource(domain, app).
		Build()

	hetznerClient := hetzner.NewClient("test-token")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDomainReconciler(cl, scheme, recorder, hetznerClient)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-dns-domain", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-dns-domain", Namespace: "zenith-test"},
	})

	// Verify domain status - DNSConfigured should be false
	updatedDomain := &zenithv1.Domain{}
	cl.Get(context.Background(), types.NamespacedName{Name: "no-dns-domain", Namespace: "zenith-test"}, updatedDomain)
	if updatedDomain.Status.DNSConfigured {
		t.Error("Expected DNS to NOT be configured when AutoConfigure is false")
	}

	// Verify no DNS record ID annotation
	if _, ok := updatedDomain.Annotations[domainDNSRecordIDAnno]; ok {
		t.Error("Expected no DNS record ID annotation")
	}
}

func TestDomainReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDomainReconciler(cl, scheme, recorder, hetznerClient)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

// ============================================================================
// StorageBucket Controller Tests
// ============================================================================

func TestStorageBucketReconciler_CreatesSecretWithCredentials(t *testing.T) {
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

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, sb).
		WithStatusSubresource(sb).
		Build()

	// Use unconfigured Hetzner to skip bucket creation
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewStorageBucketReconciler(cl, scheme, recorder, hetznerClient)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer
	updated := &zenithv1.StorageBucket{}
	cl.Get(context.Background(), types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"}, updated)
	if len(updated.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify Secret was created
	secret := &corev1.Secret{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "my-bucket-s3-credentials", Namespace: "zenith-test"}, secret); err != nil {
		t.Fatalf("Expected S3 credentials secret to be created: %v", err)
	}
	if len(secret.Data["access-key"]) == 0 {
		t.Error("Expected access-key to be generated")
	}
	if len(secret.Data["secret-key"]) == 0 {
		t.Error("Expected secret-key to be generated")
	}
	if string(secret.Data["region"]) != "fsn1" {
		t.Errorf("Expected region 'fsn1', got '%s'", string(secret.Data["region"]))
	}
	if len(secret.Data["endpoint"]) == 0 {
		t.Error("Expected endpoint in secret")
	}
	if len(secret.Data["bucket"]) == 0 {
		t.Error("Expected bucket name in secret")
	}

	// Verify status
	if updated.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updated.Status.Phase)
	}
	if updated.Status.SecretName != "my-bucket-s3-credentials" {
		t.Errorf("Expected secretName, got '%s'", updated.Status.SecretName)
	}
}

func TestStorageBucketReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	sb := &zenithv1.StorageBucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-bucket",
			Namespace:         "zenith-test",
			DeletionTimestamp: &now,
			Finalizers:        []string{storageBucketFinalizer},
		},
		Spec: zenithv1.StorageBucketSpec{
			Access: "private",
			Region: "fsn1",
		},
		Status: zenithv1.StorageBucketStatus{
			BucketName: "zenith-zenith-test-deleting-bucket",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, sb).
		WithStatusSubresource(sb).
		Build()

	// Use unconfigured Hetzner to skip actual bucket deletion
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewStorageBucketReconciler(cl, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-bucket", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.StorageBucket{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-bucket", Namespace: "zenith-test"}, updated)

	for _, f := range updated.Finalizers {
		if f == storageBucketFinalizer {
			t.Error("Expected finalizer to be removed during deletion")
		}
	}
}

func TestStorageBucketReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewStorageBucketReconciler(cl, scheme, recorder, hetznerClient)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

func TestStorageBucketReconciler_CredentialPreservation(t *testing.T) {
	scheme := setupScheme()

	sb := &zenithv1.StorageBucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cred-bucket",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.StorageBucketSpec{
			Access: "private",
			Region: "fsn1",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, sb).
		WithStatusSubresource(sb).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewStorageBucketReconciler(cl, scheme, recorder, hetznerClient)

	// First reconcile cycle
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cred-bucket", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cred-bucket", Namespace: "zenith-test"},
	})

	// Read original credentials
	secret := &corev1.Secret{}
	cl.Get(context.Background(), types.NamespacedName{Name: "cred-bucket-s3-credentials", Namespace: "zenith-test"}, secret)
	originalAccessKey := string(secret.Data["access-key"])
	originalSecretKey := string(secret.Data["secret-key"])

	// Reconcile again
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cred-bucket", Namespace: "zenith-test"},
	})

	// Verify credentials are preserved
	cl.Get(context.Background(), types.NamespacedName{Name: "cred-bucket-s3-credentials", Namespace: "zenith-test"}, secret)
	if string(secret.Data["access-key"]) != originalAccessKey {
		t.Error("Expected access-key to be preserved across reconciles")
	}
	if string(secret.Data["secret-key"]) != originalSecretKey {
		t.Error("Expected secret-key to be preserved across reconciles")
	}
}

// ============================================================================
// AuthRealm Controller Tests
// ============================================================================

func TestAuthRealmReconciler_CreatesConfigMapDeploymentService(t *testing.T) {
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
				{Name: "api", Type: "confidential"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, realm).
		WithStatusSubresource(realm).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-realm", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-realm", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer
	updatedRealm := &zenithv1.AuthRealm{}
	cl.Get(context.Background(), types.NamespacedName{Name: "my-realm", Namespace: "zenith-test"}, updatedRealm)
	if len(updatedRealm.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-realm-my-realm", Namespace: "zenith-test"}, cm); err != nil {
		t.Fatalf("Expected ConfigMap to be created: %v", err)
	}
	if cm.Data["realm.json"] == "" {
		t.Error("Expected realm.json in ConfigMap data")
	}

	// Verify Deployment was created
	dep := &appsv1.Deployment{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-my-realm", Namespace: "zenith-test"}, dep); err != nil {
		t.Fatalf("Expected Deployment to be created: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "zenith-auth:latest" {
		t.Errorf("Expected image 'zenith-auth:latest', got '%s'", dep.Spec.Template.Spec.Containers[0].Image)
	}
	// Verify env vars
	foundRealmName := false
	for _, env := range dep.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "REALM_NAME" && env.Value == "my-realm" {
			foundRealmName = true
		}
	}
	if !foundRealmName {
		t.Error("Expected REALM_NAME env var")
	}

	// Verify Service was created
	svc := &corev1.Service{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-my-realm", Namespace: "zenith-test"}, svc); err != nil {
		t.Fatalf("Expected Service to be created: %v", err)
	}

	// Verify Ingress was created
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-my-realm", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress to be created: %v", err)
	}

	// Verify status
	if updatedRealm.Status.Phase != "Ready" {
		t.Errorf("Expected phase 'Ready', got '%s'", updatedRealm.Status.Phase)
	}
	if updatedRealm.Status.ClientCount != 2 {
		t.Errorf("Expected 2 clients, got %d", updatedRealm.Status.ClientCount)
	}
	if updatedRealm.Status.Endpoint == "" {
		t.Error("Expected endpoint to be set")
	}
}

func TestAuthRealmReconciler_WithProviders(t *testing.T) {
	scheme := setupScheme()

	// Create the provider secret first
	providerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "github-secret",
			Namespace: "zenith-test",
		},
		Data: map[string][]byte{
			"clientSecret": []byte("gh-secret-abc123"),
		},
	}

	realm := &zenithv1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "provider-realm",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AuthRealmSpec{
			DisplayName: "Provider Realm",
			Providers: []zenithv1.IdentityProvider{
				{
					Name:     "github",
					Type:     "github",
					ClientID: "gh-client-id",
					ClientSecretRef: &zenithv1.SecretKeyRef{
						Name: "github-secret",
						Key:  "clientSecret",
					},
					Enabled: true,
				},
				{
					Name:     "google",
					Type:     "google",
					ClientID: "google-client-id",
					Enabled:  true,
				},
			},
			Clients: []zenithv1.AuthClient{
				{Name: "web", Type: "public"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, realm, providerSecret).
		WithStatusSubresource(realm).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "provider-realm", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "provider-realm", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify provider secret was created with extracted secrets
	secret := &corev1.Secret{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-realm-provider-realm-secrets", Namespace: "zenith-test"}, secret); err != nil {
		t.Fatalf("Expected auth secrets: %v", err)
	}
	if string(secret.Data["provider-github-client-secret"]) != "gh-secret-abc123" {
		t.Errorf("Expected github client secret to be extracted, got '%s'", string(secret.Data["provider-github-client-secret"]))
	}
}

func TestAuthRealmReconciler_IngressPath(t *testing.T) {
	scheme := setupScheme()

	realm := &zenithv1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "path-realm",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AuthRealmSpec{
			DisplayName: "Path Realm",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, realm).
		WithStatusSubresource(realm).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "path-realm", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "path-realm", Namespace: "zenith-test"},
	})

	// Verify Ingress path contains realm name
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-path-realm", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress: %v", err)
	}

	expectedPathPrefix := "/realms/path-realm"
	actualPath := ingress.Spec.Rules[0].HTTP.Paths[0].Path
	if !strings.HasPrefix(actualPath, expectedPathPrefix) {
		t.Errorf("Expected ingress path to start with '%s', got '%s'", expectedPathPrefix, actualPath)
	}

	// Verify rewrite annotation
	if ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] != "/$2" {
		t.Error("Expected rewrite-target annotation")
	}
}

func TestAuthRealmReconciler_ConfigMapValidJSON(t *testing.T) {
	scheme := setupScheme()

	realm := &zenithv1.AuthRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "json-realm",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AuthRealmSpec{
			DisplayName: "JSON Realm",
			Clients: []zenithv1.AuthClient{
				{Name: "web-app", Type: "public", RedirectURIs: []string{"https://app.example.com/callback"}, Scopes: []string{"openid", "profile"}},
			},
			Settings: &zenithv1.RealmSettings{
				MFARequired:    true,
				SessionTimeout: "12h",
				PasswordPolicy: &zenithv1.PasswordPolicy{
					MinLength:        12,
					RequireUppercase: true,
					RequireNumbers:   true,
					RequireSpecial:   true,
				},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, realm).
		WithStatusSubresource(realm).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "json-realm", Namespace: "zenith-test"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "json-realm", Namespace: "zenith-test"},
	})

	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "auth-realm-json-realm", Namespace: "zenith-test"}, cm); err != nil {
		t.Fatalf("Expected ConfigMap: %v", err)
	}

	realmJSON := cm.Data["realm.json"]
	if realmJSON == "" {
		t.Fatal("Expected realm.json to be non-empty")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(realmJSON), &parsed); err != nil {
		t.Fatalf("Expected valid JSON in realm.json: %v", err)
	}

	// Verify content
	if parsed["name"] != "json-realm" {
		t.Errorf("Expected name 'json-realm', got '%v'", parsed["name"])
	}
	if parsed["displayName"] != "JSON Realm" {
		t.Errorf("Expected displayName 'JSON Realm', got '%v'", parsed["displayName"])
	}

	// Verify settings are present
	settings, ok := parsed["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected settings in JSON")
	}
	if settings["mfaRequired"] != true {
		t.Error("Expected mfaRequired=true in settings")
	}

	// Verify password policy
	policy, ok := settings["passwordPolicy"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected passwordPolicy in settings")
	}
	if policy["minLength"] != float64(12) {
		t.Errorf("Expected minLength 12, got %v", policy["minLength"])
	}

	// Verify clients
	clients, ok := parsed["clients"].([]interface{})
	if !ok || len(clients) != 1 {
		t.Fatal("Expected 1 client in JSON")
	}
}

func TestAuthRealmReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewAuthRealmReconciler(cl, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

// ============================================================================
// GatewayRoute Controller Tests
// ============================================================================

func TestGatewayRouteReconciler_CreatesIngressWithKongAnnotations(t *testing.T) {
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

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "api-route", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "api-route", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer
	updatedRoute := &zenithv1.GatewayRoute{}
	cl.Get(context.Background(), types.NamespacedName{Name: "api-route", Namespace: "zenith-test"}, updatedRoute)
	if len(updatedRoute.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify Ingress was created
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gwr-api-route", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress to be created: %v", err)
	}

	// Verify Kong annotations
	if ingress.Annotations["kubernetes.io/ingress.class"] != "kong" {
		t.Error("Expected Kong ingress class annotation")
	}
	if ingress.Annotations["konghq.com/strip-path"] != "true" {
		t.Error("Expected strip-path annotation")
	}
	if ingress.Annotations["konghq.com/methods"] != "GET,POST" {
		t.Errorf("Expected methods annotation 'GET,POST', got '%s'", ingress.Annotations["konghq.com/methods"])
	}

	// Verify backend
	backend := ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service
	if backend.Name != "user-svc" {
		t.Errorf("Expected backend service 'user-svc', got '%s'", backend.Name)
	}
	if backend.Port.Number != 8080 {
		t.Errorf("Expected backend port 8080, got %d", backend.Port.Number)
	}

	// Verify path
	if ingress.Spec.Rules[0].HTTP.Paths[0].Path != "/api/v1/users" {
		t.Errorf("Expected path '/api/v1/users', got '%s'", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
	}

	// Verify status
	if updatedRoute.Status.Phase != "Active" {
		t.Errorf("Expected phase 'Active', got '%s'", updatedRoute.Status.Phase)
	}
	if updatedRoute.Status.KongRouteID == "" {
		t.Error("Expected KongRouteID to be set")
	}
}

func TestGatewayRouteReconciler_WithRateLimit(t *testing.T) {
	scheme := setupScheme()

	route := &zenithv1.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rate-limited-route",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GatewayRouteSpec{
			Path:    "/api/v1/data",
			Methods: []string{"GET"},
			Service: zenithv1.ServiceRef{Name: "data-svc", Port: 9090},
			RateLimit: &zenithv1.RateLimit{
				RequestsPerSecond: 10,
				RequestsPerMinute: 100,
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "rate-limited-route", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "rate-limited-route", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Ingress has rate-limiting plugin annotation
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gwr-rate-limited-route", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress: %v", err)
	}

	plugins := ingress.Annotations["konghq.com/plugins"]
	if !strings.Contains(plugins, "rate-limited-route-rate-limiting") {
		t.Errorf("Expected rate-limiting plugin in annotations, got '%s'", plugins)
	}
}

func TestGatewayRouteReconciler_WithCORS(t *testing.T) {
	scheme := setupScheme()

	route := &zenithv1.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cors-route",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GatewayRouteSpec{
			Path:    "/api/v1/public",
			Methods: []string{"GET", "POST", "OPTIONS"},
			Service: zenithv1.ServiceRef{Name: "public-svc", Port: 8080},
			CORS: &zenithv1.RouteCORS{
				AllowedOrigins: []string{"https://app.example.com", "https://admin.example.com"},
				AllowedMethods: []string{"GET", "POST"},
				AllowedHeaders: []string{"Authorization", "Content-Type"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cors-route", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "cors-route", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Ingress has CORS plugin annotation
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gwr-cors-route", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress: %v", err)
	}

	plugins := ingress.Annotations["konghq.com/plugins"]
	if !strings.Contains(plugins, "cors-route-cors") {
		t.Errorf("Expected cors plugin in annotations, got '%s'", plugins)
	}
}

func TestGatewayRouteReconciler_WithAuth(t *testing.T) {
	scheme := setupScheme()

	route := &zenithv1.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "auth-route",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GatewayRouteSpec{
			Path:    "/api/v1/protected",
			Methods: []string{"GET", "POST"},
			Service: zenithv1.ServiceRef{Name: "protected-svc", Port: 8080},
			Auth: &zenithv1.RouteAuth{
				Enabled: true,
				Scopes:  []string{"read", "write"},
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "auth-route", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "auth-route", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Ingress has JWT plugin annotation
	ingress := &networkingv1.Ingress{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gwr-auth-route", Namespace: "zenith-test"}, ingress); err != nil {
		t.Fatalf("Expected Ingress: %v", err)
	}

	plugins := ingress.Annotations["konghq.com/plugins"]
	if !strings.Contains(plugins, "auth-route-jwt") {
		t.Errorf("Expected jwt plugin in annotations, got '%s'", plugins)
	}
}

func TestGatewayRouteReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

func TestGatewayRouteReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	route := &zenithv1.GatewayRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-route",
			Namespace:         "zenith-test",
			DeletionTimestamp: &now,
			Finalizers:        []string{gatewayRouteFinalizer},
		},
		Spec: zenithv1.GatewayRouteSpec{
			Path:    "/api/test",
			Service: zenithv1.ServiceRef{Name: "test-svc", Port: 8080},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, route).
		WithStatusSubresource(route).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGatewayRouteReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-route", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updated := &zenithv1.GatewayRoute{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-route", Namespace: "zenith-test"}, updated)

	for _, f := range updated.Finalizers {
		if f == gatewayRouteFinalizer {
			t.Error("Expected finalizer to be removed during deletion")
		}
	}
}

// ============================================================================
// Finalizer Tests
// ============================================================================

func TestAppReconciler_FinalizerAddedOnFirstReconcile(t *testing.T) {
	scheme := setupScheme()

	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finalizer-app",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.AppSpec{
			Image: "nginx:latest",
			Port:  8080,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "finalizer-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "finalizer-app", Namespace: "zenith-test"}, updatedApp)

	foundFinalizer := false
	for _, f := range updatedApp.Finalizers {
		if f == appFinalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		t.Error("Expected app finalizer to be added on first reconcile")
	}
}

func TestProjectReconciler_FinalizerAddedOnFirstReconcile(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "finalizer-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Test",
			Owner:       "test@test.com",
			Plan:        "free",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "finalizer-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedProject := &zenithv1.Project{}
	cl.Get(context.Background(), types.NamespacedName{Name: "finalizer-project"}, updatedProject)

	foundFinalizer := false
	for _, f := range updatedProject.Finalizers {
		if f == projectFinalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		t.Error("Expected project finalizer to be added on first reconcile")
	}
}

func TestDatabaseReconciler_FinalizerAddedOnFirstReconcile(t *testing.T) {
	scheme := setupScheme()

	db := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finalizer-db",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.DatabaseSpec{
			Engine:  "postgresql",
			Version: "16",
			Storage: resource.MustParse("10Gi"),
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, db).
		WithStatusSubresource(db).
		Build()

	hetznerClient := hetzner.NewClient("")
	recorder := record.NewFakeRecorder(10)
	reconciler := NewDatabaseReconciler(cl, scheme, recorder, hetznerClient)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "finalizer-db", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedDB := &zenithv1.Database{}
	cl.Get(context.Background(), types.NamespacedName{Name: "finalizer-db", Namespace: "zenith-test"}, updatedDB)

	foundFinalizer := false
	for _, f := range updatedDB.Finalizers {
		if f == databaseFinalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		t.Error("Expected database finalizer to be added on first reconcile")
	}
}

// ============================================================================
// Deletion Tests
// ============================================================================

func TestAppReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-app",
			Namespace:         "zenith-test",
			DeletionTimestamp: &now,
			Finalizers:        []string{appFinalizer},
		},
		Spec: zenithv1.AppSpec{
			Image: "nginx:latest",
			Port:  8080,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, app).
		WithStatusSubresource(app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewAppReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-app", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedApp := &zenithv1.App{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-app", Namespace: "zenith-test"}, updatedApp)

	for _, f := range updatedApp.Finalizers {
		if f == appFinalizer {
			t.Error("Expected finalizer to be removed during deletion")
		}
	}
}

func TestProjectReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-project",
			DeletionTimestamp: &now,
			Finalizers:        []string{projectFinalizer},
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Test",
			Owner:       "test@test.com",
			Plan:        "free",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedProject := &zenithv1.Project{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-project"}, updatedProject)

	for _, f := range updatedProject.Finalizers {
		if f == projectFinalizer {
			t.Error("Expected finalizer to be removed during deletion")
		}
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestProjectReconciler_EnterprisePlanQuotas(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "enterprise-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Enterprise Project",
			Owner:       "enterprise@test.com",
			Plan:        "enterprise",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "enterprise-project"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "enterprise-project"},
	})

	rq := &corev1.ResourceQuota{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "enterprise-project-quota",
		Namespace: "zenith-enterprise-project",
	}, rq); err != nil {
		t.Fatalf("Expected ResourceQuota: %v", err)
	}

	cpuLimit := rq.Spec.Hard[corev1.ResourceLimitsCPU]
	if cpuLimit.Cmp(resource.MustParse("32")) != 0 {
		t.Errorf("Expected enterprise CPU limit '32', got '%s'", cpuLimit.String())
	}
	memLimit := rq.Spec.Hard[corev1.ResourceLimitsMemory]
	if memLimit.Cmp(resource.MustParse("64Gi")) != 0 {
		t.Errorf("Expected enterprise memory limit '64Gi', got '%s'", memLimit.String())
	}
	storageLimit := rq.Spec.Hard[corev1.ResourceRequestsStorage]
	if storageLimit.Cmp(resource.MustParse("1000Gi")) != 0 {
		t.Errorf("Expected enterprise storage limit '1000Gi', got '%s'", storageLimit.String())
	}
}

func TestProjectReconciler_AppAndDatabaseCounting(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "count-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Count Project",
			Owner:       "user@test.com",
			Plan:        "pro",
		},
	}

	// Pre-create the namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-count-project"}}

	// Create some apps in the project namespace
	replicas := int32(1)
	app1 := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{Name: "app1", Namespace: "zenith-count-project"},
		Spec:       zenithv1.AppSpec{Image: "nginx:latest", Port: 8080, Replicas: &replicas},
	}
	app2 := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{Name: "app2", Namespace: "zenith-count-project"},
		Spec:       zenithv1.AppSpec{Image: "node:20", Port: 3000, Replicas: &replicas},
	}

	// Create a database
	db1 := &zenithv1.Database{
		ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "zenith-count-project"},
		Spec:       zenithv1.DatabaseSpec{Engine: "postgresql", Version: "16", Storage: resource.MustParse("10Gi")},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, ns, app1, app2, db1).
		WithStatusSubresource(project, app1, app2, db1).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewProjectReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "count-project"},
	})
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "count-project"},
	})

	// Verify status counts
	updatedProject := &zenithv1.Project{}
	cl.Get(context.Background(), types.NamespacedName{Name: "count-project"}, updatedProject)

	if updatedProject.Status.AppCount != 2 {
		t.Errorf("Expected AppCount 2, got %d", updatedProject.Status.AppCount)
	}
	if updatedProject.Status.DatabaseCount != 1 {
		t.Errorf("Expected DatabaseCount 1, got %d", updatedProject.Status.DatabaseCount)
	}
	if updatedProject.Status.Namespace != "zenith-count-project" {
		t.Errorf("Expected namespace 'zenith-count-project', got '%s'", updatedProject.Status.Namespace)
	}
	if updatedProject.Status.Phase != "Active" {
		t.Errorf("Expected phase 'Active', got '%s'", updatedProject.Status.Phase)
	}
}

func TestPlanQuotas(t *testing.T) {
	tests := []struct {
		plan           string
		expectedApps   int
		expectedDBs    int
		expectedGB     int
		expectedCPU    string
		expectedMemory string
	}{
		{"free", 5, 2, 10, "2", "4Gi"},
		{"pro", 25, 10, 100, "8", "16Gi"},
		{"enterprise", 100, 50, 1000, "32", "64Gi"},
		{"unknown", 5, 2, 10, "2", "4Gi"}, // defaults to free
	}

	for _, tt := range tests {
		t.Run(tt.plan, func(t *testing.T) {
			apps, dbs, gb, cpu, mem := planQuotas(tt.plan)
			if apps != tt.expectedApps {
				t.Errorf("Expected %d apps, got %d", tt.expectedApps, apps)
			}
			if dbs != tt.expectedDBs {
				t.Errorf("Expected %d databases, got %d", tt.expectedDBs, dbs)
			}
			if gb != tt.expectedGB {
				t.Errorf("Expected %d GB, got %d", tt.expectedGB, gb)
			}
			if cpu != tt.expectedCPU {
				t.Errorf("Expected CPU '%s', got '%s'", tt.expectedCPU, cpu)
			}
			if mem != tt.expectedMemory {
				t.Errorf("Expected memory '%s', got '%s'", tt.expectedMemory, mem)
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	p1 := generatePassword(24)
	p2 := generatePassword(24)

	if len(p1) != 24 {
		t.Errorf("Expected password length 24, got %d", len(p1))
	}
	if p1 == p2 {
		t.Error("Expected different passwords on successive calls")
	}
}

func TestEngineConfig(t *testing.T) {
	tests := []struct {
		engine      string
		version     string
		expectedImg string
		expectedDir string
		expectedPort int32
	}{
		{"postgresql", "16", "postgres:16", "/var/lib/postgresql/data", 5432},
		{"mysql", "8.0", "mysql:8.0", "/var/lib/mysql", 3306},
		{"mongodb", "7", "mongo:7", "/data/db", 27017},
		{"redis", "7.2", "redis:7.2", "/data", 6379},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			port, image, dataDir, _ := engineConfig(tt.engine, tt.version)
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
			if image != tt.expectedImg {
				t.Errorf("Expected image '%s', got '%s'", tt.expectedImg, image)
			}
			if dataDir != tt.expectedDir {
				t.Errorf("Expected dataDir '%s', got '%s'", tt.expectedDir, dataDir)
			}
		})
	}
}

// ============================================================================
// GitSync Controller Tests
// ============================================================================

func TestGitSyncReconciler_CreatesSyncConfigMap(t *testing.T) {
	scheme := setupScheme()

	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-sync",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL:  "https://github.com/example/repo.git",
			Branch:   "main",
			Path:     "/manifests",
			Interval: "10m",
			AutoSync: true,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	// First reconcile adds finalizer
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-sync", Namespace: "zenith-test"},
	})

	// Second reconcile creates resources
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-sync", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should requeue for periodic sync since AutoSync is true
	if result.RequeueAfter == 0 {
		t.Error("Expected RequeueAfter to be set for AutoSync")
	}

	// Verify finalizer
	updatedGS := &zenithv1.GitSync{}
	cl.Get(context.Background(), types.NamespacedName{Name: "my-sync", Namespace: "zenith-test"}, updatedGS)
	if len(updatedGS.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	// Verify sync ConfigMap was created
	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gitsync-my-sync", Namespace: "zenith-test"}, cm); err != nil {
		t.Fatalf("Expected sync ConfigMap to be created: %v", err)
	}
	if cm.Data["repoURL"] != "https://github.com/example/repo.git" {
		t.Errorf("Expected repoURL in ConfigMap, got '%s'", cm.Data["repoURL"])
	}
	if cm.Data["branch"] != "main" {
		t.Errorf("Expected branch 'main', got '%s'", cm.Data["branch"])
	}
	if cm.Data["path"] != "/manifests" {
		t.Errorf("Expected path '/manifests', got '%s'", cm.Data["path"])
	}
	if cm.Data["autoSync"] != "true" {
		t.Errorf("Expected autoSync 'true', got '%s'", cm.Data["autoSync"])
	}
	if cm.Labels["zenith.dev/gitsync"] != "my-sync" {
		t.Errorf("Expected gitsync label, got '%s'", cm.Labels["zenith.dev/gitsync"])
	}

	// Verify owner reference
	if len(cm.OwnerReferences) == 0 {
		t.Error("Expected owner reference on ConfigMap")
	}

	// Verify status
	if updatedGS.Status.Phase != "Synced" {
		t.Errorf("Expected phase 'Synced', got '%s'", updatedGS.Status.Phase)
	}
	if updatedGS.Status.LastSyncTime == nil {
		t.Error("Expected LastSyncTime to be set")
	}
	if updatedGS.Status.LastCommitHash == "" {
		t.Error("Expected LastCommitHash to be set")
	}
}

func TestGitSyncReconciler_CustomInterval(t *testing.T) {
	scheme := setupScheme()

	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-interval",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL:  "https://github.com/example/repo.git",
			Branch:   "main",
			Interval: "30m",
			AutoSync: true,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "custom-interval", Namespace: "zenith-test"},
	})
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "custom-interval", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should requeue with the custom interval
	expectedDuration := 30 * time.Minute
	if result.RequeueAfter != expectedDuration {
		t.Errorf("Expected RequeueAfter %v, got %v", expectedDuration, result.RequeueAfter)
	}
}

func TestGitSyncReconciler_ParseManifestsMultipleDocs(t *testing.T) {
	yamlData := `
apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: app-one
spec:
  image: nginx:latest
---
apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: db-one
spec:
  engine: postgresql
---
apiVersion: zenith.dev/v1alpha1
kind: StorageBucket
metadata:
  name: bucket-one
spec:
  access: private
`

	results, err := ParseManifests([]byte(yamlData))
	if err != nil {
		t.Fatalf("ParseManifests failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 manifests, got %d", len(results))
	}

	expectedKinds := []string{"App", "Database", "StorageBucket"}
	expectedNames := []string{"app-one", "db-one", "bucket-one"}
	for i, r := range results {
		if r.GetKind() != expectedKinds[i] {
			t.Errorf("Manifest %d: expected kind '%s', got '%s'", i, expectedKinds[i], r.GetKind())
		}
		if r.GetName() != expectedNames[i] {
			t.Errorf("Manifest %d: expected name '%s', got '%s'", i, expectedNames[i], r.GetName())
		}
	}
}

func TestGitSyncReconciler_DefaultBranchAndInterval(t *testing.T) {
	scheme := setupScheme()

	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-sync",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL:  "https://github.com/example/repo.git",
			AutoSync: true,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "default-sync", Namespace: "zenith-test"},
	})
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "default-sync", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify ConfigMap uses defaults
	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "gitsync-default-sync", Namespace: "zenith-test"}, cm); err != nil {
		t.Fatalf("Expected sync ConfigMap: %v", err)
	}
	if cm.Data["branch"] != "main" {
		t.Errorf("Expected default branch 'main', got '%s'", cm.Data["branch"])
	}
	if cm.Data["path"] != "/" {
		t.Errorf("Expected default path '/', got '%s'", cm.Data["path"])
	}
}

func TestGitSyncReconciler_NoAutoSyncNoRequeue(t *testing.T) {
	scheme := setupScheme()

	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-auto-sync",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL:  "https://github.com/example/repo.git",
			AutoSync: false,
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	// Two reconciles
	reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-auto-sync", Namespace: "zenith-test"},
	})
	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-auto-sync", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should NOT requeue when AutoSync is false
	if result.RequeueAfter != 0 {
		t.Errorf("Expected no requeue when AutoSync is false, got %v", result.RequeueAfter)
	}
}

func TestGitSyncReconciler_FinalizerAddedOnFirstReconcile(t *testing.T) {
	scheme := setupScheme()

	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finalizer-gs",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL: "https://github.com/example/repo.git",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "finalizer-gs", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedGS := &zenithv1.GitSync{}
	cl.Get(context.Background(), types.NamespacedName{Name: "finalizer-gs", Namespace: "zenith-test"}, updatedGS)

	foundFinalizer := false
	for _, f := range updatedGS.Finalizers {
		if f == gitSyncFinalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		t.Error("Expected gitsync finalizer to be added on first reconcile")
	}
}

func TestGitSyncReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	gs := &zenithv1.GitSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-gs",
			Namespace:         "zenith-test",
			DeletionTimestamp: &now,
			Finalizers:        []string{gitSyncFinalizer},
		},
		Spec: zenithv1.GitSyncSpec{
			RepoURL: "https://github.com/example/repo.git",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, gs).
		WithStatusSubresource(gs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-gs", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedGS := &zenithv1.GitSync{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-gs", Namespace: "zenith-test"}, updatedGS)

	for _, f := range updatedGS.Finalizers {
		if f == gitSyncFinalizer {
			t.Error("Expected gitsync finalizer to be removed during deletion")
		}
	}
}

func TestGitSyncReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewGitSyncReconciler(cl, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

func TestParseManifests(t *testing.T) {
	yamlData := `
apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: web-app
spec:
  image: nginx:latest
  replicas: 2
---
apiVersion: zenith.dev/v1alpha1
kind: Database
metadata:
  name: my-db
spec:
  engine: postgresql
`

	results, err := ParseManifests([]byte(yamlData))
	if err != nil {
		t.Fatalf("ParseManifests failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 manifests, got %d", len(results))
	}

	if results[0].GetKind() != "App" {
		t.Errorf("Expected kind 'App', got '%s'", results[0].GetKind())
	}
	if results[0].GetName() != "web-app" {
		t.Errorf("Expected name 'web-app', got '%s'", results[0].GetName())
	}

	if results[1].GetKind() != "Database" {
		t.Errorf("Expected kind 'Database', got '%s'", results[1].GetKind())
	}
	if results[1].GetName() != "my-db" {
		t.Errorf("Expected name 'my-db', got '%s'", results[1].GetName())
	}
}

func TestParseManifests_EmptyDocument(t *testing.T) {
	yamlData := `---
---
apiVersion: zenith.dev/v1alpha1
kind: App
metadata:
  name: test
spec:
  image: test:latest
---
`

	results, err := ParseManifests([]byte(yamlData))
	if err != nil {
		t.Fatalf("ParseManifests failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 manifest (empty docs filtered), got %d", len(results))
	}
}

func TestParseManifests_InvalidYAML(t *testing.T) {
	yamlData := `
not: valid: yaml: [
`

	_, err := ParseManifests([]byte(yamlData))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestDBUsername(t *testing.T) {
	tests := []struct {
		engine   string
		expected string
	}{
		{"postgresql", "postgres"},
		{"mysql", "root"},
		{"mongodb", "admin"},
		{"redis", ""},
		{"unknown", "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			result := dbUsername(tt.engine)
			if result != tt.expected {
				t.Errorf("Expected username '%s' for engine '%s', got '%s'", tt.expected, tt.engine, result)
			}
		})
	}
}

func TestBuildDBEnvVars(t *testing.T) {
	// PostgreSQL should have POSTGRES_PASSWORD, POSTGRES_USER, POSTGRES_DB
	envVars := map[string]string{
		"passwordEnv": "POSTGRES_PASSWORD",
		"userEnv":     "POSTGRES_USER",
		"dbEnv":       "POSTGRES_DB",
	}
	result := buildDBEnvVars("postgresql", "test-pass", envVars)
	if len(result) != 3 {
		t.Fatalf("Expected 3 env vars for PostgreSQL, got %d", len(result))
	}

	envMap := make(map[string]string)
	for _, e := range result {
		envMap[e.Name] = e.Value
	}
	if envMap["POSTGRES_PASSWORD"] != "test-pass" {
		t.Errorf("Expected POSTGRES_PASSWORD='test-pass', got '%s'", envMap["POSTGRES_PASSWORD"])
	}
	if envMap["POSTGRES_USER"] != "postgres" {
		t.Errorf("Expected POSTGRES_USER='postgres', got '%s'", envMap["POSTGRES_USER"])
	}
	if envMap["POSTGRES_DB"] != "zenith" {
		t.Errorf("Expected POSTGRES_DB='zenith', got '%s'", envMap["POSTGRES_DB"])
	}

	// Redis should have no env vars (empty map)
	redisResult := buildDBEnvVars("redis", "test-pass", map[string]string{})
	if len(redisResult) != 0 {
		t.Errorf("Expected 0 env vars for Redis, got %d", len(redisResult))
	}
}

func TestBuildDBReadinessProbe(t *testing.T) {
	tests := []struct {
		engine      string
		port        int32
		expectExec  bool
		expectedCmd string
	}{
		{"postgresql", 5432, true, "pg_isready"},
		{"mysql", 3306, true, "mysqladmin"},
		{"mongodb", 27017, true, "mongosh"},
		{"redis", 6379, true, "redis-cli"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			probe := buildDBReadinessProbe(tt.engine, tt.port)
			if probe == nil {
				t.Fatal("Expected non-nil probe")
			}
			if tt.expectExec {
				if probe.Exec == nil {
					t.Fatal("Expected exec probe")
				}
				if probe.Exec.Command[0] != tt.expectedCmd {
					t.Errorf("Expected command '%s', got '%s'", tt.expectedCmd, probe.Exec.Command[0])
				}
			}
			if probe.PeriodSeconds != 10 {
				t.Errorf("Expected period 10, got %d", probe.PeriodSeconds)
			}
		})
	}

	// Test unknown engine falls back to TCP
	unknownProbe := buildDBReadinessProbe("unknown", 9999)
	if unknownProbe.TCPSocket == nil {
		t.Error("Expected TCP probe for unknown engine")
	}
}

func TestBackupCommand(t *testing.T) {
	tests := []struct {
		engine   string
		contains string
	}{
		{"postgresql", "pg_dump"},
		{"mysql", "mysqldump"},
		{"mongodb", "mongodump"},
		{"redis", "redis-cli"},
		{"unknown", "Unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			cmd := backupCommand(tt.engine, "testdb", 5432)
			if !strings.Contains(cmd, tt.contains) {
				t.Errorf("Expected backup command for '%s' to contain '%s', got '%s'", tt.engine, tt.contains, cmd)
			}
		})
	}
}

func TestGeneratePasswordLength(t *testing.T) {
	lengths := []int{8, 16, 24, 32, 40}
	for _, l := range lengths {
		p := generatePassword(l)
		if len(p) != l {
			t.Errorf("Expected password length %d, got %d", l, len(p))
		}
	}
}

func TestBuildConnectionString(t *testing.T) {
	tests := []struct {
		engine   string
		expected string
	}{
		{"postgresql", "postgresql://postgres:pass@host:5432/db?sslmode=disable"},
		{"mysql", "mysql://root:pass@host:3306/db"},
		{"mongodb", "mongodb://admin:pass@host:27017/db"},
		{"redis", "redis://:pass@host:6379"},
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			port, _, _, _ := engineConfig(tt.engine, "16")
			user := dbUsername(tt.engine)
			result := buildConnectionString(tt.engine, user, "pass", "host", port, "db")
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// Crossplane Controller Tests
// ============================================================================

func TestCrossplaneReconciler_CreatesFinalizerAndStatus(t *testing.T) {
	scheme := setupScheme()

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-bucket",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:          "aws",
			ResourceKind:      "Bucket",
			ProviderConfigRef: "default",
			DeletionPolicy:    "Delete",
			Config: map[string]string{
				"region": "eu-central-1",
				"acl":    "private",
			},
			WriteConnectionSecretToRef: &zenithv1.SecretKeyRef{
				Name: "bucket-conn",
				Key:  "endpoint",
			},
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, cr).
		WithStatusSubresource(cr).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	// First reconcile adds finalizer
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify finalizer was added
	updatedCR := &zenithv1.CrossplaneResource{}
	cl.Get(context.Background(), types.NamespacedName{Name: "my-bucket", Namespace: "zenith-test"}, updatedCR)
	if len(updatedCR.Finalizers) == 0 {
		t.Error("Expected finalizer to be added")
	}

	foundFinalizer := false
	for _, f := range updatedCR.Finalizers {
		if f == crossplaneFinalizer {
			foundFinalizer = true
			break
		}
	}
	if !foundFinalizer {
		t.Error("Expected crossplane finalizer to be added")
	}
}

func TestCrossplaneReconciler_DeletionRemovesFinalizer(t *testing.T) {
	scheme := setupScheme()

	now := metav1.Now()
	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "deleting-bucket",
			Namespace:         "zenith-test",
			DeletionTimestamp: &now,
			Finalizers:        []string{crossplaneFinalizer},
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:     "aws",
			ResourceKind: "Bucket",
		},
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-test"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns, cr).
		WithStatusSubresource(cr).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-bucket", Namespace: "zenith-test"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	updatedCR := &zenithv1.CrossplaneResource{}
	cl.Get(context.Background(), types.NamespacedName{Name: "deleting-bucket", Namespace: "zenith-test"}, updatedCR)

	for _, f := range updatedCR.Finalizers {
		if f == crossplaneFinalizer {
			t.Error("Expected finalizer to be removed during deletion")
		}
	}
}

func TestCrossplaneReconciler_NotFound(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "zenith-test"},
	})

	if err != nil {
		t.Fatalf("Expected no error for not found, got: %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

func TestCrossplaneDefaultAPIVersion(t *testing.T) {
	tests := []struct {
		provider     string
		resourceKind string
		expected     string
	}{
		{"aws", "Bucket", "s3.aws.upbound.io/v1beta1"},
		{"aws", "Instance", "ec2.aws.upbound.io/v1beta1"},
		{"aws", "Database", "rds.aws.upbound.io/v1beta1"},
		{"gcp", "Bucket", "storage.gcp.upbound.io/v1beta1"},
		{"gcp", "Instance", "compute.gcp.upbound.io/v1beta1"},
		{"azure", "StorageAccount", "storage.azure.upbound.io/v1beta1"},
		{"azure", "VirtualMachine", "compute.azure.upbound.io/v1beta1"},
		{"hetzner", "Server", "hcloud.crossplane.io/v1alpha1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.resourceKind, func(t *testing.T) {
			result := defaultAPIVersion(tt.provider, tt.resourceKind)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCrossplaneDefaultAPIVersion_AllProviders(t *testing.T) {
	// Test all default provider+kind combos including defaults/fallbacks
	tests := []struct {
		provider     string
		resourceKind string
		expected     string
	}{
		{"aws", "Bucket", "s3.aws.upbound.io/v1beta1"},
		{"aws", "Instance", "ec2.aws.upbound.io/v1beta1"},
		{"aws", "Database", "rds.aws.upbound.io/v1beta1"},
		{"aws", "RDSInstance", "rds.aws.upbound.io/v1beta1"},
		{"aws", "SomethingElse", "aws.upbound.io/v1beta1"},
		{"gcp", "Bucket", "storage.gcp.upbound.io/v1beta1"},
		{"gcp", "Instance", "compute.gcp.upbound.io/v1beta1"},
		{"gcp", "SomethingElse", "gcp.upbound.io/v1beta1"},
		{"azure", "StorageAccount", "storage.azure.upbound.io/v1beta1"},
		{"azure", "VirtualMachine", "compute.azure.upbound.io/v1beta1"},
		{"azure", "SomethingElse", "azure.upbound.io/v1beta1"},
		{"hetzner", "Server", "hcloud.crossplane.io/v1alpha1"},
		{"hetzner", "Volume", "hcloud.crossplane.io/v1alpha1"},
		{"custom-provider", "AnyKind", "custom-provider.crossplane.io/v1alpha1"},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.resourceKind, func(t *testing.T) {
			result := defaultAPIVersion(tt.provider, tt.resourceKind)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCrossplaneGVR(t *testing.T) {
	tests := []struct {
		apiVersion    string
		kind          string
		expectedGroup string
		expectedVer   string
		expectedRes   string
	}{
		{"s3.aws.upbound.io/v1beta1", "Bucket", "s3.aws.upbound.io", "v1beta1", "buckets"},
		{"hcloud.crossplane.io/v1alpha1", "Server", "hcloud.crossplane.io", "v1alpha1", "servers"},
		{"v1", "ConfigMap", "", "v1", "configmaps"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			gvr := crossplaneGVR(tt.apiVersion, tt.kind)
			if gvr.Group != tt.expectedGroup {
				t.Errorf("Expected group '%s', got '%s'", tt.expectedGroup, gvr.Group)
			}
			if gvr.Version != tt.expectedVer {
				t.Errorf("Expected version '%s', got '%s'", tt.expectedVer, gvr.Version)
			}
			if gvr.Resource != tt.expectedRes {
				t.Errorf("Expected resource '%s', got '%s'", tt.expectedRes, gvr.Resource)
			}
		})
	}
}

func TestCrossplaneBuildResource(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-bucket",
			Namespace: "zenith-myproj",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:          "aws",
			ResourceKind:      "Bucket",
			ProviderConfigRef: "aws-config",
			DeletionPolicy:    "Delete",
			Config: map[string]string{
				"region": "eu-central-1",
				"acl":    "private",
			},
			WriteConnectionSecretToRef: &zenithv1.SecretKeyRef{
				Name: "bucket-secret",
				Key:  "endpoint",
			},
		},
	}

	resource := reconciler.buildCrossplaneResource(cr)

	if resource.GetKind() != "Bucket" {
		t.Errorf("Expected kind 'Bucket', got '%s'", resource.GetKind())
	}

	expectedName := "zenith-zenith-myproj-test-bucket"
	if resource.GetName() != expectedName {
		t.Errorf("Expected name '%s', got '%s'", expectedName, resource.GetName())
	}

	labels := resource.GetLabels()
	if labels["app.kubernetes.io/managed-by"] != "zenith-operator" {
		t.Error("Expected managed-by label")
	}
	if labels["zenith.dev/crossplane-resource"] != "test-bucket" {
		t.Error("Expected crossplane-resource label")
	}
}

// ============================================================================
// Backstage Catalog Tests
// ============================================================================

func TestBackstageCatalogReconciler_GeneratesConfigMap(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "catalog-project",
			Labels: map[string]string{
				"zenith.dev/owner": "owner@test.com",
			},
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Catalog Project",
			Owner:       "owner@test.com",
			Plan:        "pro",
		},
	}

	// Create system namespace for the ConfigMap
	systemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-system"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, systemNs).
		WithStatusSubresource(project).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewBackstageCatalogReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "catalog-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify ConfigMap was created
	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "backstage-catalog",
		Namespace: "zenith-system",
	}, cm); err != nil {
		t.Fatalf("Expected ConfigMap to be created: %v", err)
	}

	if cm.Data["catalog-info.json"] == "" {
		t.Error("Expected catalog-info.json data in ConfigMap")
	}

	// Verify labels
	if cm.Labels["app.kubernetes.io/managed-by"] != "zenith-operator" {
		t.Error("Expected managed-by label on ConfigMap")
	}
	if cm.Labels["app.kubernetes.io/component"] != "backstage-catalog" {
		t.Error("Expected component label on ConfigMap")
	}
}

func TestBackstageCatalogReconciler_IncludesAppsAndDatabases(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "full-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Full Project",
			Owner:       "user@test.com",
			Plan:        "pro",
		},
	}

	replicas := int32(2)
	app := &zenithv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-app",
			Namespace: "zenith-full-project",
			Labels: map[string]string{
				"zenith.dev/project": "full-project",
			},
		},
		Spec: zenithv1.AppSpec{
			Image:    "nginx:latest",
			Replicas: &replicas,
			Port:     8080,
		},
	}

	systemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-system"}}
	projectNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-full-project"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, app, systemNs, projectNs).
		WithStatusSubresource(project, app).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewBackstageCatalogReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "full-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "backstage-catalog",
		Namespace: "zenith-system",
	}, cm); err != nil {
		t.Fatalf("Expected ConfigMap: %v", err)
	}

	catalogJSON := cm.Data["catalog-info.json"]
	if catalogJSON == "" {
		t.Fatal("Expected non-empty catalog JSON")
	}

	// The catalog should contain at least the project (System) and app (Component)
	if len(catalogJSON) < 10 {
		t.Error("Expected substantial catalog JSON content")
	}
}

func TestCrossplaneBuildResource_GCPProvider(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-instance",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:          "gcp",
			ResourceKind:      "Instance",
			ProviderConfigRef: "gcp-config",
			DeletionPolicy:    "Orphan",
			Config: map[string]string{
				"zone":        "us-central1-a",
				"machineType": "n1-standard-1",
			},
		},
	}

	resource := reconciler.buildCrossplaneResource(cr)

	if resource.GetKind() != "Instance" {
		t.Errorf("Expected kind 'Instance', got '%s'", resource.GetKind())
	}
	if resource.GetAPIVersion() != "compute.gcp.upbound.io/v1beta1" {
		t.Errorf("Expected API version 'compute.gcp.upbound.io/v1beta1', got '%s'", resource.GetAPIVersion())
	}

	// Verify spec
	spec, ok := resource.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected spec in resource")
	}
	if spec["deletionPolicy"] != "Orphan" {
		t.Errorf("Expected deletionPolicy 'Orphan', got '%v'", spec["deletionPolicy"])
	}

	providerConfigRef, ok := spec["providerConfigRef"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected providerConfigRef in spec")
	}
	if providerConfigRef["name"] != "gcp-config" {
		t.Errorf("Expected provider config ref 'gcp-config', got '%v'", providerConfigRef["name"])
	}

	forProvider, ok := spec["forProvider"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected forProvider in spec")
	}
	if forProvider["zone"] != "us-central1-a" {
		t.Errorf("Expected zone 'us-central1-a', got '%v'", forProvider["zone"])
	}
}

func TestCrossplaneBuildResource_CustomAPIVersion(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-res",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:           "aws",
			ResourceKind:       "VPCEndpoint",
			ResourceAPIVersion: "ec2.aws.upbound.io/v1beta2",
			ProviderConfigRef:  "aws-config",
			DeletionPolicy:     "Delete",
			Config:             map[string]string{"vpcId": "vpc-123"},
		},
	}

	resource := reconciler.buildCrossplaneResource(cr)

	// When ResourceAPIVersion is specified, it should use that instead of defaulting
	if resource.GetAPIVersion() != "ec2.aws.upbound.io/v1beta2" {
		t.Errorf("Expected custom API version 'ec2.aws.upbound.io/v1beta2', got '%s'", resource.GetAPIVersion())
	}
}

func TestCrossplaneBuildResource_WithConnectionSecret(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-instance",
			Namespace: "zenith-myproj",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:          "aws",
			ResourceKind:      "Database",
			ProviderConfigRef: "aws-config",
			DeletionPolicy:    "Delete",
			Config:            map[string]string{"engine": "postgres"},
			WriteConnectionSecretToRef: &zenithv1.SecretKeyRef{
				Name: "db-conn-secret",
				Key:  "password",
			},
		},
	}

	resource := reconciler.buildCrossplaneResource(cr)

	spec, ok := resource.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected spec in resource")
	}

	connSecretRef, ok := spec["writeConnectionSecretToRef"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected writeConnectionSecretToRef in spec")
	}
	if connSecretRef["name"] != "db-conn-secret" {
		t.Errorf("Expected secret name 'db-conn-secret', got '%v'", connSecretRef["name"])
	}
	if connSecretRef["namespace"] != "zenith-myproj" {
		t.Errorf("Expected secret namespace 'zenith-myproj', got '%v'", connSecretRef["namespace"])
	}
}

func TestCrossplaneBuildResource_NoConnectionSecret(t *testing.T) {
	scheme := setupScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(10)
	reconciler := NewCrossplaneReconciler(cl, scheme, recorder)

	cr := &zenithv1.CrossplaneResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-res",
			Namespace: "zenith-test",
		},
		Spec: zenithv1.CrossplaneResourceSpec{
			Provider:          "hetzner",
			ResourceKind:      "Server",
			ProviderConfigRef: "hetzner-config",
			DeletionPolicy:    "Delete",
			Config:            map[string]string{"serverType": "cx22"},
		},
	}

	resource := reconciler.buildCrossplaneResource(cr)

	spec, ok := resource.Object["spec"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected spec in resource")
	}

	if _, exists := spec["writeConnectionSecretToRef"]; exists {
		t.Error("Expected no writeConnectionSecretToRef when not specified")
	}
}

func TestBoolToString(t *testing.T) {
	if boolToString(true) != "True" {
		t.Errorf("Expected 'True', got '%s'", boolToString(true))
	}
	if boolToString(false) != "False" {
		t.Errorf("Expected 'False', got '%s'", boolToString(false))
	}
}

func TestPhaseReason(t *testing.T) {
	if phaseReason(true) != "ResourceReady" {
		t.Errorf("Expected 'ResourceReady', got '%s'", phaseReason(true))
	}
	if phaseReason(false) != "ResourceNotReady" {
		t.Errorf("Expected 'ResourceNotReady', got '%s'", phaseReason(false))
	}
}

// ============================================================================
// Backstage Catalog Tests (Additional)
// ============================================================================

func TestBackstageCatalogReconciler_WithDomains(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "domain-project",
		},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Domain Project",
			Owner:       "user@test.com",
			Plan:        "pro",
		},
	}

	domain := &zenithv1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-domain",
			Namespace: "zenith-domain-project",
			Labels: map[string]string{
				"zenith.dev/project": "domain-project",
			},
		},
		Spec: zenithv1.DomainSpec{
			Domain: "api.example.com",
			AppRef: "web-app",
		},
	}

	systemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-system"}}
	projectNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-domain-project"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, domain, systemNs, projectNs).
		WithStatusSubresource(project, domain).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewBackstageCatalogReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "domain-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "backstage-catalog",
		Namespace: "zenith-system",
	}, cm); err != nil {
		t.Fatalf("Expected ConfigMap: %v", err)
	}

	catalogJSON := cm.Data["catalog-info.json"]
	if catalogJSON == "" {
		t.Fatal("Expected non-empty catalog JSON")
	}

	// Parse and verify it contains an API entity from the domain
	var entities []backstageEntity
	if err := json.Unmarshal([]byte(catalogJSON), &entities); err != nil {
		t.Fatalf("Failed to parse catalog JSON: %v", err)
	}

	foundAPI := false
	for _, e := range entities {
		if e.Kind == "API" && e.Metadata.Name == "api-domain" {
			foundAPI = true
			if e.Metadata.Annotations["zenith.dev/domain"] != "api.example.com" {
				t.Errorf("Expected domain annotation 'api.example.com', got '%s'", e.Metadata.Annotations["zenith.dev/domain"])
			}
			if e.Metadata.Annotations["zenith.dev/app-ref"] != "web-app" {
				t.Errorf("Expected app-ref annotation 'web-app', got '%s'", e.Metadata.Annotations["zenith.dev/app-ref"])
			}
			if e.Spec["type"] != "openapi" {
				t.Errorf("Expected API type 'openapi', got '%v'", e.Spec["type"])
			}
		}
	}
	if !foundAPI {
		t.Error("Expected API entity from domain in catalog")
	}
}

func TestBackstageCatalogReconciler_WithStorageBuckets(t *testing.T) {
	scheme := setupScheme()

	project := &zenithv1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "storage-project"},
		Spec: zenithv1.ProjectSpec{
			DisplayName: "Storage Project",
			Owner:       "user@test.com",
			Plan:        "pro",
		},
	}

	bucket := &zenithv1.StorageBucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-bucket",
			Namespace: "zenith-storage-project",
			Labels: map[string]string{
				"zenith.dev/project": "storage-project",
			},
		},
		Spec: zenithv1.StorageBucketSpec{
			Access: "private",
			Region: "fsn1",
		},
	}

	systemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-system"}}
	projectNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-storage-project"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, bucket, systemNs, projectNs).
		WithStatusSubresource(project, bucket).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewBackstageCatalogReconciler(cl, scheme, recorder)

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "storage-project"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	cm := &corev1.ConfigMap{}
	cl.Get(context.Background(), types.NamespacedName{Name: "backstage-catalog", Namespace: "zenith-system"}, cm)

	var entities []backstageEntity
	json.Unmarshal([]byte(cm.Data["catalog-info.json"]), &entities)

	foundBucket := false
	for _, e := range entities {
		if e.Kind == "Resource" && e.Metadata.Name == "my-bucket" {
			foundBucket = true
			if e.Spec["type"] != "storage" {
				t.Errorf("Expected type 'storage', got '%v'", e.Spec["type"])
			}
		}
	}
	if !foundBucket {
		t.Error("Expected Resource entity from storage bucket in catalog")
	}
}

func TestBackstageCatalogReconciler_EmptyCluster(t *testing.T) {
	scheme := setupScheme()

	systemNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "zenith-system"}}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(systemNs).
		Build()

	recorder := record.NewFakeRecorder(10)
	reconciler := NewBackstageCatalogReconciler(cl, scheme, recorder)

	// Should not fail even if there are no projects
	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := cl.Get(context.Background(), types.NamespacedName{
		Name:      "backstage-catalog",
		Namespace: "zenith-system",
	}, cm); err != nil {
		t.Fatalf("Expected ConfigMap even with empty cluster: %v", err)
	}
}

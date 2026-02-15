package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
)

const authRealmFinalizer = "zenith.dev/authrealm-cleanup"

type AuthRealmReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewAuthRealmReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder) *AuthRealmReconciler {
	return &AuthRealmReconciler{Client: c, Scheme: s, Recorder: r}
}

func (r *AuthRealmReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var realm zenithv1.AuthRealm
	if err := r.Get(ctx, req.NamespacedName, &realm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !realm.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&realm, authRealmFinalizer) {
			logger.Info("Cleaning up auth realm resources", "name", realm.Name)
			// K8s cascade handles children via owner references
			controllerutil.RemoveFinalizer(&realm, authRealmFinalizer)
			if err := r.Update(ctx, &realm); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&realm, authRealmFinalizer) {
		controllerutil.AddFinalizer(&realm, authRealmFinalizer)
		if err := r.Update(ctx, &realm); err != nil {
			return ctrl.Result{}, err
		}
	}

	if realm.Status.Phase == "" {
		realm.Status.Phase = "Provisioning"
		if err := r.Status().Update(ctx, &realm); err != nil {
			return ctrl.Result{}, err
		}
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       fmt.Sprintf("auth-%s", realm.Name),
		"app.kubernetes.io/component":  "auth",
		"app.kubernetes.io/managed-by": "zenith-operator",
		"zenith.dev/authrealm":         realm.Name,
	}

	// Step 1: CreateOrUpdate ConfigMap with realm configuration
	realmConfig := buildRealmConfigJSON(&realm)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("auth-realm-%s", realm.Name),
			Namespace: realm.Namespace,
		},
	}
	cmResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		configMap.Labels = labels
		configMap.Data = map[string]string{
			"realm.json": realmConfig,
		}
		return ctrl.SetControllerReference(&realm, configMap, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&realm, corev1.EventTypeWarning, "ConfigMapFailed", "Failed to ensure ConfigMap: %v", err)
		return ctrl.Result{}, err
	}
	if cmResult != controllerutil.OperationResultNone {
		logger.Info("ConfigMap reconciled", "operation", cmResult)
	}

	// Step 2: CreateOrUpdate Secret with provider client secrets
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("auth-realm-%s-secrets", realm.Name),
			Namespace: realm.Namespace,
		},
	}
	secretResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Type = corev1.SecretTypeOpaque

		// Collect secrets from provider ClientSecretRefs
		secretData := make(map[string][]byte)
		for _, provider := range realm.Spec.Providers {
			if provider.ClientSecretRef != nil {
				// Read the referenced secret
				refSecret := &corev1.Secret{}
				if err := r.Get(ctx, client.ObjectKey{
					Name:      provider.ClientSecretRef.Name,
					Namespace: realm.Namespace,
				}, refSecret); err != nil {
					logger.Error(err, "Failed to read provider secret", "provider", provider.Name, "secret", provider.ClientSecretRef.Name)
					continue
				}
				if val, ok := refSecret.Data[provider.ClientSecretRef.Key]; ok {
					secretData[fmt.Sprintf("provider-%s-client-secret", provider.Name)] = val
				}
			}
		}

		// Always ensure at least an empty data map
		if len(secretData) == 0 {
			secretData["placeholder"] = []byte("no-provider-secrets")
		}
		secret.Data = secretData

		return ctrl.SetControllerReference(&realm, secret, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&realm, corev1.EventTypeWarning, "SecretFailed", "Failed to ensure auth secrets: %v", err)
		return ctrl.Result{}, err
	}
	if secretResult != controllerutil.OperationResultNone {
		logger.Info("Auth secret reconciled", "operation", secretResult)
	}

	// Step 3: CreateOrUpdate Deployment for auth service replica
	authPort := int32(8080)
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("auth-%s", realm.Name),
			Namespace: realm.Namespace,
		},
	}
	depResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		deployment.Labels = labels
		deployment.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "auth",
							Image: "zenith-auth:latest",
							Ports: []corev1.ContainerPort{
								{ContainerPort: authPort, Protocol: corev1.ProtocolTCP},
							},
							Env: []corev1.EnvVar{
								{Name: "REALM_NAME", Value: realm.Name},
								{Name: "REALM_DISPLAY_NAME", Value: realm.Spec.DisplayName},
								{Name: "REALM_CONFIG_PATH", Value: "/etc/zenith-auth/realm.json"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/zenith-auth",
									ReadOnly:  true,
								},
								{
									Name:      "secrets",
									MountPath: "/etc/zenith-auth-secrets",
									ReadOnly:  true,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(authPort),
									},
								},
								PeriodSeconds:       10,
								InitialDelaySeconds: 5,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMap.Name,
									},
								},
							},
						},
						{
							Name: "secrets",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secret.Name,
								},
							},
						},
					},
				},
			},
		}
		return ctrl.SetControllerReference(&realm, deployment, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&realm, corev1.EventTypeWarning, "DeploymentFailed", "Failed to ensure auth deployment: %v", err)
		return ctrl.Result{}, err
	}
	if depResult != controllerutil.OperationResultNone {
		logger.Info("Auth deployment reconciled", "operation", depResult)
	}

	// Step 4: CreateOrUpdate Service for the auth deployment
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("auth-%s", realm.Name),
			Namespace: realm.Namespace,
		},
	}
	svcResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec = corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       authPort,
					TargetPort: intstr.FromInt32(authPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		}
		return ctrl.SetControllerReference(&realm, svc, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&realm, corev1.EventTypeWarning, "ServiceFailed", "Failed to ensure auth service: %v", err)
		return ctrl.Result{}, err
	}
	if svcResult != controllerutil.OperationResultNone {
		logger.Info("Auth service reconciled", "operation", svcResult)
	}

	// Step 5: CreateOrUpdate Ingress for auth endpoint
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("auth-%s", realm.Name),
			Namespace: realm.Namespace,
		},
	}
	ingressResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = labels
		ingress.Annotations = map[string]string{
			"kubernetes.io/ingress.class":                    "nginx",
			"nginx.ingress.kubernetes.io/rewrite-target":     "/$2",
			"nginx.ingress.kubernetes.io/ssl-redirect":       "true",
			"cert-manager.io/cluster-issuer":                 "letsencrypt-prod",
		}

		pathType := networkingv1.PathTypePrefix
		ingress.Spec = networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					Hosts:      []string{"auth.zenith.dev"},
					SecretName: fmt.Sprintf("auth-%s-tls", realm.Name),
				},
			},
			Rules: []networkingv1.IngressRule{
				{
					Host: "auth.zenith.dev",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     fmt.Sprintf("/realms/%s(/|$)(.*)", realm.Name),
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: svc.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: authPort,
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
		return ctrl.SetControllerReference(&realm, ingress, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&realm, corev1.EventTypeWarning, "IngressFailed", "Failed to ensure auth ingress: %v", err)
		return ctrl.Result{}, err
	}
	if ingressResult != controllerutil.OperationResultNone {
		logger.Info("Auth ingress reconciled", "operation", ingressResult)
	}

	// Update status
	realm.Status.Phase = "Ready"
	realm.Status.ClientCount = len(realm.Spec.Clients)
	realm.Status.Endpoint = fmt.Sprintf("https://auth.zenith.dev/realms/%s/.well-known/openid-configuration", realm.Name)

	if err := r.Status().Update(ctx, &realm); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&realm, corev1.EventTypeNormal, "Ready", "AuthRealm is ready")
	return ctrl.Result{}, nil
}

// buildRealmConfigJSON serializes the realm configuration to JSON for the ConfigMap.
func buildRealmConfigJSON(realm *zenithv1.AuthRealm) string {
	config := map[string]interface{}{
		"name":        realm.Name,
		"displayName": realm.Spec.DisplayName,
	}

	// Providers
	if len(realm.Spec.Providers) > 0 {
		providers := make([]map[string]interface{}, 0, len(realm.Spec.Providers))
		for _, p := range realm.Spec.Providers {
			provider := map[string]interface{}{
				"name":     p.Name,
				"type":     p.Type,
				"clientID": p.ClientID,
				"enabled":  p.Enabled,
			}
			if len(p.Config) > 0 {
				provider["config"] = p.Config
			}
			providers = append(providers, provider)
		}
		config["providers"] = providers
	}

	// Clients
	if len(realm.Spec.Clients) > 0 {
		clients := make([]map[string]interface{}, 0, len(realm.Spec.Clients))
		for _, c := range realm.Spec.Clients {
			authClient := map[string]interface{}{
				"name":         c.Name,
				"type":         c.Type,
				"redirectURIs": c.RedirectURIs,
				"scopes":       c.Scopes,
			}
			clients = append(clients, authClient)
		}
		config["clients"] = clients
	}

	// Settings
	if realm.Spec.Settings != nil {
		settings := map[string]interface{}{
			"mfaRequired":    realm.Spec.Settings.MFARequired,
			"sessionTimeout": realm.Spec.Settings.SessionTimeout,
		}
		if realm.Spec.Settings.PasswordPolicy != nil {
			settings["passwordPolicy"] = map[string]interface{}{
				"minLength":        realm.Spec.Settings.PasswordPolicy.MinLength,
				"requireUppercase": realm.Spec.Settings.PasswordPolicy.RequireUppercase,
				"requireNumbers":   realm.Spec.Settings.PasswordPolicy.RequireNumbers,
				"requireSpecial":   realm.Spec.Settings.PasswordPolicy.RequireSpecial,
			}
		}
		config["settings"] = settings
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (r *AuthRealmReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.AuthRealm{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Named("authrealm").
		Complete(r)
}

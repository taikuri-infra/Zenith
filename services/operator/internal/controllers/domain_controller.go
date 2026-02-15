package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	zenithv1 "github.com/dotechhq/zenith/services/operator/api/v1alpha1"
	"github.com/dotechhq/zenith/services/operator/internal/provider/hetzner"
)

const (
	domainFinalizer       = "zenith.dev/domain-cleanup"
	domainDNSRecordIDAnno = "zenith.dev/dns-record-id"
	domainDNSZoneIDAnno   = "zenith.dev/dns-zone-id"
)

type DomainReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Hetzner  *hetzner.Client
}

func NewDomainReconciler(c client.Client, s *runtime.Scheme, r record.EventRecorder, h *hetzner.Client) *DomainReconciler {
	return &DomainReconciler{Client: c, Scheme: s, Recorder: r, Hetzner: h}
}

func (r *DomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var domain zenithv1.Domain
	if err := r.Get(ctx, req.NamespacedName, &domain); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !domain.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&domain, domainFinalizer) {
			logger.Info("Cleaning up domain resources", "name", domain.Name)

			// Delete DNS record via Hetzner if one was created
			if r.Hetzner.IsConfigured() {
				if recordID, ok := domain.Annotations[domainDNSRecordIDAnno]; ok && recordID != "" {
					if err := r.Hetzner.DeleteDNSRecord(ctx, recordID); err != nil {
						logger.Error(err, "Failed to delete DNS record", "recordID", recordID)
						// Continue anyway - DNS record deletion is best-effort
					} else {
						r.Recorder.Event(&domain, corev1.EventTypeNormal, "DNSRecordDeleted", "Deleted DNS record")
					}
				}
			}

			// cert-manager Certificate and Ingress are cleaned up by K8s cascade via owner references
			controllerutil.RemoveFinalizer(&domain, domainFinalizer)
			if err := r.Update(ctx, &domain); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&domain, domainFinalizer) {
		controllerutil.AddFinalizer(&domain, domainFinalizer)
		if err := r.Update(ctx, &domain); err != nil {
			return ctrl.Result{}, err
		}
	}

	if domain.Status.Phase == "" {
		domain.Status.Phase = "Configuring"
		if err := r.Status().Update(ctx, &domain); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Step 1: Look up the referenced App to get its Service
	var app zenithv1.App
	if err := r.Get(ctx, types.NamespacedName{Name: domain.Spec.AppRef, Namespace: domain.Namespace}, &app); err != nil {
		logger.Error(err, "Failed to find referenced App", "appRef", domain.Spec.AppRef)
		r.Recorder.Eventf(&domain, corev1.EventTypeWarning, "AppNotFound", "Referenced app %s not found", domain.Spec.AppRef)
		domain.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, &domain)
		return ctrl.Result{}, err
	}

	appPort := app.Spec.Port
	if appPort == 0 {
		appPort = 8080
	}

	// Step 2: If DNS auto-configure and Hetzner configured, create DNS record
	if domain.Spec.DNS != nil && domain.Spec.DNS.AutoConfigure && r.Hetzner.IsConfigured() {
		// Only create if not already created (check annotation)
		if domain.Annotations == nil {
			domain.Annotations = make(map[string]string)
		}
		if _, exists := domain.Annotations[domainDNSRecordIDAnno]; !exists {
			recordType := "A"
			if domain.Spec.DNS.Type != "" {
				recordType = domain.Spec.DNS.Type
			}

			// Use a well-known zone ID - in production this would be looked up
			zoneID := "zenith-dns-zone"
			record, err := r.Hetzner.CreateDNSRecord(ctx, zoneID, recordType, domain.Spec.Domain, "0.0.0.0", 300)
			if err != nil {
				r.Recorder.Eventf(&domain, corev1.EventTypeWarning, "DNSFailed", "Failed to create DNS record: %v", err)
				// Don't fail the whole reconcile - DNS can be retried
				logger.Error(err, "Failed to create DNS record")
			} else {
				domain.Annotations[domainDNSRecordIDAnno] = record.ID
				domain.Annotations[domainDNSZoneIDAnno] = zoneID
				if err := r.Update(ctx, &domain); err != nil {
					return ctrl.Result{}, err
				}
				domain.Status.DNSConfigured = true
				r.Recorder.Event(&domain, corev1.EventTypeNormal, "DNSConfigured", "DNS record created")
			}
		} else {
			domain.Status.DNSConfigured = true
		}
	}

	// Step 3: If SSL enabled, CreateOrUpdate cert-manager Certificate (unstructured)
	sslEnabled := domain.Spec.SSL == nil || domain.Spec.SSL.Enabled
	issuer := "letsencrypt-prod"
	if domain.Spec.SSL != nil && domain.Spec.SSL.Issuer != "" {
		issuer = domain.Spec.SSL.Issuer
	}
	tlsSecretName := fmt.Sprintf("%s-tls", domain.Name)

	if sslEnabled {
		cert := &unstructured.Unstructured{}
		cert.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "cert-manager.io",
			Version: "v1",
			Kind:    "Certificate",
		})
		cert.SetName(domain.Name)
		cert.SetNamespace(domain.Namespace)

		certResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, cert, func() error {
			cert.Object["spec"] = map[string]interface{}{
				"secretName": tlsSecretName,
				"issuerRef": map[string]interface{}{
					"name": issuer,
					"kind": "ClusterIssuer",
				},
				"dnsNames": []interface{}{domain.Spec.Domain},
			}
			// Set owner reference
			return ctrl.SetControllerReference(&domain, cert, r.Scheme)
		})
		if err != nil {
			// cert-manager CRD may not be installed - log warning but continue
			logger.Error(err, "Failed to ensure cert-manager Certificate - cert-manager may not be installed")
			r.Recorder.Eventf(&domain, corev1.EventTypeWarning, "CertificateFailed", "Failed to ensure Certificate: %v", err)
		} else {
			if certResult != controllerutil.OperationResultNone {
				logger.Info("Certificate reconciled", "operation", certResult)
			}
			domain.Status.SSLReady = true
		}
	}

	// Step 4: CreateOrUpdate Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("domain-%s", domain.Name),
			Namespace: domain.Namespace,
		},
	}
	ingressResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
		ingress.Labels = map[string]string{
			"app.kubernetes.io/managed-by": "zenith-operator",
			"zenith.dev/domain":            domain.Name,
		}
		ingress.Annotations = map[string]string{
			"kubernetes.io/ingress.class":               "nginx",
			"nginx.ingress.kubernetes.io/ssl-redirect":  "true",
		}
		if sslEnabled {
			ingress.Annotations["cert-manager.io/cluster-issuer"] = issuer
		}

		pathType := networkingv1.PathTypePrefix
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: domain.Spec.Domain,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: domain.Spec.AppRef,
											Port: networkingv1.ServiceBackendPort{
												Number: appPort,
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

		if sslEnabled {
			ingress.Spec.TLS = []networkingv1.IngressTLS{
				{
					Hosts:      []string{domain.Spec.Domain},
					SecretName: tlsSecretName,
				},
			}
		}

		return ctrl.SetControllerReference(&domain, ingress, r.Scheme)
	})
	if err != nil {
		r.Recorder.Eventf(&domain, corev1.EventTypeWarning, "IngressFailed", "Failed to ensure Ingress: %v", err)
		return ctrl.Result{}, err
	}
	if ingressResult != controllerutil.OperationResultNone {
		logger.Info("Ingress reconciled", "operation", ingressResult)
	}

	// Update status
	domain.Status.Phase = "Active"
	if err := r.Status().Update(ctx, &domain); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&domain, corev1.EventTypeNormal, "Active", "Domain is active")
	return ctrl.Result{}, nil
}

func (r *DomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zenithv1.Domain{}).
		Owns(&networkingv1.Ingress{}).
		Named("domain").
		Complete(r)
}

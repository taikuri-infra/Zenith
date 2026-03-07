package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

// PgwebSession represents a running pgweb explorer session.
type PgwebSession struct {
	DatabaseID string    `json:"database_id"`
	UserID     string    `json:"user_id"`
	Token      string    `json:"token"`
	URL        string    `json:"url"`
	ReadOnly   bool      `json:"readonly"`
	Status     string    `json:"status"` // "starting", "running"
	CreatedAt  time.Time `json:"created_at"`
}

// PgwebService manages on-demand pgweb pods for database exploration.
type PgwebService struct {
	dbRepo     ports.DatabaseRepository
	k8sClient  k8sclient.Client
	namespace  string // K8s namespace for pgweb pods (e.g. "zenith-staging")
	baseDomain string // e.g. "apps.stage.freezenith.com"
	sessions   sync.Map
}

// NewPgwebService creates a new PgwebService.
func NewPgwebService(
	dbRepo ports.DatabaseRepository,
	k8sClient k8sclient.Client,
	namespace string,
	baseDomain string,
) *PgwebService {
	return &PgwebService{
		dbRepo:     dbRepo,
		k8sClient:  k8sClient,
		namespace:  namespace,
		baseDomain: baseDomain,
	}
}

// StartSession creates a pgweb Deployment + Service + IngressRoute for exploring a database.
func (s *PgwebService) StartSession(ctx context.Context, dbID, userID string, readOnly bool) (*PgwebSession, error) {
	// Check for existing session
	if existing, ok := s.sessions.Load(dbID); ok {
		return existing.(*PgwebSession), nil
	}

	// Verify database exists and is PostgreSQL
	db, err := s.dbRepo.GetDatabase(ctx, dbID)
	if err != nil {
		return nil, fmt.Errorf("database not found: %w", err)
	}
	if db.Engine != "postgresql" {
		return nil, fmt.Errorf("explorer only supports PostgreSQL databases")
	}

	// Generate random token for subdomain
	tokenBytes := make([]byte, 6)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	session := &PgwebSession{
		DatabaseID: dbID,
		UserID:     userID,
		Token:      token,
		URL:        "https://pgweb-" + token + "." + s.baseDomain,
		ReadOnly:   readOnly,
		Status:     "starting",
		CreatedAt:  time.Now(),
	}

	// Create K8s resources
	resourceName := "pgweb-" + token
	secretName := "db-" + dbID[:8] + "-credentials"
	labels := map[string]string{
		"zenith.io/pgweb":    "true",
		"zenith.io/database": dbID,
		"zenith.io/user":     userID,
	}

	// 1. Deployment
	if err := s.createDeployment(ctx, resourceName, secretName, labels, readOnly); err != nil {
		return nil, fmt.Errorf("create pgweb deployment: %w", err)
	}

	// 2. Service
	if err := s.createService(ctx, resourceName, labels); err != nil {
		// Cleanup deployment on failure
		s.k8sClient.DeleteCRD(ctx, "Deployment", s.namespace, resourceName)
		return nil, fmt.Errorf("create pgweb service: %w", err)
	}

	// 3. IngressRoute
	if err := s.createIngressRoute(ctx, resourceName, labels); err != nil {
		s.k8sClient.DeleteCRD(ctx, "Service", s.namespace, resourceName)
		s.k8sClient.DeleteCRD(ctx, "Deployment", s.namespace, resourceName)
		return nil, fmt.Errorf("create pgweb ingress: %w", err)
	}

	session.Status = "running"
	s.sessions.Store(dbID, session)
	_ = db // silence unused

	log.Printf("[pgweb] Started session for database %s (token: %s, readonly: %v)", dbID, token, readOnly)
	return session, nil
}

// GetSession returns an existing pgweb session for a database.
func (s *PgwebService) GetSession(ctx context.Context, dbID string) (*PgwebSession, error) {
	if session, ok := s.sessions.Load(dbID); ok {
		return session.(*PgwebSession), nil
	}
	return nil, fmt.Errorf("no active explorer session")
}

// StopSession deletes all K8s resources for a pgweb session.
func (s *PgwebService) StopSession(ctx context.Context, dbID string) error {
	sessionVal, ok := s.sessions.Load(dbID)
	if !ok {
		return fmt.Errorf("no active explorer session")
	}
	session := sessionVal.(*PgwebSession)
	resourceName := "pgweb-" + session.Token

	// Delete in reverse order: IngressRoute -> Service -> Deployment
	if err := s.k8sClient.DeleteCRD(ctx, "IngressRoute", s.namespace, resourceName); err != nil {
		log.Printf("[pgweb] Warning: failed to delete IngressRoute %s: %v", resourceName, err)
	}
	if err := s.k8sClient.DeleteCRD(ctx, "Service", s.namespace, resourceName); err != nil {
		log.Printf("[pgweb] Warning: failed to delete Service %s: %v", resourceName, err)
	}
	if err := s.k8sClient.DeleteCRD(ctx, "Deployment", s.namespace, resourceName); err != nil {
		log.Printf("[pgweb] Warning: failed to delete Deployment %s: %v", resourceName, err)
	}

	s.sessions.Delete(dbID)
	log.Printf("[pgweb] Stopped session for database %s (token: %s)", dbID, session.Token)
	return nil
}

// CleanupExpired runs periodically to kill sessions older than 30 minutes.
func (s *PgwebService) CleanupExpired(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sessions.Range(func(key, value interface{}) bool {
				session := value.(*PgwebSession)
				if time.Since(session.CreatedAt) > 30*time.Minute {
					log.Printf("[pgweb] Auto-expiring session for database %s (age: %s)", session.DatabaseID, time.Since(session.CreatedAt))
					s.StopSession(ctx, session.DatabaseID)
				}
				return true
			})
		}
	}
}

// createDeployment creates a pgweb Deployment that reads DATABASE_URL from the existing credentials secret.
func (s *PgwebService) createDeployment(ctx context.Context, name, secretName string, labels map[string]string, readOnly bool) error {
	args := []string{"--bind=0.0.0.0", "--listen=8081"}
	if readOnly {
		args = append(args, "--readonly")
	}

	argsJSON, _ := json.Marshal(args)

	spec := map[string]interface{}{
		"replicas": 1,
		"selector": map[string]interface{}{
			"matchLabels": map[string]string{
				"app": name,
			},
		},
		"template": map[string]interface{}{
			"metadata": map[string]interface{}{
				"labels": func() map[string]string {
					l := make(map[string]string, len(labels)+1)
					for k, v := range labels {
						l[k] = v
					}
					l["app"] = name
					return l
				}(),
			},
			"spec": map[string]interface{}{
				"containers": []map[string]interface{}{
					{
						"name":  "pgweb",
						"image": "sosedoff/pgweb:latest",
						"args":  json.RawMessage(argsJSON),
						"ports": []map[string]interface{}{
							{
								"containerPort": 8081,
								"protocol":      "TCP",
							},
						},
						"env": []map[string]interface{}{
							{
								"name": "DATABASE_URL",
								"valueFrom": map[string]interface{}{
									"secretKeyRef": map[string]interface{}{
										"name": secretName,
										"key":  "DATABASE_URL",
									},
								},
							},
						},
						"resources": map[string]interface{}{
							"limits": map[string]string{
								"cpu":    "100m",
								"memory": "128Mi",
							},
							"requests": map[string]string{
								"cpu":    "50m",
								"memory": "64Mi",
							},
						},
					},
				},
			},
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	return s.k8sClient.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
			Labels:    labels,
		},
		Spec: specJSON,
	})
}

// createService creates a ClusterIP Service for the pgweb pod.
func (s *PgwebService) createService(ctx context.Context, name string, labels map[string]string) error {
	spec := map[string]interface{}{
		"selector": map[string]string{
			"app": name,
		},
		"ports": []map[string]interface{}{
			{
				"port":       80,
				"targetPort": 8081,
				"protocol":   "TCP",
			},
		},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	return s.k8sClient.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: "v1",
		Kind:       "Service",
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
			Labels:    labels,
		},
		Spec: specJSON,
	})
}

// createIngressRoute creates a Traefik IngressRoute for the pgweb session.
func (s *PgwebService) createIngressRoute(ctx context.Context, name string, labels map[string]string) error {
	host := name + "." + s.baseDomain

	spec := map[string]interface{}{
		"entryPoints": []string{"websecure"},
		"routes": []map[string]interface{}{
			{
				"match": fmt.Sprintf("Host(`%s`)", host),
				"kind":  "Rule",
				"services": []map[string]interface{}{
					{
						"name": name,
						"port": 80,
					},
				},
			},
		},
		"tls": map[string]interface{}{},
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	return s.k8sClient.CreateCRD(ctx, &k8sclient.CRDObject{
		APIVersion: "traefik.io/v1alpha1",
		Kind:       "IngressRoute",
		Metadata: k8sclient.ObjectMeta{
			Name:      name,
			Namespace: s.namespace,
			Labels:    labels,
		},
		Spec: specJSON,
	})
}

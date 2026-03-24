package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDeployTokenCreate(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, err := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token",
		[]string{string(entities.ScopeDeployStaging)}, nil)
	if err != nil {
		t.Fatalf("CreateDeployToken failed: %v", err)
	}
	if dt.ID == "" {
		t.Error("Expected ID to be set")
	}
	if dt.TokenID == "" || dt.TokenID[:7] != "znt_id_" {
		t.Errorf("Expected token ID with znt_id_ prefix, got '%s'", dt.TokenID)
	}
	if dt.Secret == "" || dt.Secret[:7] != "znt_sk_" {
		t.Errorf("Expected secret with znt_sk_ prefix, got '%s'", dt.Secret)
	}
	if dt.TokenHash == "" {
		t.Error("Expected token hash to be set")
	}
	if dt.ProjectID != "proj-1" {
		t.Errorf("Expected project ID 'proj-1', got '%s'", dt.ProjectID)
	}
	if dt.Name != "CI Token" {
		t.Errorf("Expected name 'CI Token', got '%s'", dt.Name)
	}
}

func TestDeployTokenGet(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	created, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)

	dt, err := repo.GetDeployToken(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetDeployToken failed: %v", err)
	}
	if dt.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, dt.ID)
	}
}

func TestDeployTokenGetNotFound(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	_, err := repo.GetDeployToken(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent token")
	}
}

func TestDeployTokenGetByTokenID(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	created, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)

	dt, err := repo.GetDeployTokenByTokenID(ctx, created.TokenID)
	if err != nil {
		t.Fatalf("GetDeployTokenByTokenID failed: %v", err)
	}
	if dt.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, dt.ID)
	}
}

func TestDeployTokenGetByTokenIDNotFound(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	_, err := repo.GetDeployTokenByTokenID(ctx, "znt_id_nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent token ID")
	}
}

func TestDeployTokenListByProject(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	repo.CreateDeployToken(ctx, "user-1", "proj-1", "Token A", nil, nil)
	repo.CreateDeployToken(ctx, "user-1", "proj-1", "Token B", nil, nil)
	repo.CreateDeployToken(ctx, "user-1", "proj-2", "Token C", nil, nil)

	tokens, err := repo.ListDeployTokensByProject(ctx, "proj-1")
	if err != nil {
		t.Fatalf("ListDeployTokensByProject failed: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens for proj-1, got %d", len(tokens))
	}
}

func TestDeployTokenListExcludesRevoked(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "Token A", nil, nil)
	repo.CreateDeployToken(ctx, "user-1", "proj-1", "Token B", nil, nil)
	repo.RevokeDeployToken(ctx, dt.ID)

	tokens, _ := repo.ListDeployTokensByProject(ctx, "proj-1")
	if len(tokens) != 1 {
		t.Errorf("Expected 1 active token (revoked excluded), got %d", len(tokens))
	}
}

func TestDeployTokenRevoke(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)

	err := repo.RevokeDeployToken(ctx, dt.ID)
	if err != nil {
		t.Fatalf("RevokeDeployToken failed: %v", err)
	}

	fetched, _ := repo.GetDeployToken(ctx, dt.ID)
	if fetched.RevokedAt == nil {
		t.Error("Expected RevokedAt to be set")
	}
	if !fetched.IsRevoked() {
		t.Error("Expected IsRevoked() to return true")
	}
}

func TestDeployTokenRevokeAlreadyRevoked(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)
	repo.RevokeDeployToken(ctx, dt.ID)

	err := repo.RevokeDeployToken(ctx, dt.ID)
	if err == nil {
		t.Error("Expected error when revoking an already-revoked token")
	}
}

func TestDeployTokenRevokeNotFound(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	err := repo.RevokeDeployToken(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent token")
	}
}

func TestDeployTokenRotate(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)
	oldHash := dt.TokenHash
	oldSecret := dt.Secret

	rotated, err := repo.RotateDeployToken(ctx, dt.ID)
	if err != nil {
		t.Fatalf("RotateDeployToken failed: %v", err)
	}
	if rotated.TokenHash == oldHash {
		t.Error("Expected new token hash after rotation")
	}
	if rotated.Secret == oldSecret {
		t.Error("Expected new secret after rotation")
	}
	if rotated.PreviousHash != oldHash {
		t.Error("Expected previous hash to be set to old hash")
	}
	if rotated.PreviousExpiresAt == nil {
		t.Error("Expected PreviousExpiresAt to be set for grace period")
	}
	if !rotated.InGracePeriod() {
		t.Error("Expected token to be in grace period after rotation")
	}
}

func TestDeployTokenRotateNotFound(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	_, err := repo.RotateDeployToken(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent token")
	}
}

func TestDeployTokenVerifySecret(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)
	secret := dt.Secret

	if !repo.VerifySecret(dt, secret) {
		t.Error("Expected VerifySecret to return true for correct secret")
	}
	if repo.VerifySecret(dt, "wrong-secret") {
		t.Error("Expected VerifySecret to return false for wrong secret")
	}
}

func TestDeployTokenUpdateLastUsed(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "CI Token", nil, nil)
	if dt.LastUsedAt != nil {
		t.Error("Expected LastUsedAt to be nil initially")
	}

	err := repo.UpdateLastUsed(ctx, dt.ID)
	if err != nil {
		t.Fatalf("UpdateLastUsed failed: %v", err)
	}

	fetched, _ := repo.GetDeployToken(ctx, dt.ID)
	if fetched.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set after UpdateLastUsed")
	}
}

func TestDeployTokenExpiry(t *testing.T) {
	repo := NewMemoryDeployTokenRepository()
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	dt, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "Expired Token", nil, &past)

	if !dt.IsExpired() {
		t.Error("Expected expired token to return IsExpired() = true")
	}

	future := time.Now().Add(24 * time.Hour)
	dt2, _ := repo.CreateDeployToken(ctx, "user-1", "proj-1", "Valid Token", nil, &future)
	if dt2.IsExpired() {
		t.Error("Expected non-expired token to return IsExpired() = false")
	}
}

func TestDeployTokenHasScope(t *testing.T) {
	dt := &entities.DeployToken{
		Scopes: []string{
			string(entities.ScopeDeployStaging),
			string(entities.ScopeAppRead),
		},
	}

	if !dt.HasScope(string(entities.ScopeDeployStaging)) {
		t.Error("Expected HasScope to return true for ScopeDeployStaging")
	}
	if !dt.HasScope(string(entities.ScopeAppRead)) {
		t.Error("Expected HasScope to return true for ScopeAppRead")
	}
	if dt.HasScope(string(entities.ScopeDeployProduction)) {
		t.Error("Expected HasScope to return false for ScopeDeployProduction")
	}
}

func TestDeployTokenInfraAllScope(t *testing.T) {
	dt := &entities.DeployToken{
		Scopes: []string{string(entities.ScopeInfraAll)},
	}

	// infra:* should grant all permissions
	if !dt.HasScope(string(entities.ScopeDeployStaging)) {
		t.Error("Expected infra:* to grant deploy:staging")
	}
	if !dt.HasScope(string(entities.ScopeDeployProduction)) {
		t.Error("Expected infra:* to grant deploy:production")
	}
	if !dt.HasScope(string(entities.ScopeLogsRead)) {
		t.Error("Expected infra:* to grant logs:read")
	}
}

func TestDeployTokenValidScope(t *testing.T) {
	if !entities.ValidDeployTokenScope(string(entities.ScopeDeployStaging)) {
		t.Error("Expected deploy:staging to be valid")
	}
	if entities.ValidDeployTokenScope("invalid:scope") {
		t.Error("Expected invalid:scope to be invalid")
	}
	if entities.ValidDeployTokenScope("") {
		t.Error("Expected empty string to be invalid")
	}
}

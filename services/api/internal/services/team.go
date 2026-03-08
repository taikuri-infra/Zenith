package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	zenithJWT "github.com/dotechhq/zenith/services/api/pkg/jwt"
	"github.com/google/uuid"
)

const inviteTokenExpiry = 7 * 24 * time.Hour

// TeamMemberService handles team member business logic.
type TeamMemberService struct {
	teamRepo    ports.TeamMemberRepository
	userRepo    ports.UserRepository
	planRepo    ports.UserPlanRepository
	jwtSecret   string
	emailSender ports.EmailSender
	appURL      string
}

// NewTeamMemberService creates a new TeamMemberService.
func NewTeamMemberService(teamRepo ports.TeamMemberRepository, userRepo ports.UserRepository, planRepo ports.UserPlanRepository, jwtSecret string) *TeamMemberService {
	return &TeamMemberService{
		teamRepo:  teamRepo,
		userRepo:  userRepo,
		planRepo:  planRepo,
		jwtSecret: jwtSecret,
	}
}

// SetEmailSender configures the email sender for invite emails.
func (s *TeamMemberService) SetEmailSender(sender ports.EmailSender, appURL string) {
	s.emailSender = sender
	s.appURL = appURL
}

// InviteMember invites a user by email to join the owner's account.
func (s *TeamMemberService) InviteMember(ctx context.Context, accountID, email string, role entities.Role) (*entities.TeamMember, error) {
	// Validate role — cannot invite as owner
	if role == entities.RoleOwner || role == entities.RoleCustomer {
		return nil, fmt.Errorf("invalid role: %s", role)
	}
	if !role.IsValid() {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Check plan limit
	plan, err := s.planRepo.GetUserPlan(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	count, err := s.teamRepo.CountMembers(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}
	// +1 because the owner counts as a member
	if count+1 >= plan.Limits.MaxTeamMembers {
		return nil, fmt.Errorf("team member limit reached (%d/%d). Upgrade your plan for more.", count+1, plan.Limits.MaxTeamMembers)
	}

	// Check for duplicate
	existing, _ := s.teamRepo.GetMemberByEmail(ctx, accountID, email)
	if existing != nil {
		return nil, fmt.Errorf("member with email %s already exists", email)
	}

	// Generate invite token
	rawToken, err := generateInviteToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}
	tokenHash := hashInviteToken(rawToken)
	expiresAt := time.Now().Add(inviteTokenExpiry)

	now := time.Now()
	member := &entities.TeamMember{
		ID:              uuid.New().String(),
		AccountID:       accountID,
		Email:           email,
		Role:            role,
		Status:          entities.TeamMemberPending,
		InviteTokenHash: tokenHash,
		InviteExpiresAt: &expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.teamRepo.CreateMember(ctx, member); err != nil {
		return nil, err
	}

	// Send invite email
	if s.emailSender != nil && s.appURL != "" {
		inviteURL := s.appURL + "/invite?token=" + rawToken
		owner, _ := s.userRepo.GetByID(ctx, accountID)
		inviterName := "Team owner"
		teamName := "their team"
		if owner != nil {
			inviterName = owner.Name
			teamName = owner.Name + "'s team"
		}
		if err := s.emailSender.SendTeamInviteEmail(ctx, email, inviterName, teamName, inviteURL); err != nil {
			fmt.Printf("[team] failed to send invite email to %s: %v\n", email, err)
		}
	}

	return member, nil
}

// AcceptInvite accepts a team invite and returns a token pair.
func (s *TeamMemberService) AcceptInvite(ctx context.Context, rawToken, email, password, name string) (*TokenPair, error) {
	tokenHash := hashInviteToken(rawToken)

	member, err := s.teamRepo.GetMemberByInviteHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired invite")
	}

	if member.Status != entities.TeamMemberPending {
		return nil, fmt.Errorf("invite already accepted or expired")
	}

	if member.InviteExpiresAt != nil && time.Now().After(*member.InviteExpiresAt) {
		return nil, fmt.Errorf("invite has expired")
	}

	// Find or create user account
	var user *ports.StoredUser
	user, err = s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Create new user
		if name == "" {
			name = email
		}
		newUser, createErr := s.userRepo.Create(ctx, email, password, name, entities.RoleCustomer)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create user: %w", createErr)
		}
		// Mark email as verified (invite link is proof)
		_ = s.userRepo.SetEmailVerified(ctx, newUser.ID)
		newUser.EmailVerified = true
		user = &ports.StoredUser{User: *newUser}
	}

	// Update member
	member.UserID = user.ID
	member.Status = entities.TeamMemberActive
	member.InviteTokenHash = ""
	member.InviteExpiresAt = nil
	member.UpdatedAt = time.Now()

	if err := s.teamRepo.UpdateMember(ctx, member); err != nil {
		return nil, fmt.Errorf("failed to accept invite: %w", err)
	}

	// Issue tokens with AccountID set
	return s.issueTeamTokens(&user.User, member)
}

// ListMembers returns all team members for the account.
func (s *TeamMemberService) ListMembers(ctx context.Context, accountID string) ([]entities.TeamMember, error) {
	return s.teamRepo.ListMembers(ctx, accountID)
}

// UpdateMemberRole changes a member's role.
func (s *TeamMemberService) UpdateMemberRole(ctx context.Context, accountID, memberID string, newRole entities.Role) error {
	if newRole == entities.RoleOwner || newRole == entities.RoleCustomer {
		return fmt.Errorf("invalid role: %s", newRole)
	}
	if !newRole.IsValid() {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	member, err := s.teamRepo.GetMember(ctx, memberID)
	if err != nil {
		return err
	}
	if member.AccountID != accountID {
		return fmt.Errorf("member does not belong to this account")
	}

	member.Role = newRole
	member.UpdatedAt = time.Now()
	return s.teamRepo.UpdateMember(ctx, member)
}

// RemoveMember removes a team member.
func (s *TeamMemberService) RemoveMember(ctx context.Context, accountID, memberID string) error {
	member, err := s.teamRepo.GetMember(ctx, memberID)
	if err != nil {
		return err
	}
	if member.AccountID != accountID {
		return fmt.Errorf("member does not belong to this account")
	}

	return s.teamRepo.DeleteMember(ctx, memberID)
}

// GetMemberForLogin checks if a user is an active team member and returns the membership.
func (s *TeamMemberService) GetMemberForLogin(ctx context.Context, userID string) (*entities.TeamMember, error) {
	return s.teamRepo.GetMemberByUserID(ctx, userID)
}

func (s *TeamMemberService) issueTeamTokens(user *entities.User, member *entities.TeamMember) (*TokenPair, error) {
	overrides := zenithJWT.TeamMemberOverrides{
		AccountID: member.AccountID,
		MemberID:  member.ID,
		Role:      member.Role,
	}

	accessToken, err := zenithJWT.GenerateTeamMemberToken(s.jwtSecret, user, AccessTokenExpiry, overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token")
	}

	refreshToken, err := zenithJWT.GenerateTeamMemberToken(s.jwtSecret, user, RefreshTokenExpiry, overrides)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token")
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(AccessTokenExpiry.Seconds()),
	}, nil
}

func generateInviteToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashInviteToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

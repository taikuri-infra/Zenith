package providers

// Provider defines the interface for identity providers (Google, GitHub, etc.)
type Provider interface {
	// Name returns the provider identifier
	Name() string
	// AuthURL returns the OAuth2 authorization URL
	AuthURL(state, redirectURI string) string
	// Exchange exchanges an auth code for user info
	Exchange(code, redirectURI string) (*UserInfo, error)
}

type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	AvatarURL     string `json:"avatar_url"`
	Provider      string `json:"provider"`
}

// GoogleProvider implements OAuth2 for Google
type GoogleProvider struct {
	ClientID     string
	ClientSecret string
}

func NewGoogleProvider(clientID, clientSecret string) *GoogleProvider {
	return &GoogleProvider{ClientID: clientID, ClientSecret: clientSecret}
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) AuthURL(state, redirectURI string) string {
	return "https://accounts.google.com/o/oauth2/v2/auth" +
		"?client_id=" + p.ClientID +
		"&redirect_uri=" + redirectURI +
		"&response_type=code" +
		"&scope=openid+email+profile" +
		"&state=" + state
}

func (p *GoogleProvider) Exchange(code, redirectURI string) (*UserInfo, error) {
	// TODO: Implement real Google OAuth2 exchange
	return &UserInfo{Provider: "google"}, nil
}

// GitHubProvider implements OAuth2 for GitHub
type GitHubProvider struct {
	ClientID     string
	ClientSecret string
}

func NewGitHubProvider(clientID, clientSecret string) *GitHubProvider {
	return &GitHubProvider{ClientID: clientID, ClientSecret: clientSecret}
}

func (p *GitHubProvider) Name() string { return "github" }

func (p *GitHubProvider) AuthURL(state, redirectURI string) string {
	return "https://github.com/login/oauth/authorize" +
		"?client_id=" + p.ClientID +
		"&redirect_uri=" + redirectURI +
		"&scope=user:email" +
		"&state=" + state
}

func (p *GitHubProvider) Exchange(code, redirectURI string) (*UserInfo, error) {
	// TODO: Implement real GitHub OAuth2 exchange
	return &UserInfo{Provider: "github"}, nil
}

package github

import (
	"errors"
	"net/http"
	"time"

	"github.com/nsheridan/cashier/server/auth"
	"github.com/nsheridan/cashier/server/config"

	githubapi "github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	name = "github"
)

// Config is an implementation of `auth.Provider` for authenticating using a
// Github account.
type Config struct {
	config       *oauth2.Config
	organization string
	whitelist    map[string]bool
}

// New creates a new Github provider from a configuration.
func New(c *config.Auth) (auth.Provider, error) {
	uw := make(map[string]bool)
	for _, u := range c.UsersWhitelist {
		uw[u] = true
	}
	if c.ProviderOpts["organization"] == "" && len(uw) == 0 {
		return nil, errors.New("github_opts organization and the users whitelist must not be both empty")
	}
	return &Config{
		config: &oauth2.Config{
			ClientID:     c.OauthClientID,
			ClientSecret: c.OauthClientSecret,
			RedirectURL:  c.OauthCallbackURL,
			Endpoint:     github.Endpoint,
			Scopes: []string{
				string(githubapi.ScopeUser),
				string(githubapi.ScopeReadOrg),
			},
		},
		organization: c.ProviderOpts["organization"],
		whitelist:    uw,
	}, nil
}

// A new oauth2 http client.
func (c *Config) newClient(token *oauth2.Token) *http.Client {
	return c.config.Client(oauth2.NoContext, token)
}

// Name returns the name of the provider.
func (c *Config) Name() string {
	return name
}

// Valid validates the oauth token.
func (c *Config) Valid(token *oauth2.Token) bool {
	if len(c.whitelist) > 0 && !c.whitelist[c.Username(token)] {
		return false
	}
	if !token.Valid() {
		return false
	}
	if c.organization == "" {
		// There's no organization and the token is valid. Can only reach here
		// if there's a user whitelist set and the user is in the whitelist.
		return true
	}
	client := githubapi.NewClient(c.newClient(token))
	member, _, err := client.Organizations.IsMember(c.organization, c.Username(token))
	if err != nil {
		return false
	}
	return member
}

// Revoke is a no-op revoke method. GitHub doesn't seem to allow token
// revocation - tokens are indefinite and there are no refresh options etc.
// Returns nil to satisfy the Provider interface.
func (c *Config) Revoke(token *oauth2.Token) error {
	return nil
}

// StartSession retrieves an authentication endpoint from Github.
func (c *Config) StartSession(state string) *auth.Session {
	return &auth.Session{
		AuthURL: c.config.AuthCodeURL(state),
	}
}

// Exchange authorizes the session and returns an access token.
func (c *Config) Exchange(code string) (*oauth2.Token, error) {
	t, err := c.config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}
	// Github tokens don't have an expiry. Set one so that the session expires
	// after a period.
	if t.Expiry.Unix() <= 0 {
		t.Expiry = time.Now().Add(1 * time.Hour)
	}
	return t, nil
}

// Username retrieves the username portion of the user's email address.
func (c *Config) Username(token *oauth2.Token) string {
	client := githubapi.NewClient(c.newClient(token))
	u, _, err := client.Users.Get("")
	if err != nil {
		return ""
	}
	return *u.Login
}

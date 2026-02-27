package qbo

import "golang.org/x/oauth2"

const (
	authURL  = "https://appcenter.intuit.com/connect/oauth2"
	tokenURL = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer"
	scope    = "com.intuit.quickbooks.accounting"
)

// NewOAuthConfig returns an oauth2.Config configured for QuickBooks Online.
func NewOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{scope},
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}
}

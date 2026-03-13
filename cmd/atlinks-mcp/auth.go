package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type tokenFile struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ServerURL    string `json:"server_url"`
}

func tokenFilePath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "atlinks-mcp", "token.json")
}

func loadTokens() (*tokenFile, error) {
	data, err := os.ReadFile(tokenFilePath())
	if err != nil {
		return nil, err
	}
	var tf tokenFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, err
	}
	return &tf, nil
}

func saveTokens(tf *tokenFile) error {
	path := tokenFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// isTokenExpired checks if the JWT access token is expired or about to expire (within 5 min).
func isTokenExpired(accessToken string) bool {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(accessToken, jwt.MapClaims{})
	if err != nil {
		return true
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return true
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return true
	}
	return time.Now().Add(5 * time.Minute).After(exp.Time)
}

// refreshAccessToken uses the refresh_token to get a new access_token.
func refreshAccessToken(serverURL, refreshToken string) (newAccess, newRefresh string, err error) {
	resp, err := http.PostForm(serverURL+"/oauth/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	if err != nil {
		return "", "", fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parse refresh response: %w", err)
	}

	if errStr, ok := result["error"].(string); ok {
		return "", "", fmt.Errorf("refresh failed: %s", errStr)
	}

	newAccess, _ = result["access_token"].(string)
	newRefresh, _ = result["refresh_token"].(string)
	if newAccess == "" {
		return "", "", fmt.Errorf("no access_token in refresh response")
	}
	return newAccess, newRefresh, nil
}

// runDeviceCodeFlow initiates the device code flow and blocks until the user approves.
func runDeviceCodeFlow(serverURL string) (*tokenFile, error) {
	resp, err := http.PostForm(serverURL+"/oauth/device", url.Values{
		"client_id": {"atlinks-mcp"},
	})
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var dcResp map[string]any
	if err := json.Unmarshal(body, &dcResp); err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}

	deviceCode, _ := dcResp["device_code"].(string)
	userCode, _ := dcResp["user_code"].(string)
	verifyURI, _ := dcResp["verification_uri_complete"].(string)
	interval := 5.0
	if v, ok := dcResp["interval"].(float64); ok {
		interval = v
	}

	fmt.Fprintf(os.Stderr, "\n  To authorize this device, open:\n  %s\n\n  Or go to %s and enter code: %s\n\n  Waiting for authorization...\n",
		verifyURI,
		strings.TrimSuffix(verifyURI, "?code="+userCode),
		userCode,
	)

	// Poll for token
	for {
		time.Sleep(time.Duration(interval) * time.Second)

		resp, err := http.PostForm(serverURL+"/oauth/token", url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
		})
		if err != nil {
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tokenResp map[string]any
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			continue
		}

		if errStr, ok := tokenResp["error"].(string); ok {
			switch errStr {
			case "authorization_pending":
				continue
			case "slow_down":
				interval += 5
				continue
			case "expired_token":
				return nil, fmt.Errorf("device code expired — please try again")
			case "access_denied":
				return nil, fmt.Errorf("authorization denied by user")
			default:
				return nil, fmt.Errorf("token error: %s", errStr)
			}
		}

		accessToken, _ := tokenResp["access_token"].(string)
		refreshToken, _ := tokenResp["refresh_token"].(string)

		if accessToken == "" {
			return nil, fmt.Errorf("no access_token in token response")
		}

		fmt.Fprintf(os.Stderr, "  Authorized successfully!\n\n")

		tf := &tokenFile{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ServerURL:    serverURL,
		}
		if err := saveTokens(tf); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save tokens: %v\n", err)
		}
		return tf, nil
	}
}

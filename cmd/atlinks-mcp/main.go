package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	serverURL := os.Getenv("ATLINKS_URL")
	if serverURL == "" {
		serverURL = "https://atlinks.app"
	}

	// Check for --server flag
	for i, arg := range os.Args[1:] {
		if arg == "--server" && i+1 < len(os.Args)-1 {
			serverURL = os.Args[i+2]
		}
	}

	// Handle CLI commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "login":
			tf, err := runDeviceCodeFlow(serverURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Logged in. Token saved to %s\n", tokenFilePath())
			_ = tf
			return
		case "logout":
			path := tokenFilePath()
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Logout failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Logged out. Token file removed.\n")
			return
		case "status":
			tf, err := loadTokens()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Not logged in (no token file found).\n")
				os.Exit(1)
			}
			expired := isTokenExpired(tf.AccessToken)
			fmt.Fprintf(os.Stderr, "Server: %s\n", tf.ServerURL)
			if expired {
				fmt.Fprintf(os.Stderr, "Access token: expired (will auto-refresh)\n")
			} else {
				fmt.Fprintf(os.Stderr, "Access token: valid\n")
			}
			fmt.Fprintf(os.Stderr, "Refresh token: %s\n", func() string {
				if tf.RefreshToken != "" {
					return "present"
				}
				return "missing"
			}())
			return
		}
	}

	// Load or acquire tokens
	tokens, err := loadTokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "No saved tokens found. Starting device code flow...\n")
		tokens, err = runDeviceCodeFlow(serverURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Update server URL from tokens if saved
	if tokens.ServerURL != "" {
		serverURL = tokens.ServerURL
	}

	client := newAtlClient(serverURL, tokens)

	s := server.NewMCPServer(
		"atlinks",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false),
	)

	registerAllTools(s, client)
	registerResources(s, client)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

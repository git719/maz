// token.go

package ezmsal

import (
	"fmt"
	"context"
	"path/filepath"
	"strings"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
)

func GetTokenInteractively(scopes []string, confDir, tokenFile, authorityUrl, username string) (token string, err error) {
	// Interactive login with 'public' app
	// See https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/public/public.go

	// Set up token cache storage file and accessor
	cacheFilePath := filepath.Join(confDir, tokenFile)
	cacheAccessor := &TokenCache{cacheFilePath}

	// Note we're using constant constAzPowerShellClientId for interactive login
	app, err := public.New(constAzPowerShellClientId, public.WithAuthority(authorityUrl), public.WithCache(cacheAccessor))
	if err != nil { panic(err.Error()) }

	// Select the account to use based on username variable 
	var targetAccount public.Account  // Type is defined in 'public' module 
	for _, i := range app.Accounts() {
		if strings.ToLower(i.PreferredUsername) == username {
			targetAccount = i
			break
		}
	}

	// Try getting cached token 1st
	result, err := app.AcquireTokenSilent(context.Background(), scopes, public.WithSilentAccount(targetAccount))
	if err != nil {
		// Else, get a new token
		result, err = app.AcquireTokenInteractive(context.Background(), scopes)
		// AcquireTokenInteractive acquires a security token from the authority using the default web browser to select the account.
		if err != nil { panic(err.Error()) }
	}
	return result.AccessToken, nil	// Return only the AccessToken, which is of type string
}

func GetTokenByCredentials(scopes []string, confDir, tokenFile, authorityUrl, clientId, clientSecret string) (token string, err error) {
	// ClientId+Secret automated login with 'confidential' app
	// See See https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/confidential/confidential.go

	// Set up token cache storage file and accessor
	cacheFilePath := filepath.Join(confDir, tokenFile)
	cacheAccessor := &TokenCache{cacheFilePath}
	
	// Initializing the client credential
	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil { fmt.Println("Could not create a cred object from client_secret.") }

	// Automated login obviously uses the registered app client_id (App ID)
	app, err := confidential.New(clientId,	cred, confidential.WithAuthority(authorityUrl), confidential.WithAccessor(cacheAccessor))
	if err != nil { panic(err.Error()) }

	// Try getting cached token 1st
	// targetAccount not required, as it appears to locate existing cached tokens without it
	result, err := app.AcquireTokenSilent(context.Background(), scopes)
	if err != nil {
		// Else, get a new token
		result, err = app.AcquireTokenByCredential(context.Background(), scopes)
		// AcquireTokenByCredential acquires a security token from the authority, using the client credentials grant.
		if err != nil { panic(err.Error()) }
	}
	return result.AccessToken, nil  // Return only the AccessToken, which is of type string
}

// token.go

package maz

import (
	"context"
	"fmt"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/confidential"
	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	"github.com/git719/utl"
	"github.com/golang-jwt/jwt/v4"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetTokenInteractively(scopes []string, confDir, tokenFile, authorityUrl, username string) (token string, err error) {
	// Interactive login with 'public' app
	// See https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/public/public.go

	// Set up token cache storage file and accessor
	cacheFilePath := filepath.Join(confDir, tokenFile)
	cacheAccessor := &TokenCache{cacheFilePath}

	// Note we're using constant ConstAzPowerShellClientId for interactive login
	app, err := public.New(ConstAzPowerShellClientId, public.WithAuthority(authorityUrl), public.WithCache(cacheAccessor))
	if err != nil {
		PrintApiErrMsg(err.Error())
		utl.Die("")
	}

	// Select the account to use based on username variable
	var targetAccount public.Account // Type is defined in 'public' module
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
		// app.AcquireTokenInteractive uses the default web browser to select the account and acquire a
		// security token from the authority.

		// Note that this obviously does not work from within a VM environment.
		// TODO: Allow use of app.AcquireByDeviceCodeOption or app.AcquireByAuthCodeOption, which
		// would allow interactive login from a virtualize environment.

		if err != nil {
			PrintApiErrMsg(err.Error())
			utl.Die("")
		}

	}
	return result.AccessToken, nil // Return only the AccessToken, which is of type string
}

func GetTokenByCredentials(scopes []string, confDir, tokenFile, authorityUrl, clientId, clientSecret string) (token string, err error) {
	// ClientId+Secret automated login with 'confidential' app
	// See See https://github.com/AzureAD/microsoft-authentication-library-for-go/blob/dev/apps/confidential/confidential.go

	// Set up token cache storage file and accessor
	cacheFilePath := filepath.Join(confDir, tokenFile)
	cacheAccessor := &TokenCache{cacheFilePath}

	// Initializing the client credential
	cred, err := confidential.NewCredFromSecret(clientSecret)
	if err != nil {
		fmt.Println("Could not create a cred object from client_secret.")
	}

	// Automated login obviously uses the registered app client_id (App ID)
	app, err := confidential.New(clientId, cred, confidential.WithAuthority(authorityUrl), confidential.WithAccessor(cacheAccessor))
	if err != nil {
		PrintApiErrMsg(err.Error())
		utl.Die("")
	}

	// Try getting cached token 1st
	// targetAccount not required, as it appears to locate existing cached tokens without it
	result, err := app.AcquireTokenSilent(context.Background(), scopes)
	if err != nil {
		// Else, get a new token
		result, err = app.AcquireTokenByCredential(context.Background(), scopes)
		// AcquireTokenByCredential acquires a security token from the authority, using the client credentials grant.
		if err != nil {
			PrintApiErrMsg(err.Error())
			utl.Die("")
		}
	}
	return result.AccessToken, nil // Return only the AccessToken, which is of type string
}

func DecodeJwtToken(tokenString string) {
	// Decode and dump token string, trusting, without verifying/validating

	// Validate as per https://tools.ietf.org/html/rfc7519
	if tokenString == "" || (!strings.HasPrefix(tokenString, "eyJ") && !strings.Contains(tokenString, ".")) {
		utl.Die("Invalid token: Does not start with 'eyJ', contain any '.', or it's empty.\n")
	}
	// A JSON Web Token consists of three parts which are separated using .(dot):
	// Header: It indicates the token’s type it is and which signing algorithm has been used.
	// Payload: It consists of the claims. And claims comprise of application’s data( email id,
	// username, role), the expiration period of a token (Exp), and so on.
	// Signature: It is generated using the secret (provided by the user), encoded header, and payload.
	//
	// Token struct fields:
	//   Raw       string                 // The raw token.  Populated when you Parse a token
	//   Method    SigningMethod          // The signing method used or to be used
	//   Header    map[string]interface{} // The first segment of the token
	//   Claims    Claims                 // The second segment of the token
	//   Signature string                 // The third segment of the token.  Populated when you Parse a token
	//   Valid     bool                   // Is the token valid?  Populated when you Parse/Verify a token

	// Parse the token without verifying the signature
	claims := jwt.MapClaims{} // claims are actually a map[string]interface{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("<YOUR VERIFICATION KEY>"), nil
	})
	// // Below no yet needed, since this is only printing claims in an unverified way
	// if err != nil {
	// 	fmt.Println(utl.Red("Token is invalid: " + err.Error()))
	// }
	if token == nil {
		fmt.Println(utl.Red("Error parsing token: " + err.Error()))
	}

	fmt.Println(utl.Blu("header") + ":")

	sortedKeys := utl.SortObjStringKeys(token.Header)
	for _, k := range sortedKeys {
		v := token.Header[k]
		fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(v))
	}

	fmt.Println(utl.Blu("claims") + ":")
	sortedKeys = utl.SortObjStringKeys(token.Claims.(jwt.MapClaims))
	for _, k := range sortedKeys {
		v := token.Claims.(jwt.MapClaims)[k]
		vType := utl.GetType(v)
		switch vType {
		case "string":
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(v))
		case "float64":
			vFlt64 := v.(float64)
			t := time.Unix(int64(vFlt64), 0)
			vStr := utl.Gre(t.Format("2006-01-02 15:04:05"))
			vStr += fmt.Sprintf("  # %d", int64(vFlt64))
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), vStr)
		case "[]interface {}":
			vList := v.([]interface{})
			vStr := ""
			for _, i := range vList {
				vStr += utl.Str(i) + " "
			}
			fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(vStr))
		}
	}

	fmt.Println(utl.Blu("signature") + ":")
	if token.Signature != "" {
		k := "signature"
		fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), utl.Gre(token.Signature))
	}

	fmt.Println(utl.Blu("status") + ":")
	k := "valid"
	vStr := ""
	if token.Valid {
		vStr = utl.Gre("true")
	} else {
		vStr = utl.Gre("false") + "  # Since this parsing isn't verifying it"
	}
	fmt.Printf("  %s:%s %s\n", utl.Blu(k), utl.PadSpaces(20, len(k)), vStr)

	os.Exit(0)
}

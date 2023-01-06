// creds.go

package ezmsal

import (
	"os"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/git719/utl"
)

func StrVal(x interface{}) string {
	return utl.StrVal(x)		// Shorthand
}

func DumpVariables(z GlobVarsType) {
	// Dump essential global variables
	fmt.Printf("%-16s %s\n", "tenant_id:", z.tenantId)
    if z.interactive {
		fmt.Printf("%-16s %s\n", "username:", z.username)	
		fmt.Printf("%-16s %s\n", "interactive:", "true")	
	} else {
		fmt.Printf("%-16s %s\n", "client_id:", z.clientId)
		fmt.Printf("%-16s %s\n", "client_secret:", z.clientSecret)	
	}
	fmt.Printf("%-16s %s\n%-16s %s\n%-16s %s\n", "authority_url:", z.authorityUrl, "mg_url:", constMgUrl, "az_url:", constAzUrl)
	fmt.Printf("mg_headers:\n")
	for k, v := range z.mgHeaders {
		fmt.Printf("  %-14s %s\n", StrVal(k) + ":", StrVal(v))
	}
	fmt.Printf("az_headers:\n")
	for k, v := range z.azHeaders {
		fmt.Printf("  %-14s %s\n", StrVal(k) + ":", StrVal(v))
	}
	os.Exit(0)
}

func DumpCredentials(z GlobVarsType) {
	// Dump credentials file
	filePath := filepath.Join(z.confDir, z.credsFile)  // credentials.yaml
	credsRaw, err := utl.LoadFileYaml(filePath)
    if err != nil { utl.Die("[%s] %s\n", filePath, err) }
	creds := credsRaw.(map[string]interface{})
	fmt.Printf("%-14s %s\n", "tenant_id:", StrVal(creds["tenant_id"]))
	if strings.ToLower(StrVal(creds["interactive"])) == "true" {
		fmt.Printf("%-14s %s\n", "username:", StrVal(creds["username"]))
		fmt.Printf("%-14s %s\n", "interactive:", "true")
	} else {
		fmt.Printf("%-14s %s\n", "client_id:", StrVal(creds["client_id"]))
		fmt.Printf("%-14s %s\n", "client_secret:", StrVal(creds["client_secret"]))
	}
	os.Exit(0)
}

func SetupInterativeLogin(z GlobVarsType) {
	// Set up credentials file for interactive login
	filePath := filepath.Join(z.confDir, z.credsFile)  // credentials.yaml
	if !utl.ValidUuid(z.tenantId) { utl.Die("Error. TENANT_ID is an invalid UUIs.\n") }
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.tenantId, "username:", z.username, "interactive:", "true")
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupAutomatedLogin(z GlobVarsType) {
	// Set up credentials file for client_id + secret login
	filePath := filepath.Join(z.confDir, z.credsFile)  // credentials.yaml
	if !utl.ValidUuid(z.tenantId) { utl.Die("Error. TENANT_ID is an invalid UUIs.\n") }
	if !utl.ValidUuid(z.clientId) { utl.Die("Error. CLIENT_ID is an invalid UUIs.\n") }
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.tenantId, "client_id:", z.clientId, "client_secret:", z.clientSecret)
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupCredentials(z GlobVarsType) (GlobVarsType) {
	// Read credentials file and set up authentication parameters as global variables
	filePath := filepath.Join(z.confDir, z.credsFile)  // credentials.yaml
	if utl.FileNotExist(filePath) && utl.FileSize(filePath) < 1 {
		utl.Die("Missing credentials file: '%s'\n", filePath +
			"Please rerun program using '-cr' or '-cri' option to specify credentials.\n")
	}
	credsRaw, err := utl.LoadFileYaml(filePath)
    if err != nil { utl.Die("[%s] %s\n", filePath, err) }
	creds := credsRaw.(map[string]interface{})

	// Note that we are updating variables to be returned and used globally
	z.tenantId = StrVal(creds["tenant_id"])
	if !utl.ValidUuid(z.tenantId) { utl.Die("[%s] tenant_id '%s' is not a valid UUID\n", filePath, z.tenantId) }
	
	z.interactive, err = strconv.ParseBool(StrVal(creds["interactive"]))
	
	if z.interactive {
		z.username = strings.ToLower(StrVal(creds["username"]))
	} else {
		z.clientId = StrVal(creds["client_id"])
		if !utl.ValidUuid(z.clientId) { utl.Die("[%s] client_id '%s' is not a valid UUID\n", filePath, z.clientId) }	
		z.clientSecret = StrVal(creds["client_secret"])
		if z.clientSecret == "" { utl.Die("[%s] client_secret is blank\n", filePath) }
	}
	return z
}

func SetupApiTokens(z GlobVarsType) (GlobVarsType) {
	// Initialize necessary global variables, acquire all API tokens, and set them up for use
	z = SetupCredentials(z)  // Sets up tenant ID, client ID, authentication method, etc
	z.authorityUrl = constAuthUrl + z.tenantId

	// Currently supporting calls for 2 different APIs (Azure Resource Management (ARM) and MS Graph), so each needs its own
	// separate token. The Microsoft identity platform does not allow using same token for multiple resources at once.
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-net-user-gets-consent-for-multiple-resources

	// Get a token for ARM access
	azScope := []string{constAzUrl + "/.default"}  // The scope is a slice of URL strings
	// Appending '/.default' allows using all static and consented permissions of the identity in use
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-v1-app-scopes
	if z.interactive {
		// Get token interactively
		z.azToken, _ = GetTokenInteractively(azScope, z.confDir, z.tokenFile, z.authorityUrl, z.username)
	} else {
		// Get token with clientId + Secret
		z.azToken, _ = GetTokenByCredentials(azScope, z.confDir, z.tokenFile, z.authorityUrl, z.clientId, z.clientSecret)
	}
	z.azHeaders = map[string]string{ "Authorization": "Bearer " + z.azToken, "Content-Type":  "application/json", }
	
	// Get a token for MS Graph access	
	mgScope := []string{constMgUrl + "/.default"}
	if z.interactive {
		// Get token interactively
		z.mgToken, _ = GetTokenInteractively(mgScope, z.confDir, z.tokenFile, z.authorityUrl, z.username)
	} else {
		// Get token with clientId + Secret
		z.mgToken, _ = GetTokenByCredentials(mgScope, z.confDir, z.tokenFile, z.authorityUrl, z.clientId, z.clientSecret)
	}
	z.mgHeaders = map[string]string{"Authorization": "Bearer " + z.mgToken, "Content-Type":  "application/json", 	}

	// Support for other APIs can be added here in the future ...

	return z
}

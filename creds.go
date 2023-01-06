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
	fmt.Printf("%-16s %s\n", "tenant_id:", z.TenantId)
    if z.Interactive {
		fmt.Printf("%-16s %s\n", "username:", z.Username)	
		fmt.Printf("%-16s %s\n", "interactive:", "true")	
	} else {
		fmt.Printf("%-16s %s\n", "client_id:", z.ClientId)
		fmt.Printf("%-16s %s\n", "client_secret:", z.ClientSecret)	
	}
	fmt.Printf("%-16s %s\n%-16s %s\n%-16s %s\n", "authority_url:", z.AuthorityUrl, "mg_url:", ConstMgUrl, "az_url:", ConstAzUrl)
	fmt.Printf("mg_headers:\n")
	for k, v := range z.MgHeaders {
		fmt.Printf("  %-14s %s\n", StrVal(k) + ":", StrVal(v))
	}
	fmt.Printf("az_headers:\n")
	for k, v := range z.AzHeaders {
		fmt.Printf("  %-14s %s\n", StrVal(k) + ":", StrVal(v))
	}
	os.Exit(0)
}

func DumpCredentials(z GlobVarsType) {
	// Dump credentials file
	filePath := filepath.Join(z.ConfDir, z.CredsFile)  // credentials.yaml
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
	filePath := filepath.Join(z.ConfDir, z.CredsFile)  // credentials.yaml
	if !utl.ValidUuid(z.TenantId) { utl.Die("Error. TENANT_ID is an invalid UUIs.\n") }
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "username:", z.Username, "interactive:", "true")
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupAutomatedLogin(z GlobVarsType) {
	// Set up credentials file for client_id + secret login
	filePath := filepath.Join(z.ConfDir, z.CredsFile)  // credentials.yaml
	if !utl.ValidUuid(z.TenantId) { utl.Die("Error. TENANT_ID is an invalid UUIs.\n") }
	if !utl.ValidUuid(z.ClientId) { utl.Die("Error. CLIENT_ID is an invalid UUIs.\n") }
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "client_id:", z.ClientId, "client_secret:", z.ClientSecret)
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupCredentials(z GlobVarsType) (GlobVarsType) {
	// Read credentials file and set up authentication parameters as global variables
	filePath := filepath.Join(z.ConfDir, z.CredsFile)  // credentials.yaml
	if utl.FileNotExist(filePath) && utl.FileSize(filePath) < 1 {
		utl.Die("Missing credentials file: '%s'\n", filePath +
			"Please rerun program using '-cr' or '-cri' option to specify credentials.\n")
	}
	credsRaw, err := utl.LoadFileYaml(filePath)
    if err != nil { utl.Die("[%s] %s\n", filePath, err) }
	creds := credsRaw.(map[string]interface{})

	// Note that we are updating variables to be returned and used globally
	z.TenantId = StrVal(creds["tenant_id"])
	if !utl.ValidUuid(z.TenantId) { utl.Die("[%s] tenant_id '%s' is not a valid UUID\n", filePath, z.TenantId) }
	
	z.Interactive, err = strconv.ParseBool(StrVal(creds["interactive"]))
	
	if z.Interactive {
		z.Username = strings.ToLower(StrVal(creds["username"]))
	} else {
		z.ClientId = StrVal(creds["client_id"])
		if !utl.ValidUuid(z.ClientId) { utl.Die("[%s] client_id '%s' is not a valid UUID\n", filePath, z.ClientId) }	
		z.ClientSecret = StrVal(creds["client_secret"])
		if z.ClientSecret == "" { utl.Die("[%s] client_secret is blank\n", filePath) }
	}
	return z
}

func SetupApiTokens(z GlobVarsType) (GlobVarsType) {
	// Initialize necessary global variables, acquire all API tokens, and set them up for use
	z = SetupCredentials(z)  // Sets up tenant ID, client ID, authentication method, etc
	z.AuthorityUrl = ConstAuthUrl + z.TenantId

	// Currently supporting calls for 2 different APIs (Azure Resource Management (ARM) and MS Graph), so each needs its own
	// separate token. The Microsoft identity platform does not allow using same token for multiple resources at once.
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-net-user-gets-consent-for-multiple-resources

	// Get a token for ARM access
	azScope := []string{ConstAzUrl + "/.default"}  // The scope is a slice of URL strings
	// Appending '/.default' allows using all static and consented permissions of the identity in use
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-v1-app-scopes
	if z.Interactive {
		// Get token interactively
		z.AzToken, _ = GetTokenInteractively(azScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.Username)
	} else {
		// Get token with clientId + Secret
		z.AzToken, _ = GetTokenByCredentials(azScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.ClientId, z.ClientSecret)
	}
	z.AzHeaders = map[string]string{ "Authorization": "Bearer " + z.AzToken, "Content-Type":  "application/json", }
	
	// Get a token for MS Graph access	
	mgScope := []string{ConstMgUrl + "/.default"}
	if z.Interactive {
		// Get token interactively
		z.MgToken, _ = GetTokenInteractively(mgScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.Username)
	} else {
		// Get token with clientId + Secret
		z.MgToken, _ = GetTokenByCredentials(mgScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.ClientId, z.ClientSecret)
	}
	z.MgHeaders = map[string]string{"Authorization": "Bearer " + z.MgToken, "Content-Type":  "application/json", 	}

	// Support for other APIs can be added here in the future ...

	return z
}

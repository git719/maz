// maz.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ConstAuthUrl              = "https://login.microsoftonline.com/"
	ConstMgUrl                = "https://graph.microsoft.com"
	ConstAzUrl                = "https://management.azure.com"
	ConstAzPowerShellClientId = "1950a258-227b-4e31-a9cf-717495945fc2"
	// Interactive login uses above 'Azure PowerShell' clientId
	// See https://stackoverflow.com/questions/30454771/how-does-azure-powershell-work-with-username-password-based-auth
        rUp = "\x1B[2K\r" // Used to print in previous line
	// See https://stackoverflow.com/questions/1508490/erase-the-current-printed-console-line
)

type Bundle struct {
	ConfDir      string // Directory where utility will store all its file
	CredsFile    string
	TokenFile    string
	TenantId     string
	ClientId     string
	ClientSecret string
	Interactive  bool
	Username     string
	AuthorityUrl string

        // To support MS Graph API
	MgToken      string 
	MgHeaders    map[string]string

        // To support Azure Resource Management API
	AzToken      string
	AzHeaders    map[string]string

       // In the future, other token/headers pairs for other APIs can be added below
}

func DumpVariables(z Bundle) {
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
		fmt.Printf("  %-14s %s\n", utl.Str(k)+":", utl.Str(v))
	}
	fmt.Printf("az_headers:\n")
	for k, v := range z.AzHeaders {
		fmt.Printf("  %-14s %s\n", utl.Str(k)+":", utl.Str(v))
	}
	os.Exit(0)
}

func DumpCredentials(z Bundle) {
	// Dump credentials file
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	credsRaw, err := utl.LoadFileYaml(filePath)
	if err != nil {
		utl.Die("[%s] %s\n", filePath, err)
	}
	creds := credsRaw.(map[string]interface{})
	fmt.Printf("%-14s %s\n", "tenant_id:", utl.Str(creds["tenant_id"]))
	if strings.ToLower(utl.Str(creds["interactive"])) == "true" {
		fmt.Printf("%-14s %s\n", "username:", utl.Str(creds["username"]))
		fmt.Printf("%-14s %s\n", "interactive:", "true")
	} else {
		fmt.Printf("%-14s %s\n", "client_id:", utl.Str(creds["client_id"]))
		fmt.Printf("%-14s %s\n", "client_secret:", utl.Str(creds["client_secret"]))
	}
	os.Exit(0)
}

func SetupInterativeLogin(z Bundle) {
	// Set up credentials file for interactive login
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUIs.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "username:", z.Username, "interactive:", "true")
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupAutomatedLogin(z Bundle) {
	// Set up credentials file for client_id + secret login
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUIs.\n")
	}
	if !utl.ValidUuid(z.ClientId) {
		utl.Die("Error. CLIENT_ID is an invalid UUIs.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "client_id:", z.ClientId, "client_secret:", z.ClientSecret)
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("[%s] Updated credentials\n", filePath)
}

func SetupCredentials(z *Bundle) Bundle {
	// Read credentials file and set up authentication parameters as global variables
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	if utl.FileNotExist(filePath) && utl.FileSize(filePath) < 1 {
		utl.Die("Missing credentials file: '%s'\n", filePath+
			"Please rerun program using '-cr' or '-cri' option to specify credentials.\n")
	}
	credsRaw, err := utl.LoadFileYaml(filePath)
	if err != nil {
		utl.Die("[%s] %s\n", filePath, err)
	}
	creds := credsRaw.(map[string]interface{})

	// Note that we are updating variables to be returned and used globally
	z.TenantId = utl.Str(creds["tenant_id"])
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("[%s] tenant_id '%s' is not a valid UUID\n", filePath, z.TenantId)
	}

	z.Interactive, err = strconv.ParseBool(utl.Str(creds["interactive"]))

	if z.Interactive {
		z.Username = strings.ToLower(utl.Str(creds["username"]))
	} else {
		z.ClientId = utl.Str(creds["client_id"])
		if !utl.ValidUuid(z.ClientId) {
			utl.Die("[%s] client_id '%s' is not a valid UUID\n", filePath, z.ClientId)
		}
		z.ClientSecret = utl.Str(creds["client_secret"])
		if z.ClientSecret == "" {
			utl.Die("[%s] client_secret is blank\n", filePath)
		}
	}
	return *z
}

func SetupApiTokens(z *Bundle) Bundle {
	// Initialize necessary global variables, acquire all API tokens, and set them up for use
	*z = SetupCredentials(z) // Sets up tenant ID, client ID, authentication method, etc
	z.AuthorityUrl = ConstAuthUrl + z.TenantId

	// Currently supporting calls for 2 different APIs (Azure Resource Management (ARM) and MS Graph), so each needs its own
	// separate token. The Microsoft identity platform does not allow using same token for multiple resources at once.
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-net-user-gets-consent-for-multiple-resources

	// Get a token for ARM access
	azScope := []string{ConstAzUrl + "/.default"} // The scope is a slice of URL strings
	// Appending '/.default' allows using all static and consented permissions of the identity in use
	// See https://learn.microsoft.com/en-us/azure/active-directory/develop/msal-v1-app-scopes
	if z.Interactive {
		// Get token interactively
		z.AzToken, _ = GetTokenInteractively(azScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.Username)
	} else {
		// Get token with clientId + Secret
		z.AzToken, _ = GetTokenByCredentials(azScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.ClientId, z.ClientSecret)
	}
	z.AzHeaders = map[string]string{"Authorization": "Bearer " + z.AzToken, "Content-Type": "application/json"}

	// Get a token for MS Graph access
	mgScope := []string{ConstMgUrl + "/.default"}
	if z.Interactive {
		// Get token interactively
		z.MgToken, _ = GetTokenInteractively(mgScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.Username)
	} else {
		// Get token with clientId + Secret
		z.MgToken, _ = GetTokenByCredentials(mgScope, z.ConfDir, z.TokenFile, z.AuthorityUrl, z.ClientId, z.ClientSecret)
	}
	z.MgHeaders = map[string]string{"Authorization": "Bearer " + z.MgToken, "Content-Type": "application/json"}

	// Support for other APIs can be added here in the future ...

	return *z
}

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
	MgToken      string // This and below to support MS Graph API
	MgHeaders    map[string]string
	AzToken      string // This and below to support Azure Resource Management API
	AzHeaders    map[string]string
	// To support other future APIs, those token/headers pairs can be added here
}

func DumpVariables(z Bundle) {
	// Dump essential global variables
	co := utl.Red(":")
	cConfDir := utl.Cya("config_dir") + co
	cComment := utl.Blu("# Utility's config and cache directory")
	fmt.Printf("%s %s  %s\n", cConfDir, z.ConfDir, cComment)

	cTenant := utl.Cya("tenant_id") + co
	fmt.Printf("%s %s\n", cTenant, z.TenantId)
	if z.Interactive {
		cUsername := utl.Cya("username") + co
		fmt.Printf("%s %s\n", cUsername, z.Username)
		cInterative := utl.Cya("interactive") + co
		fmt.Printf("%s %s\n", cInterative, "true")
	} else {
		cClientId := utl.Cya("client_id") + co
		fmt.Printf("%s %s\n", cClientId, z.ClientId)
		cClientSecret := utl.Cya("client_secret") + co
		fmt.Printf("%s %s\n", cClientSecret, z.ClientSecret)
	}

	cAuthorityUrl := utl.Cya("authority_url") + co
	fmt.Printf("%s %s\n", cAuthorityUrl, z.AuthorityUrl)
	cMgUrl := utl.Cya("mg_url") + co
	fmt.Printf("%s %s\n", cMgUrl, ConstMgUrl)
	cAzUrl := utl.Cya("az_url") + co
	fmt.Printf("%s %s\n", cAzUrl, ConstAzUrl)

	fmt.Println(utl.Cya("mg_headers") + co)
	PrintStringMapColor(z.MgHeaders)
	fmt.Println(utl.Cya("az_headers") + co)
	PrintStringMapColor(z.AzHeaders)
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
	co := utl.Red(":")
	cTenant := utl.Cya("tenant_id") + co
	fmt.Printf("%s %s\n", cTenant, utl.Str(creds["tenant_id"]))
	if strings.ToLower(utl.Str(creds["interactive"])) == "true" {
		cUsername := utl.Cya("username") + co
		fmt.Printf("%s %s\n", cUsername, utl.Str(creds["username"]))
		cInterative := utl.Cya("interactive") + co
		fmt.Printf("%s %s\n", cInterative, "true")
	} else {
		cClientId := utl.Cya("client_id") + co
		fmt.Printf("%s %s\n", cClientId, utl.Str(creds["client_id"]))
		cClientSecret := utl.Cya("client_secret") + co
		fmt.Printf("%s %s\n", cClientSecret, utl.Str(creds["client_secret"]))
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
		utl.Die("Missing credentials file: " + filePath + "\n" +
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

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
	ConstAuthUrl = "https://login.microsoftonline.com/"
	ConstMgUrl   = "https://graph.microsoft.com"
	ConstAzUrl   = "https://management.azure.com"

	ConstAzPowerShellClientId = "1950a258-227b-4e31-a9cf-717495945fc2" // 'Microsoft Azure PowerShell' ClientId
	//ConstAzPowerShellClientId = "04b07795-8ddb-461a-bbee-02f9e1bf7b46" // 'Microsoft Azure CLI' ClientId
	// Interactive login can use either of above ClientIds. See below references:
	//   - https://learn.microsoft.com/en-us/troubleshoot/azure/active-directory/verify-first-party-apps-sign-in
	//   - https://stackoverflow.com/questions/30454771/how-does-azure-powershell-work-with-username-password-based-auth

	rUp = "\x1B[2K\r" // Used to print in previous line
	// See https://stackoverflow.com/questions/1508490/erase-the-current-printed-console-line
	ConstCacheFileExtension   = "gz"
	ConstMgCacheFileAgePeriod = 1800  // Half hour
	ConstAzCacheFileAgePeriod = 86400 // One day
)

var (
	mazTypes     = []string{"d", "a", "s", "u", "g", "sp", "ap", "ad"}
	mazTypesLong = map[string]string{
		"d":  "RBAC Role Definition",
		"a":  "RBAC Role Assignment",
		"s":  "Azure Subscription",
		"u":  "Azure AD User",
		"g":  "Azure AD Group",
		"sp": "Service Principal",
		"ap": "Registered Application",
		"ad": "Azure AD Role",
	}
	eVars = map[string]string{
		"MAZ_TENANT_ID":     "",
		"MAZ_USERNAME":      "",
		"MAZ_INTERACTIVE":   "",
		"MAZ_CLIENT_ID":     "",
		"MAZ_CLIENT_SECRET": "",
	}
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

func DumpLoginValues(z Bundle) {
	// Dump configured login values
	fmt.Printf("%s: %s  # Config and cache directory\n", utl.Blu("config_dir"), utl.Gre(z.ConfDir))

	fmt.Println(utl.Blu("os_environment_variables:"))
	fmt.Println("  # 1. Environment Variable login values override values in credentials_config_file")
	fmt.Println("  # 2. MAZ_USERNAME+MAZ_INTERACTIVE login have priority over MAZ_CLIENT_ID+MAZ_CLIENT_SECRET login")
	fmt.Println("  # 3. To use MAZ_CLIENT_ID+MAZ_CLIENT_SECRET login ensure MAZ_USERNAME & MAZ_INTERACTIVE are unset")
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_TENANT_ID"), utl.Gre(os.Getenv("MAZ_TENANT_ID")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_USERNAME"), utl.Gre(os.Getenv("MAZ_USERNAME")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_INTERACTIVE"), utl.Mag(os.Getenv("MAZ_INTERACTIVE")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_CLIENT_ID"), utl.Gre(os.Getenv("MAZ_CLIENT_ID")))
	fmt.Printf("  %s: %s\n", utl.Blu("MAZ_CLIENT_SECRET"), utl.Gre(os.Getenv("MAZ_CLIENT_SECRET")))

	fmt.Println(utl.Blu("credentials_config_file:"))
	filePath := filepath.Join(z.ConfDir, z.CredsFile)
	fmt.Printf("  %s: %s\n", utl.Blu("file_path"), utl.Gre(filePath))
	credsRaw, err := utl.LoadFileYaml(filePath)
	if err != nil {
		utl.Die("[%s] %s\n", filePath, err)
	}
	creds := credsRaw.(map[string]interface{})
	fmt.Printf("  %s: %s\n", utl.Blu("tenant_id"), utl.Gre(utl.Str(creds["tenant_id"])))
	if strings.ToLower(utl.Str(creds["interactive"])) == "true" {
		fmt.Printf("  %s: %s\n", utl.Blu("username"), utl.Gre(utl.Str(creds["username"])))
		fmt.Printf("  %s: %s\n", utl.Blu("interactive"), utl.Mag("true"))
	} else {
		fmt.Printf("  %s: %s\n", utl.Blu("client_id"), utl.Gre(utl.Str(creds["client_id"])))
		fmt.Printf("  %s: %s\n", utl.Blu("client_secret"), utl.Gre(utl.Str(creds["client_secret"])))
	}
	os.Exit(0)
}

func DumpRuntimeValues(z Bundle) {
	// Dump runtime global variables
	fmt.Printf("%s: %s  # Config and cache directory\n", utl.Blu("config_dir"), utl.Gre(z.ConfDir))

	fmt.Println(utl.Blu("runtime_credentials:"))
	fmt.Printf("  %s: %s\n", utl.Blu("tenant_id"), utl.Gre(z.TenantId))
	if z.Interactive {
		fmt.Printf("  %s: %s\n", utl.Blu("username"), utl.Gre(z.Username))
		fmt.Printf("  %s: %s\n", utl.Blu("interactive"), utl.Mag("true"))
	} else {
		fmt.Printf("  %s: %s\n", utl.Blu("client_id"), utl.Gre(z.ClientId))
		fmt.Printf("  %s: %s\n", utl.Blu("client_secret"), utl.Gre(z.ClientSecret))
	}

	fmt.Println(utl.Blu("api_variables:"))
	fmt.Printf("  %s: %s\n", utl.Blu("authority_url"), utl.Gre(z.AuthorityUrl))
	fmt.Printf("  %s: %s\n", utl.Blu("mg_url"), utl.Gre(ConstMgUrl))
	fmt.Printf("  %s: %s\n", utl.Blu("az_url"), utl.Gre(ConstAzUrl))
	os.Exit(0)
}

func SetupInterativeLogin(z Bundle) {
	// Set up credentials file for interactive login
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUID.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "username:", z.Username, "interactive:", "true")
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("Updated %s file\n", utl.Gre(filePath))
	os.Exit(0)
}

func SetupAutomatedLogin(z Bundle) {
	// Set up credentials file for client_id + secret login
	filePath := filepath.Join(z.ConfDir, z.CredsFile) // credentials.yaml
	if !utl.ValidUuid(z.TenantId) {
		utl.Die("Error. TENANT_ID is an invalid UUID.\n")
	}
	if !utl.ValidUuid(z.ClientId) {
		utl.Die("Error. CLIENT_ID is an invalid UUID.\n")
	}
	content := fmt.Sprintf("%-14s %s\n%-14s %s\n%-14s %s\n", "tenant_id:", z.TenantId, "client_id:", z.ClientId, "client_secret:", z.ClientSecret)
	if err := ioutil.WriteFile(filePath, []byte(content), 0600); err != nil { // Write string to file
		panic(err.Error())
	}
	fmt.Printf("Updated %s file\n", utl.Gre(filePath))
	os.Exit(0)
}

func SetupCredentials(z *Bundle) Bundle {
	// Get credentials from OS environment variables (takes precedence) or from credentials file
	usingEnv := false // Assume environment variables are not being used
	for k := range eVars {
		eVars[k] = os.Getenv(k) // Read all pertinent variables
		if eVars[k] != "" {
			usingEnv = true
		}
	}
	if usingEnv {
		// Getting from OS environment variables
		z.TenantId = eVars["MAZ_TENANT_ID"]
		if !utl.ValidUuid(z.TenantId) {
			utl.Die("[MAZ_TENANT_ID] tenant_id '%s' is not a valid UUID\n", z.TenantId)
		}
		z.Interactive, _ = strconv.ParseBool(utl.Str(eVars["MAZ_INTERACTIVE"]))
		if z.Interactive {
			z.Username = strings.ToLower(utl.Str(eVars["MAZ_USERNAME"]))
			if z.ClientId != "" || z.ClientSecret != "" {
				fmt.Println("Warning: ", utl.Yel(""))
			}
		} else {
			z.ClientId = utl.Str(eVars["MAZ_CLIENT_ID"])
			if !utl.ValidUuid(z.ClientId) {
				utl.Die("[MAZ_CLIENT_ID] client_id '%s' is not a valid UUID\n", z.ClientId)
			}
			z.ClientSecret = utl.Str(eVars["MAZ_CLIENT_SECRET"])
			if z.ClientSecret == "" {
				utl.Die("[MAZ_CLIENT_SECRET] client_secret is blank\n")
			}
		}
	} else {
		// Getting from credentials file
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
		z.TenantId = utl.Str(creds["tenant_id"])
		if !utl.ValidUuid(z.TenantId) {
			utl.Die("[%s] tenant_id '%s' is not a valid UUID\n", filePath, z.TenantId)
		}
		z.Interactive, _ = strconv.ParseBool(utl.Str(creds["interactive"]))
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

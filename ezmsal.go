// ezmsal.go

package ezmsal

const (
	constAuthUrl = "https://login.microsoftonline.com/"
	constMgUrl   = "https://graph.microsoft.com"
	constAzUrl   = "https://management.azure.com"
	constAzPowerShellClientId = "1950a258-227b-4e31-a9cf-717495945fc2"
	// Interactive login uses above 'Azure PowerShell' clientId
	// See https://stackoverflow.com/questions/30454771/how-does-azure-powershell-work-with-username-password-based-auth
)

type MapType map[string]string          // For easier reading

type GlobVarsType struct {
	confDir       string                // Directory where utility will store all its file 
	credsFile     string
	tokenFile     string
	tenantId      string
	clientId      string
	clientSecret  string
	interactive   bool
	username      string
	authorityUrl  string
	mgToken       string                // MS Graph API ...
	mgHeaders     map[string]string
	azToken       string                // Azure Resource Management API
	azHeaders     map[string]string
}

// See README.mg

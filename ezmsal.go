// ezmsal.go

package ezmsal

const (
	ConstAuthUrl = "https://login.microsoftonline.com/"
	ConstMgUrl   = "https://graph.microsoft.com"
	ConstAzUrl   = "https://management.azure.com"
	ConstAzPowerShellClientId = "1950a258-227b-4e31-a9cf-717495945fc2"
	// Interactive login uses above 'Azure PowerShell' clientId
	// See https://stackoverflow.com/questions/30454771/how-does-azure-powershell-work-with-username-password-based-auth
)

type MapType map[string]string          // For easier reading

type GlobVarsType struct {
	ConfDir       string                // Directory where utility will store all its file 
	CredsFile     string
	TokenFile     string
	TenantId      string
	ClientId      string
	ClientSecret  string
	Interactive   bool
	Username      string
	AuthorityUrl  string
	MgToken       string                // MS Graph API ...
	MgHeaders     map[string]string
	AzToken       string                // Azure Resource Management API
	AzHeaders     map[string]string
}

// See README.mg

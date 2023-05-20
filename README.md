# maz
Microsoft Azure library module for simple MSAL authentication, and calling MS Graph and Azure resource APIs.
Other APIs could be added in the future.

WARNING: Currently under constant changes.

## Getting Started
1. Any program or utility wanting to use this libray module can simply import it, then instantiate a variable
of type `maz.Bundle` to manage the interaction. For example: 

```go
import (
    "github.com/git719/maz"
)
z := maz.Bundle{
    ConfDir:      "",                   // Set up later, see example below
    CredsFile:    "credentials.yaml",
    TokenFile:    "accessTokens.json",
    TenantId:     "",
    ClientId:     "",
    ClientSecret: "",
    Interactive:  false,
    Username:     "",
    AuthorityUrl: "",                   // Set up later with maz.ConstAuthUrl + z.TenantId (see const block in maz.go)
    MgToken:      "",                   // Set up below 4 later with function maz.SetupApiTokens()
    MgHeaders:    map[string]string{},
    AzToken:      "",
    AzHeaders:    map[string]string{},  
}
// Then update the variables within the Bundle, to set up configuration directory
z.ConfDir = filepath.Join(os.Getenv("HOME"), "." + prgname)
if utl.FileNotExist(z.ConfDir) {
    if err := os.Mkdir(z.ConfDir, 0700); err != nil {
        panic(err.Error())
    }
}
```

2. Then call `maz.SetupInterativeLogin(z)` or `maz.SetupAutomatedLogin(z)` to setup the credentials file accordingly.
3. Then call `z := maz.SetupApiTokens(*z)` to acquire the respective API tokens, web headers, and other variables.
4. Now call whatever MS Graph and Azure Resource API functions you want by passing and using the `z` variables,
with its `z.mgHeaders` and/or `z.azHeaders` attributes, and so on.

## Login Credentials
There are 4 different ways to set up the login credentials to use this library module:
||*||Type||Method||Details||
||1|Interactive|Config file|Set up `credentials.yaml` file with the 3 special attributes|
||2|Interactive|Environment variables|Set up the 3 special attributes via environment variables|
||3|Automated|Config file|Set up `credentials.yaml` file with the 3 special attributes|
||4|Automated|Environment variables|Set up the 3 special attributes via environment variables|

1. 

## Functions
List of all available functions.
TBD

### maz.SetupInterativeLogin
This functions allows you to set up interactive Azure login.
...

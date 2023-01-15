# maz
Microsoft Azure module for simple MSAL authentication, and calling MS Graph and Azure resource APIs. Other APIs could be added in the future.

WARNING: Currently under constant changes.

## Getting Started
1. Any program or utility wanting to use this module can simply import it, then instantiate a variable of type `maz.Bundle`
to manage the interaction. For example: 

```
import (
    "github.com/git719/maz"
)
z := maz.Bundle{
    ConfDir:      "",                   // You set later, see example below
    CredsFile:    "credentials.yaml",
    TokenFile:    "accessTokens.json",
    TenantId:     "",
    ClientId:     "",
    ClientSecret: "",
    Interactive:  false,
    Username:     "",
    AuthorityUrl: "",                   // You set later to maz.ConstAuthUrl + z.TenantId
    MgToken:      "",                   // you set below 4 with function maz.SetupApiTokens()
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
4. Now pass and use the `z` variables, with its `z.mgHeaders` and/or `z.azHeaders` attributes, to call your own REST
API functions to do whatever you want.

## Functions
A breakdown of available functions.

### maz.SetupInterativeLogin
This functions allows you to set up interactive Azure login.
...

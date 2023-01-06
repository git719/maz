# ezmsal
Easy Azure MSAL authentication for command line utilities.

# Getting Started
1. Any utility using this ezmsal module must instantiate a variable of type `ezmsal.GlobVarsType`. For example: 

```
import (
    "github.com/git719/ezmsal"
)

z := make(ezmsal.GlobVarsType{
    confDir:      "",               // Set to your liking, e.g., filepath.Join(os.Getenv("HOME"), "." + prgname)
    credsFile:    "",               // Set to something like "credentials.yaml"
    tokenFile:    "",               // Set to something like "accessTokens.json"
    tenantId:     "",               // Set the following 5 according to credsFile
    clientId:     "",
    clientSecret: "",
    interactive:  nil,
    username:     "",
    authorityUrl: "",               // Set to constAuthUrl + tenantID
    mgToken:      "",               // Below 4 will be set up by SetupApiTokens()
    mgHeaders:    ezmsl.MapType{},
    azToken:      "",
    azHeaders:    ezmsl.MapType{},  
})
```

2. Then call `SetupInterativeLogin(z)` or `SetupAutomatedLogin(z)` to setup the credentials file accordingly.
3. Then call `z := SetupApiTokens(z)` to acquire the respective API tokens and web headers.
4. Now use `z.mgHeaders` and/or `z.azHeaders` to call your own functions to do whatever you want with those APIs.

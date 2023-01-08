# aza
Easy Azure MSAL authentication for command line and automated utilities.

# Getting Started
1. Any utility using this `aza` module must instantiate a variable of type `aza.AzaBundle`. For example: 

```
import (
    "github.com/git719/aza"
)

z := aza.AzaBundle{
    confDir:      "",               // To be set to filepath.Join(os.Getenv("HOME"), "." + prgname)
    credsFile:    "",               // To be set to something like "credentials.yaml"
    tokenFile:    "",               // To be set to something like "accessTokens.json"
    tenantId:     "",               // The following 5 to be set according to credsFile
    clientId:     "",
    clientSecret: "",
    interactive:  false,
    username:     "",
    authorityUrl: "",               // Set to ConstAuthUrl + tenantID
    mgToken:      "",               // Below 4 will be set up by SetupApiTokens()
    mgHeaders:    aza.MapString{},
    azToken:      "",
    azHeaders:    aza.MapString{},  
}
```

2. Then call `SetupInterativeLogin(z)` or `SetupAutomatedLogin(z)` to setup the credentials file accordingly.
3. Then call `z := SetupApiTokens(*z)` to acquire the respective API tokens and web headers.
4. Now use `z.mgHeaders` and/or `z.azHeaders` to call your own REST API functions to do whatever you want.

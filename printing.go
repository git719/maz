// printing.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"os"
	"time"
)

func PrintCountStatus(z Bundle) {
	fmt.Printf("Note: Counting some Azure resources can take a long time.\n")
	fmt.Printf("%-36s %10s %10s\n", "OBJECTS", "LOCAL", "AZURE")
	fmt.Printf("%-36s %10d %10d\n", "Azure AD Users", UsersCountLocal(z), UsersCountAzure(z))
	fmt.Printf("%-36s %10d %10d\n", "Azure AD Groups", GroupsCountLocal(z), GroupsCountAzure(z))
	fmt.Printf("%-36s %10d %10d\n", "Azure App Registrations", AppsCountLocal(z), AppsCountAzure(z))
	nativeSpsLocal, msSpsLocal := SpsCountLocal(z)
	nativeSpsAzure, msSpsAzure := SpsCountAzure(z)
	fmt.Printf("%-36s %10d %10d\n", "Azure SPs (multi-tenant)", msSpsLocal, msSpsAzure)
	fmt.Printf("%-36s %10d %10d\n", "Azure SPs (native to tenant)", nativeSpsLocal, nativeSpsAzure)
	fmt.Printf("%-36s %10d %10d\n", "Azure AD Roles", AdRolesCountLocal(z), AdRolesCountAzure(z))
	fmt.Printf("%-36s %10d %10d\n", "Azure Management Groups", MgGroupCountLocal(z), MgGroupCountAzure(z))
	fmt.Printf("%-36s %10d %10d\n", "Azure Subscriptions", SubsCountLocal(z), SubsCountAzure(z))
	builtinLocal, customLocal := RoleDefinitionCountLocal(z)
	builtinAzure, customAzure := RoleDefinitionCountAzure(z)
	fmt.Printf("%-36s %10d %10d\n", "Resource Role Definitions BuiltIn", builtinLocal, builtinAzure)
	fmt.Printf("%-36s %10d %10d\n", "Resource Role Definitions Custom", customLocal, customAzure)
	fmt.Printf("%-36s %10d %10d\n", "Resource Role Assignments", RoleAssignmentsCountLocal(z), RoleAssignmentsCountAzure(z))
}

func PrintTersely(t string, object interface{}) {
	// Print this single object of type 't' tersely (minimal attributes)
	x := object.(map[string]interface{}) // Assert as JSON object
	switch t {
	case "d":
		xProp := x["properties"].(map[string]interface{})
		fmt.Printf("%s  %-60s  %s\n", utl.Str(x["name"]), utl.Str(xProp["roleName"]), utl.Str(xProp["type"]))
	case "a":
		xProp := x["properties"].(map[string]interface{})
		rdId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
		principalId := utl.Str(xProp["principalId"])
		principalType := utl.Str(xProp["principalType"])
		scope := utl.Str(xProp["scope"])
		fmt.Printf("%s  %s  %s %-20s %s\n", utl.Str(x["name"]), rdId, principalId, "("+principalType+")", scope)
	case "s":
		fmt.Printf("%s  %-10s  %s\n", utl.Str(x["subscriptionId"]), utl.Str(x["state"]), utl.Str(x["displayName"]))
	case "m":
		xProp := x["properties"].(map[string]interface{})
		fmt.Printf("%-38s  %-20s  %s\n", utl.Str(x["name"]), utl.Str(xProp["displayName"]), MgType(utl.Str(x["type"])))
	case "u":
		upn := utl.Str(x["userPrincipalName"])
		onPremisesSamAccountName := utl.Str(x["onPremisesSamAccountName"])
		fmt.Printf("%s  %-50s %-18s %s\n", utl.Str(x["id"]), upn, onPremisesSamAccountName, utl.Str(x["displayName"]))
	case "g":
		fmt.Printf("%s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]))
	case "sp":
		fmt.Printf("%s  %-60s %-22s %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), utl.Str(x["servicePrincipalType"]), utl.Str(x["appId"]))
	case "ap":
		fmt.Printf("%s  %-60s %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), utl.Str(x["appId"]))
	case "ad":
		builtIn := "Custom"
		if utl.Str(x["isBuiltIn"]) == "true" {
			builtIn = "BuiltIn"
		}
		enabled := "NotEnabled"
		if utl.Str(x["isEnabled"]) == "true" {
			enabled = "Enabled"
		}
		fmt.Printf("%s  %-60s  %s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), builtIn, enabled)
	}
}

func PrintObjectByUuid(uuid string, z Bundle) {
	// Search for object with given UUID and print it
	if !utl.ValidUuid(uuid) {
		os.Exit(1) // Do nothing if UUID is invalid
	}
	// Search for this UUID under all maz objects types
	list := FindAzObjectsByUuid(uuid, z)
	for _, i := range list {
		x := i.(map[string]interface{})
		if x != nil && x["mazType"] != nil {
			PrintObject(utl.Str(x["mazType"]), x, z)
		}
	}
	// Hopefully below is ever rarely seen
	if len(list) > 1 {
		fmt.Println(utl.Red("WARNING! Multiple Azure object types share this UUID!"))
	}
}

func PrintObject(t string, x map[string]interface{}, z Bundle) {
	switch t {
	case "d":
		PrintRoleDefinition(x, z)
	case "a":
		PrintRoleAssignment(x, z)
	case "s":
		PrintSubscription(x)
	case "m":
		PrintMgGroup(x)
	case "u":
		PrintUser(x, z)
	case "g":
		PrintGroup(x, z)
	case "sp":
		PrintSp(x, z)
	case "ap":
		PrintApp(x, z)
	case "ad":
		PrintAdRole(x, z)
	}
}

func PrintMemberOfs(t string, memberOf []interface{}) {
	// Print all memberOf entries
	co := utl.Red(":") // Colorize ":" text to Red
	if len(memberOf) > 0 {
		fmt.Printf(utl.Cya("memberof") + co + "\n")
		for _, i := range memberOf {
			x := i.(map[string]interface{}) // Assert as JSON object type
			Type := utl.LastElem(utl.Str(x["@odata.type"]), ".")
			fmt.Printf("  %-50s %s (%s)\n", utl.Str(x["displayName"]), utl.Str(x["id"]), Type)
		}
	}
}

func PrintSecretList(pwdCreds []interface{}) {
	// Print password credentials stanza for Apps and Sps
	if len(pwdCreds) > 0 {
		co := utl.Red(":") // Colorize ":" text to Red
		fmt.Println(utl.Cya("secrets") + co)
		for _, i := range pwdCreds {
			a := i.(map[string]interface{})
			cId := utl.Str(a["keyId"])
			cName := utl.Str(a["displayName"])
			cHint := utl.Str(a["hint"]) + "********"
			// Reformat date strings for better readability
			cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			// Check if expiring soon
			now := time.Now().Unix()
			expiry, err := utl.DateStringToEpocInt64(utl.Str(a["endDateTime"]), time.RFC3339Nano)
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			daysDiff := (expiry - now) / 86400
			if daysDiff <= 0 {
				cExpiry = utl.Red(cExpiry) // If it's expired print in red
			} else if daysDiff < 7 {
				cExpiry = utl.Yel(cExpiry) // If expiring within a week print in yellow
			}
			fmt.Printf("  %-38s  %-24s  %-40s  %-10s  %s\n", cId, cName, cHint, cStart, cExpiry)
		}
	}
}

func PrintCertificateList(certificates []interface{}) {
	// Print password credentials stanza for Apps and Sps
	if len(certificates) > 0 {
		co := utl.Red(":") // Colorize ":" text to Red
		fmt.Println(utl.Cya("certificates") + co)
		for _, i := range certificates {
			a := i.(map[string]interface{})
			cId := utl.Str(a["keyId"])
			cName := utl.Str(a["displayName"])
			cType := utl.Str(a["type"])
			// Reformat date strings for better readability
			cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			// Check if expiring soon
			now := time.Now().Unix()
			expiry, err := utl.DateStringToEpocInt64(utl.Str(a["endDateTime"]), time.RFC3339Nano)
			if err != nil {
				utl.Die(utl.Trace() + err.Error() + "\n")
			}
			daysDiff := (expiry - now) / 86400
			if daysDiff <= 0 {
				cExpiry = utl.Red(cExpiry) // If it's expired print in red
			} else if daysDiff < 7 {
				cExpiry = utl.Yel(cExpiry) // If expiring within a week print in yellow
			}
			// There's also:
			// 	"customKeyIdentifier": "09228573F93570D8113D90DA69D8DF6E2E396874",
			// 	"key": "<RSA_KEY>",
			// 	"usage": "Verify"
			fmt.Printf("  %-38s  %-24s  %-40s  %-10s  %s\n", cId, cName, cType, cStart, cExpiry)
		}
		// https://learn.microsoft.com/en-us/graph/api/application-addkey
	}
}

func PrintOwners(owners []interface{}) {
	// Print owners stanza for Apps and Sps
	if len(owners) > 0 {
		co := utl.Red(":")
		fmt.Printf(utl.Cya("owners") + co + "\n")
		for _, i := range owners {
			o := i.(map[string]interface{})
			Type, Name := "???", "???"
			Type = utl.LastElem(utl.Str(o["@odata.type"]), ".")
			switch Type {
			case "user":
				Name = utl.Str(o["userPrincipalName"])
			case "group":
				Name = utl.Str(o["displayName"])
			case "servicePrincipal":
				Name = utl.Str(o["servicePrincipalType"])
			default:
				Name = "???"
			}
			fmt.Printf("  %-50s %s (%s)\n", Name, utl.Str(o["id"]), Type)
		}
	}
}

func PrintStringMapColor(strMap map[string]string) {
	// Print string map in YAML-like format, sorted, and in color
	co := utl.Red(":")
	sortedKeys := utl.SortMapStringKeys(strMap)
	for _, k := range sortedKeys {
		v := strMap[k]
		cK := utl.Cya(utl.Str(k)) + co // Colorized key (Cyan) + colon (Red)
		fmt.Printf("  %s %s\n", cK, utl.Str(v))
	}
}

func PrintMatching(printFormat, t, specifier string, z Bundle) {
	// Print matching object or objecs in JSON format
	if utl.ValidUuid(specifier) { // Search/print single object, if it's valid UUID
		x := GetAzObjectByUuid(t, specifier, z)
		if printFormat == "json" {
			utl.PrintJson(x)
		} else if printFormat == "reg" {
			PrintObject(t, x, z)
		}
	} else {
		matchingObjects := GetObjects(t, specifier, false, z)
		if len(matchingObjects) == 1 {
			// If it's only one object, we'll try to get the Azure copy instead of using the local cache
			x := matchingObjects[0].(map[string]interface{})
			uuid := utl.Str(x["id"])
			if utl.ValidUuid(uuid) {
				x = GetAzObjectByUuid(t, uuid, z)
			}
			if printFormat == "json" {
				utl.PrintJson(x)
			} else if printFormat == "reg" {
				PrintObject(t, x, z)
			}
		} else if len(matchingObjects) > 1 {
			if printFormat == "json" {
				utl.PrintJson(matchingObjects) // Print all matching objects in JSON
			} else if printFormat == "reg" {
				for _, i := range matchingObjects { // Print all matching object teresely
					x := i.(map[string]interface{})
					PrintTersely(t, x)
				}
			}
		}
	}
	return
}

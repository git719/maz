// printing.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"os"
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

func PrintObjectById(id string, z Bundle) {
	// Search for object with given UUID and print it
	if !utl.ValidUuid(id) {
		os.Exit(1) // Do nothing if UUID is invalid
	}
	// Search for all mazType objects with this UUID
	list := GetObjectsWithThisUuid(id, z)
	for _, i := range list {
		x := i.(map[string]interface{})
		if x != nil && x["mazType"] != nil {
			PrintObject(utl.Str(x["mazType"]), x, z)
		}
	}
	// Hopefully below is ever rarely seen
	if len(list) > 1 {
		fmt.Println(utl.ColRed("WARNING! Multiple Azure object types share this UUID!"))
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
	if len(memberOf) > 0 {
		fmt.Printf("memberof:\n")
		for _, i := range memberOf {
			x := i.(map[string]interface{}) // Assert as JSON object type
			Type := utl.LastElem(utl.Str(x["@odata.type"]), ".")
			fmt.Printf("  %-50s %s (%s)\n", utl.Str(x["displayName"]), utl.Str(x["id"]), Type)
		}
	} else {
		fmt.Printf("%s: %s\n", "memberof", "None")
	}
}

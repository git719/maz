// definitions.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
	"strings"
)

func PrintRoleDefinition(x map[string]interface{}, z Bundle) {
	// Print role definition object in YAML-like format
	if x == nil {
		return
	}
	if x["name"] != nil {
		fmt.Printf("id: %s\n", utl.Str(x["name"]))
	}

	fmt.Printf("properties:\n")
	if x["properties"] == nil {
		fmt.Printf("  <Missing??>\n")
		return
	}
	xProp := x["properties"].(map[string]interface{})

	list := []string{"roleName", "description"}
	for _, i := range list {
		fmt.Printf("  %s %s\n", i+":", utl.Str(xProp[i]))
	}

	fmt.Printf("  %-18s", "assignableScopes: ")
	if xProp["assignableScopes"] == nil {
		fmt.Printf("[]\n")
	} else {
		fmt.Printf("\n")
		scopes := xProp["assignableScopes"].([]interface{})
		if len(scopes) > 0 {
			subNameMap := GetIdMapSubs(z) // Get all subscription id:name pairs
			for _, i := range scopes {
				if strings.HasPrefix(i.(string), "/subscriptions") {
					// Print subscription name as a comment at end of line
					subId := utl.LastElem(i.(string), "/")
					fmt.Printf("    - %s # %s\n", utl.Str(i), subNameMap[subId])
				} else {
					fmt.Printf("    - %s\n", utl.Str(i))
				}
			}
		} else {
			fmt.Printf("    <Not an arrays??>\n")
		}
	}

	fmt.Printf("  %-18s\n", "permissions:")
	if xProp["permissions"] == nil {
		fmt.Printf("    %s\n", "<No permissions??>")
	} else {
		permsSet := xProp["permissions"].([]interface{})
		if len(permsSet) == 1 {
			perms := permsSet[0].(map[string]interface{}) // Select the 1 expected single permission set

			fmt.Printf("    - actions:\n") // Note that this one is different, as it starts the YAML array with the dash '-'
			if perms["actions"] != nil {
				permsA := perms["actions"].([]interface{})
				if utl.GetType(permsA)[0] != '[' { // Open bracket character means it's an array list
					fmt.Printf("        <Not an array??>\n")
				} else {
					for _, i := range permsA {
						fmt.Printf("        - %s\n", utl.Str(i))
					}
				}
			}

			fmt.Printf("      notActions:\n")
			if perms["notActions"] != nil {
				permsNA := perms["notActions"].([]interface{})
				if utl.GetType(permsNA)[0] != '[' {
					fmt.Printf("        <Not an array??>\n")
				} else {
					for _, i := range permsNA {
						fmt.Printf("        - %s\n", utl.Str(i))
					}
				}
			}

			fmt.Printf("      dataActions:\n")
			if perms["dataActions"] != nil {
				permsDA := perms["dataActions"].([]interface{})
				if utl.GetType(permsDA)[0] != '[' {
					fmt.Printf("        <Not an array??>\n")
				} else {
					for _, i := range permsDA {
						fmt.Printf("        - %s\n", utl.Str(i))
					}
				}
			}

			fmt.Printf("      notDataActions:\n")
			if perms["notDataActions"] != nil {
				permsNDA := perms["notDataActions"].([]interface{})
				if utl.GetType(permsNDA)[0] != '[' {
					fmt.Printf("        <Not an array??>\n")
				} else {
					for _, i := range permsNDA {
						fmt.Printf("        - %s\n", utl.Str(i))
					}
				}
			}

		} else {
			fmt.Printf("    <More than one set??>\n")
		}
	}
}

func UpsertAzRoleDefinition(x map[string]interface{}, z Bundle) {
	// Create or Update Azure role definition as defined by give x object
	if x == nil {
		return
	}

	// $p = $x.properties
	// $name = $p.roleName
	// $scope = $p.assignableScopes[0]
	// if ( ($null -eq $p ) -or ($null -eq $name ) -or ($null -eq $scope ) -or
	//      ($null -eq $p.type ) -or ($null -eq $p.description ) ) {
	//     die("Specfile is missing required attributes.`n" +
	//         "Run script with '-kd[j]' option to create a properly formatted sample skeleton file.")
	// }

	// $existing = GetAzObjectByName "d" $name
	// if ( $null -eq $existing.name ) {
	//     print("Creating NEW role definition '{0}' as per specfile" -f $name)
	//     $roleId = [guid]::NewGuid()  # Generate a new global UUID
	// } else {
	//     print("id: {0}" -f $existing.name)
	//     PrintAzObject "d" $x  # Print the one we got from specfile
	//     warning("WARNING: Role already exists in Azure.")
	//     $Confirm = Read-Host -Prompt "UPDATE existing one with above? y/n "
	//     if ( $Confirm -ne "y" ) {
	//         die("Aborted.")
	//     }
	//     print("Updating role ...")
	//     $roleId = $existing.name  # Existing role definition UUID
	// }

	// # For the scope in the API call we can just use the 1st one
	// $body = $x | ConvertTo-Json -Depth 10
	// $url = $az_url + $scope + "/providers/Microsoft.Authorization/roleDefinitions/"
	// $r = ApiCall "PUT" ( $url + $roleId + "?api-version=2022-04-01") -data $body
	// PrintJson $r

	return
}

func DeleteAzRoleDefinitionByFqid(fqid string, z Bundle) map[string]interface{} {
	// Delete Azure resource RBAC roleDefinition by fully qualified object Id
	// Example of a fully qualified Id string:
	//   "/providers/Microsoft.Authorization/roleDefinitions/50a6ff7c-3ac5-4acc-b4f4-9a43aee0c80f"
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	url := ConstAzUrl + fqid
	r, statusCode, _ := ApiDelete(url, z.AzHeaders, params)
	//ApiErrorCheck("DELETE", url, utl.Trace(), r)
	if statusCode != 200 {
		if statusCode == 204 {
			fmt.Println("Role definition already deleted or does not exist.")
		} else {
			e := r["error"].(map[string]interface{})
			fmt.Println(e["message"].(string))
		}
	}
	return nil
}

func GetIdMapRoleDefs(z Bundle) (nameMap map[string]string) {
	// Return role definition id:name map
	nameMap = make(map[string]string)
	roleDefs := GetRoleDefinitions("", false, z) // false = don't force going to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range roleDefs {
		x := i.(map[string]interface{})
		if x["name"] != nil {
			xProp := x["properties"].(map[string]interface{})
			if xProp["roleName"] != nil {
				nameMap[utl.Str(x["name"])] = utl.Str(xProp["roleName"])
			}
		}
	}
	return nameMap
}

func GetRoleDefinitions(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all roleDefinitions that match on provided filter, empty "" filter grabs all
	// Defaults to querying local cache if it's within the cache retention period, unless force
	// boolean option is given to call Azure. The verbose option details the progress.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if cacheNoGood || force {
		list = GetAzRoleDefinitions(true, z) // Get the entire set from Azure, true = show progress
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	for _, i := range list { // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant roleDefinitions attributes
		xProp := x["properties"].(map[string]interface{})
		if utl.SubString(utl.Str(x["name"]), filter) || utl.SubString(utl.Str(xProp["roleName"]), filter) ||
			utl.SubString(utl.Str(x["description"]), filter) {
			matchingList = append(matchingList, x)
		}
	}
	return matchingList
}

func GetAzRoleDefinitions(verbose bool, z Bundle) (list []interface{}) {
	// Get ALL roleDefinitions in current Azure tenant AND save them to local cache file
	// Option to be verbose (true) or quiet (false), since it can take a while.
	// References:
	// - https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list
	// - https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list

	// Important Azure resource RBAC role definitions API note: As of api-version 2022-04-01, the filter
	// AtScopeAndBelow() does not work as documented at:
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list.

	// This means that anyone searching for a comprehensive list of ALL role definitions within an Azure tenant
	// is forced to do so by having to traverse and search for all role definitions under each MG and subscription
	// scope. This process grabs all Azure BuiltIn role definitions, as well as als all custom ones.

	list = nil         // We have to zero it out
	var uuIds []string // Keep track of each unique object to eliminate inherited repeats
	k := 1             // Track number of API calls to provide progress
	mgGroupNameMap := GetIdMapMgGroups(z)
	subNameMap := GetIdMapSubs(z)                            // Get all subscription id:name pairs
	scopes := GetAzRbacScopes(z)                             // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		scopeName := scope // Default scope name is the whole scope string
		if strings.HasPrefix(scope, "/providers") {
			scopeName = mgGroupNameMap[scope] // If it's an MG, just use its name
		} else if strings.HasPrefix(scope, "/subscriptions") {
			scopeName = subNameMap[utl.LastElem(scope, "/")] // If it's a sub, user its name
		}
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		ApiErrorCheck("GET", url, utl.Trace(), r) // DEBUG. Until ApiGet rewrite with nullable _ err
		if r != nil && r["value"] != nil {
			definitionsUnderThisScope := r["value"].([]interface{})
			u := 0 // Keep track of unique definitions in this scope
			for _, i := range definitionsUnderThisScope {
				x := i.(map[string]interface{})
				uuid := utl.Str(x["name"]) // Note that 'name' is actually the role assignment UUID
				if utl.ItemInList(uuid, uuIds) {
					continue // Role assignments DO repeat! Skip if it's already been added.
				}
				list = append(list, x)      // This one is unique, append to growing list
				uuIds = append(uuIds, uuid) // Keep track of the UUIDs we are seeing
				u++
			}
			if verbose { // Using global var rUp to overwrite last line. Defer newline until done
				fmt.Printf("%s(API calls = %d) %d unique role definitions under scope %s", rUp, k, u, scopeName)
			}
			k++
		}
	}
	if verbose {
		fmt.Printf("\n") // Use newline now
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions.json")
	utl.SaveFileJson(list, cacheFile) // Update the local cache
	return list
}

func RoleDefinitionCountLocal(z Bundle) (builtin, custom int64) {
	// Dedicated role definition local cache counter able to discern if role is custom to native tenant or it's an Azure BuilIn role
	var customList []interface{} = nil
	var builtinList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			definitions := rawList.([]interface{})
			for _, i := range definitions {
				x := i.(map[string]interface{}) // Assert as JSON object type
				xProp := x["properties"].(map[string]interface{})
				if utl.Str(xProp["type"]) == "CustomRole" {
					customList = append(customList, x)
				} else {
					builtinList = append(builtinList, x)
				}
			}
			return int64(len(builtinList)), int64(len(customList))
		}
	}
	return 0, 0
}

func RoleDefinitionCountAzure(z Bundle) (builtin, custom int64) {
	// Dedicated role definition Azure counter able to discern if role is custom to native tenant or it's an Azure BuilIn role
	var customList []interface{} = nil
	var builtinList []interface{} = nil
	definitions := GetAzRoleDefinitions(false, z) // false = be silent
	for _, i := range definitions {
		x := i.(map[string]interface{}) // Assert as JSON object type
		xProp := x["properties"].(map[string]interface{})
		if utl.Str(xProp["type"]) == "CustomRole" {
			customList = append(customList, x)
		} else {
			builtinList = append(builtinList, x)
		}
	}
	return int64(len(builtinList)), int64(len(customList))
}

func GetAzRoleDefinitionByName(specifier string, z Bundle) (y map[string]interface{}) {
	// Get Azure resource roleDefinition by displayName
	// See https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list
	y = nil
	scopes := GetAzRbacScopes(z) // Get all scopes
	params := map[string]string{
		"api-version": "2022-04-01", // roleDefinitions
		"$filter":     "roleName eq '" + specifier + "'",
	}
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		ApiErrorCheck("GET", url, utl.Trace(), r) // DEBUG. Until ApiGet rewrite with nullable _ err
		if r != nil && r["value"] != nil {
			results := r["value"].([]interface{})
			if len(results) == 1 {
				y = results[0].(map[string]interface{}) // Select first, only index entry
				return y                                // We found it
			}
		}
	}
	// If above logic ever finds than 1, then we have serious issuses, just nil below
	return nil
}

func GetAzRoleDefinitionByObject(x map[string]interface{}, z Bundle) (y map[string]interface{}) {
	// Get Azure resource RBAC role definition object if it exists exactly as x object.
	// Looks for matching: displayName and assignableScopes

	// First, make sure x is a searchable role definition object
	if x == nil { // Don't look for empty objects
		return nil
	}
	xProp := x["properties"].(map[string]interface{})
	if xProp == nil {
		return nil
	}

	xScopes := xProp["assignableScopes"].([]interface{})
	if utl.GetType(xScopes)[0] != '[' || len(xScopes) < 1 {
		return nil // Return nil if assignableScopes not an array, or it's empty
	}
	xRoleName := utl.Str(xProp["roleName"])
	if xRoleName == "" {
		return nil
	}

	// Look for x under all its scopes
	for _, i := range xScopes {
		scope := utl.Str(i)
		if scope == "/" {
			scope = ""
		} // Highly unlikely but just to avoid an err
		// Get all role assignments for xPrincipalId under xScope
		params := map[string]string{
			"api-version": "2022-04-01", // roleDefinitions
			"$filter":     "roleName eq '" + xRoleName + "'",
		}
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		ApiErrorCheck("GET", url, utl.Trace(), r)
		if r != nil && r["value"] != nil {
			results := r["value"].([]interface{})
			if len(results) == 1 {
				y = results[0].(map[string]interface{}) // Select first index entry
				return y                                // We found it
			} else {
				return nil // If there's more than one entry we have other problems, so just return nil
			}
		}
	}
	return nil
}

func GetAzRoleDefinitionByUuid(uuid string, z Bundle) (x map[string]interface{}) {
	// Get Azure resource roleDefinitions by Object Id. Unfortunately we have to traverse
	// and search the ENTIRE Azure resource scope hierarchy, which can take time.
	x = nil
	scopes := GetAzRbacScopes(z)                             // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		//ApiErrorCheck("GET", url, utl.Trace(), r) // DEBUG. Until ApiGet rewrite with nullable _ err
		if r != nil && r["value"] != nil {
			definitionsUnderThisScope := r["value"].([]interface{})
			for _, i := range definitionsUnderThisScope {
				x := i.(map[string]interface{})
				if utl.Str(x["name"]) == uuid {
					return x // Return immediately if found
				}
			}
		}
	}
	return x
}

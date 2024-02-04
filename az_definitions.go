// az_definitions.go
// Azure resource RBAC role definitions

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"github.com/google/uuid"
	"path/filepath"
	"strings"
)

func PrintRoleDefinition(x map[string]interface{}, z Bundle) {
	// Print role definition object in YAML-like format
	if x == nil {
		return
	}
	if x["name"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(utl.Str(x["name"])))
	}
	if x["properties"] != nil {
		fmt.Println(utl.Blu("properties") + ":")
	} else {
		fmt.Println(utl.Red("  <Missing properties??>"))
	}

	xProp := x["properties"].(map[string]interface{})

	list := []string{"roleName", "description"}
	for _, i := range list {
		fmt.Printf("  %s: %s\n", utl.Blu(i), utl.Gre(utl.Str(xProp[i])))
	}

	fmt.Printf("  %s: ", utl.Blu("assignableScopes"))
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
					comment := "# " + subNameMap[subId]
					fmt.Printf("    - %s  %s\n", utl.Gre(utl.Str(i)), comment)
				} else {
					fmt.Printf("    - %s\n", utl.Gre(utl.Str(i)))
				}
			}
		} else {
			fmt.Printf(utl.Red("    <Not an arrays??>\n"))
		}
	}

	fmt.Printf("  %s:\n", utl.Blu("permissions"))
	if xProp["permissions"] == nil {
		fmt.Printf(utl.Red("    < No permissions?? >\n"))
	} else {
		permsSet := xProp["permissions"].([]interface{})
		if len(permsSet) == 1 {
			perms := permsSet[0].(map[string]interface{}) // Select the 1 expected single permission set

			fmt.Printf("    - " + utl.Blu("actions") + ":\n") // Note that this one is different, as it starts the YAML array with the dash '-'
			if perms["actions"] != nil {
				permsA := perms["actions"].([]interface{})
				if utl.GetType(permsA)[0] != '[' { // Open bracket character means it's an array list
					fmt.Printf(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsA {
						fmt.Printf("        - %s\n", utl.Gre(utl.Str(i)))
					}
				}
			}

			fmt.Printf("      " + utl.Blu("notActions") + ":\n")
			if perms["notActions"] != nil {
				permsNA := perms["notActions"].([]interface{})
				if utl.GetType(permsNA)[0] != '[' {
					fmt.Printf(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsNA {
						fmt.Printf("        - %s\n", utl.Gre(utl.Str(i)))
					}
				}
			}

			fmt.Printf("      " + utl.Blu("dataActions") + ":\n")
			if perms["dataActions"] != nil {
				permsDA := perms["dataActions"].([]interface{})
				if utl.GetType(permsDA)[0] != '[' {
					fmt.Printf(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsDA {
						fmt.Printf("        - %s\n", utl.Gre(utl.Str(i)))
					}
				}
			}

			fmt.Printf("      " + utl.Blu("notDataActions") + ":\n")
			if perms["notDataActions"] != nil {
				permsNDA := perms["notDataActions"].([]interface{})
				if utl.GetType(permsNDA)[0] != '[' {
					fmt.Printf(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsNDA {
						fmt.Printf("        - %s\n", utl.Gre(utl.Str(i)))
					}
				}
			}

		} else {
			fmt.Printf(utl.Red("    <More than one set??>\n"))
		}
	}
}

func UpsertAzRoleDefinition(x map[string]interface{}, z Bundle) {
	// Create or Update Azure role definition as defined by give x object
	if x == nil {
		return
	}
	xProp := x["properties"].(map[string]interface{})
	xRoleName := utl.Str(xProp["roleName"])
	xType := utl.Str(xProp["type"])
	xDesc := utl.Str(xProp["description"])
	xScopes := xProp["assignableScopes"].([]interface{})
	xScope1 := utl.Str(xScopes[0]) // For deployment, we'll use 1st scope
	if xProp == nil || xScopes == nil || xRoleName == "" || xScope1 == "" ||
		xDesc == "" || strings.ToLower(xType) != "customrole" {
		utl.Die("Specfile is missing required attributes. Need at least:\n\n" +
			"properties:\n" +
			"  type: CustomRole\n" +
			"  roleName: \"My Role Name\"\n" +
			"  description: \"My role's description\"\n" +
			"  assignableScopes:\n" +
			"    - \"/subscriptions/UUID\"  # At least one scope\n\n" +
			"See script '-k*' options to create properly formatted sample files.\n")
	}

	roleId := ""
	existing := GetAzRoleDefinitionByName(xRoleName, z)
	if existing == nil {
		// Role definition doesn't exist, so we're creating a new one
		roleId = uuid.New().String() // Generate a new global UUID in string format
	} else {
		// Role exists, we'll prompt for update choice
		PrintRoleDefinition(existing, z)
		msg := utl.Yel("Role already exists! UPDATE it? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Aborted.\n")
		}
		fmt.Println("Updating role ...")
		roleId = utl.Str(existing["name"])
	}

	payload := x                                             // Obviously using x object as the payload
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	url := ConstAzUrl + xScope1 + "/providers/Microsoft.Authorization/roleDefinitions/" + roleId
	r, statusCode, _ := ApiPut(url, z, payload, params)
	if statusCode == 201 {
		PrintRoleDefinition(r, z) // Print the newly updated object
	} else {
		e := r["error"].(map[string]interface{})
		fmt.Println(e["message"].(string))
	}
	return
}

func DeleteAzRoleDefinitionByFqid(fqid string, z Bundle) map[string]interface{} {
	// Delete Azure resource RBAC roleDefinition by fully qualified object Id
	// Example of a fully qualified Id string:
	//   "/providers/Microsoft.Authorization/roleDefinitions/50a6ff7c-3ac5-4acc-b4f4-9a43aee0c80f"
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	url := ConstAzUrl + fqid
	r, statusCode, _ := ApiDelete(url, z, params)
	//ApiErrorCheck("DELETE", url, utl.Trace(), r)
	if statusCode != 200 {
		if statusCode == 204 {
			fmt.Println("Role definition already deleted or does not exist. Give Azure a minute to flush it out.")
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
	roleDefs := GetMatchingRoleDefinitions("", false, z) // false = don't force going to Azure
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

func RoleDefinitionCountLocal(z Bundle) (builtin, custom int64) {
	// Dedicated role definition local cache counter able to discern if role is custom to native tenant or it's an Azure BuilIn role
	var customList []interface{} = nil
	var builtinList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions."+ConstCacheFileExtension)
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJsonGzip(cacheFile)
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
	definitions := GetAzRoleDefinitions(z, false) // false = be silent
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

func DiffRoleDefinitionSpecfileVsAzure(a, b map[string]interface{}, z Bundle) {
	// Prints differences between role definition in Specfile (a) vs what is in Azure (b). The
	// calling function must ensure that both a & b are valid role definition objects from the
	// specfile and from Azure. A generic DiffJsonObject() function would probably be better for this.

	// Gather the Azure object values
	azureId := utl.Str(b["name"])
	azureProp := b["properties"].(map[string]interface{})
	azureRoleName := utl.Str(azureProp["roleName"])
	azureDesc := utl.Str(azureProp["description"])
	azureScopes := azureProp["assignableScopes"].([]interface{})
	azurePermSet := azureProp["permissions"].([]interface{})
	azurePerms := azurePermSet[0].(map[string]interface{})
	azurePermsA := azurePerms["actions"].([]interface{})
	azurePermsNA := azurePerms["notActions"].([]interface{})
	azurePermsDA := azurePerms["dataActions"].([]interface{})
	azurePermsNDA := azurePerms["notDataActions"].([]interface{})

	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(azureId))
	fmt.Println(utl.Blu("properties") + ":")

	// Gather the specfile object values
	fileProp := b["properties"].(map[string]interface{})
	fileRoleName := utl.Str(fileProp["roleName"])
	fileDesc := utl.Str(fileProp["description"])
	fileScopes := fileProp["assignableScopes"].([]interface{})
	filePermSet := fileProp["permissions"].([]interface{})
	filePerms := filePermSet[0].(map[string]interface{})
	filePermsA := filePerms["actions"].([]interface{})
	filePermsNA := filePerms["notActions"].([]interface{})
	filePermsDA := filePerms["dataActions"].([]interface{})
	filePermsNDA := filePerms["notDataActions"].([]interface{})

	// Display differences
	fmt.Printf("  %s: %s\n", utl.Blu("roleName"), utl.Gre(azureRoleName))
	if fileRoleName != azureRoleName {
		fmt.Printf("  %s: %s\n", utl.Blu("roleName"), utl.Red(fileRoleName))
	}
	
	fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Gre(azureDesc))
	if fileDesc != azureDesc {
		fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Red(fileDesc))
	}
	
	fmt.Printf("  %s: %s\n", utl.Blu("assignableScopes"), utl.Gre(azureScopes))
	sameLen := true // Assume they are both the same length
	if len(fileScopes) != len(azureScopes) {
		sameLen = false
	}
	for k, v := range azureScopes {
		azureVal := utl.Str(v)
		fmt.Printf("    - %s\n", utl.Gre(azureVal))
        if sameLen {
			fileVal := utl.Str(fileScopes[k])
			if fileVal != azureVal {
				fmt.Printf("    - %s\n", utl.Red(fileVal))
			}
		}
	}

	if len(fileScopes) != len(azureScopes) {
		fmt.Printf("  %s: %s\n", utl.Blu("assignableScopes"), utl.Red(fileScopes))
	}
}

func GetMatchingRoleDefinitions(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all RBAC role definitions matching on 'filter'; return entire list if filter is empty ""

	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions."+ConstCacheFileExtension)
	cacheFileAge := utl.FileAge(cacheFile)
	if utl.InternetIsAvailable() && (force || cacheFileAge == 0 || cacheFileAge > ConstAzCacheFileAgePeriod) {
		// If Internet is available AND (force was requested OR cacheFileAge is zero (meaning does not exist)
		// OR it is older than ConstAzCacheFileAgePeriod) then query Azure directly to get all objects
		// and show progress while doing so (true = verbose below)
		list = GetAzRoleDefinitions(z, true)
	} else {
		// Use local cache for all other conditions
		list = GetCachedObjects(cacheFile)
	}

	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	for _, i := range list { // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant strings within roleDefinitions JSON object (Note: Not all attributes are maintained)
		if utl.StringInJson(x, filter) {
			matchingList = append(matchingList, x)
		}
	}
	return matchingList
}

func GetAzRoleDefinitions(z Bundle, verbose bool) (list []interface{}) {
	// Get all roleDefinitions in current Azure tenant and save them to local cache file
	// Option to be verbose (true) or quiet (false), since it can take a while.
	// References:
	//   https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list
	//   https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list

	list = nil         // We have to zero it out
	var uuIds []string // Keep track of each unique object to eliminate inherited repeats
	k := 1             // Track number of API calls to provide progress

	var mgGroupNameMap, subNameMap map[string]string
	if verbose {
		mgGroupNameMap = GetIdMapMgGroups(z)
		subNameMap = GetIdMapSubs(z)
	}

	scopes := GetAzRbacScopes(z)                             // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z, params)
		if r != nil && r["value"] != nil {
			objectsUnderThisScope := r["value"].([]interface{})
			count := 0
			for _, i := range objectsUnderThisScope {
				x := i.(map[string]interface{})
				uuid := utl.Str(x["name"])
				if utl.ItemInList(uuid, uuIds) {
					// Role definitions & assignments do repeat!
					continue // Skip if already seen
				}
				uuIds = append(uuIds, uuid) // Keep track of the UUIDs we are seeing
				list = append(list, x)
				count++
			}
			if verbose && count > 0 {
				scopeName := scope
				if strings.HasPrefix(scope, "/providers") {
					scopeName = mgGroupNameMap[scope]
				} else if strings.HasPrefix(scope, "/subscriptions") {
					scopeName = subNameMap[utl.LastElem(scope, "/")]
				}
				fmt.Printf("API call %4d: %5d objects under %s\n", k, count, scopeName)
			}
		}
		k++
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions."+ConstCacheFileExtension)
	utl.SaveFileJsonGzip(list, cacheFile) // Update the local cache
	return list
}

func GetAzRoleDefinitionByName(roleName string, z Bundle) (y map[string]interface{}) {
	// Get Azure resource roleDefinition by displayName
	// See https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list
	y = nil
	scopes := GetAzRbacScopes(z) // Get all scopes
	params := map[string]string{
		"api-version": "2022-04-01", // roleDefinitions
		"$filter":     "roleName eq '" + roleName + "'",
	}
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(url, z, params)
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
		r, _, _ := ApiGet(url, z, params)
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

func GetAzRoleDefinitionByUuid(uuid string, z Bundle) map[string]interface{} {
	// Get Azure resource roleDefinitions by Object Id. Unfortunately we have to iterate
	// through the entire tenant scope hierarchy, which can take time.
	scopes := GetAzRbacScopes(z)
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions/" + uuid
		r, _, _ := ApiGet(url, z, params)
		if r != nil && r["id"] != nil {
			return r // Return as soon as we find a match
		}
	}
	return nil
}

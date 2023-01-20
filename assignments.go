// assignments.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"github.com/google/uuid"
	"path/filepath"
	"strings"
)

func PrintRoleAssignment(x map[string]interface{}, z Bundle) {
	// Print role definition object in YAML-like
	if x == nil {
		return
	}
	co := utl.Red(":") // Colorize ":" text to Red
	if x["name"] != nil {
		cId := utl.Cya("id") + co // Colorize "id" text to Cyan
		fmt.Printf("%s %s\n", cId, utl.Str(x["name"]))
	}
	if x["properties"] != nil {
		fmt.Println(utl.Cya("properties") + co)
	} else {
		fmt.Println("  < Missing properties? What's going? >")
	}

	xProp := x["properties"].(map[string]interface{})

	roleNameMap := GetIdMapRoleDefs(z) // Get all role definition id:name pairs
	roleId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
	cRoleDefinitionId := utl.Cya("roleDefinitionId") + co
	cComment := utl.Blu("# roleName = \"" + roleNameMap[roleId] + "\"") // Blue comments
	fmt.Printf("  %-17s %s  %s\n", cRoleDefinitionId, roleId, cComment)

	var principalNameMap map[string]string = nil
	pType := utl.Str(xProp["principalType"])
	switch pType {
	case "Group":
		principalNameMap = GetIdMapGroups(z) // Get all users id:name pairs
	case "User":
		principalNameMap = GetIdMapUsers(z) // Get all users id:name pairs
	case "ServicePrincipal":
		principalNameMap = GetIdMapSps(z) // Get all SPs id:name pairs
	default:
		pType = "not provided"
	}
	principalId := utl.Str(xProp["principalId"])
	pName := principalNameMap[principalId]
	if pName == "" {
		pName = "???"
	}
	cPrincipalId := utl.Cya("principalId") + co
	cComment = utl.Blu("# principalType = " + pType + ", displayName = \"" + pName + "\"")
	fmt.Printf("  %-17s %s  %s\n", cPrincipalId, principalId, cComment)

	subNameMap := GetIdMapSubs(z) // Get all subscription id:name pairs
	scope := utl.Str(xProp["scope"])
	if scope == "" {
		scope = utl.Str(xProp["Scope"])
	} // Account for possibly capitalized key
	cScope := utl.Cya("scope") + co
	if strings.HasPrefix(scope, "/subscriptions") {
		split := strings.Split(scope, "/")
		subName := subNameMap[split[2]]
		cComment = utl.Blu("# Sub = " + subName)
		fmt.Printf("  %-17s %s  %s\n", cScope, scope, cComment)
	} else if scope == "/" {
		cComment = utl.Blu("# Entire tenant")
		fmt.Printf("  %-17s %s  %s\n", cScope, scope, cComment)
	} else {
		fmt.Printf("  %-17s %s\n", cScope, scope)
	}
}

func PrintRoleAssignmentReport(z Bundle) {
	// Print a human-readable report of all role assignments
	roleNameMap := GetIdMapRoleDefs(z) // Get all role definition id:name pairs
	subNameMap := GetIdMapSubs(z)      // Get all subscription id:name pairs
	groupNameMap := GetIdMapGroups(z)  // Get all users id:name pairs
	userNameMap := GetIdMapUsers(z)    // Get all users id:name pairs
	spNameMap := GetIdMapSps(z)        // Get all SPs id:name pairs

	assignments := GetAzRoleAssignments(false, z)
	for _, i := range assignments {
		x := i.(map[string]interface{})
		xProp := x["properties"].(map[string]interface{})
		Rid := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
		principalId := utl.Str(xProp["principalId"])
		Type := utl.Str(xProp["principalType"])
		pName := "ID-Not-Found"
		switch Type {
		case "Group":
			pName = groupNameMap[principalId]
		case "User":
			pName = userNameMap[principalId]
		case "ServicePrincipal":
			pName = spNameMap[principalId]
		}

		Scope := utl.Str(xProp["scope"])
		if strings.HasPrefix(Scope, "/subscriptions") {
			// Replace sub ID to name
			split := strings.Split(Scope, "/")
			// Map subscription Id to its name + the rest of the resource path
			Scope = subNameMap[split[2]] + " " + strings.Join(split[3:], "/")
		}
		Scope = strings.TrimSpace(Scope)

		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\"\n", roleNameMap[Rid], pName, Type, Scope)
	}
}

func CreateAzRoleAssignment(x map[string]interface{}, z Bundle) {
	// Create Azure role assignment as defined by give x object
	if x == nil {
		return
	}
	xProp := x["properties"].(map[string]interface{})
	roleDefinitionId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/") // Note we only care about the UUID
	principalId := utl.Str(xProp["principalId"])
	scope := utl.Str(xProp["scope"])
	if scope == "" {
		scope = utl.Str(xProp["Scope"]) // Account for possibly capitalized key
	}
	if roleDefinitionId == "" || principalId == "" || scope == "" {
		utl.Die("Specfile is missing required attributes. Need at least:\n\n" +
			"properties:\n" +
			"    roleDefinitionId: <UUID or fully_qualified_roleDefinitionId>\n" +
			"    principalId: <UUID>\n" +
			"    scope: <resource_path_scope>\n\n" +
			"See script '-k*' options to create properly formatted sample files.\n")
	}

	// Note, there is no need to pre-check if assignment exists, since call will simply let us know
	newUuid := uuid.New().String() // Generate a new global UUID in string format
	payload := map[string]interface{}{
		"properties": map[string]string{
			"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionId,
			"principalId":      principalId,
		},
	}
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments/" + newUuid
	r, statusCode, _ := ApiPut(url, payload, z.AzHeaders, params)
	//ApiErrorCheck("PUT", url, utl.Trace(), r)
	if statusCode == 200 || statusCode == 201 {
		utl.PrintYaml(r)
	} else {
		e := r["error"].(map[string]interface{})
		fmt.Println(e["message"].(string))
	}
	return
}

func DeleteAzRoleAssignmentByFqid(fqid string, z Bundle) map[string]interface{} {
	// Delete Azure resource RBAC roleAssignments by fully qualified object Id
	// Example of a fully qualified Id string:
	//   "/providers/Microsoft.Management/managementGroups/33550b0b-2929-4b4b-adad-cccc66664444 +
	//    /providers/Microsoft.Authorization/roleAssignments/5d586a7b-3f4b-4b5c-844a-3fa8efe49ab3"
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	url := ConstAzUrl + fqid
	r, statusCode, _ := ApiDelete(url, z.AzHeaders, params)
	//ApiErrorCheck("DELETE", url, utl.Trace(), r)
	if statusCode != 200 {
		if statusCode == 204 {
			fmt.Println("Role assignment already deleted or does not exist. Give Azure a minute to flush it out.")
		} else {
			e := r["error"].(map[string]interface{})
			fmt.Println(e["message"].(string))
		}
	}
	return nil
}

func RoleAssignmentsCountLocal(z Bundle) int64 {
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func RoleAssignmentsCountAzure(z Bundle) int64 {
	list := GetAzRoleAssignments(false, z) // false = quiet
	return int64(len(list))
}

func GetRoleAssignments(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all roleAssignments that match on provided filter. An empty "" filter means return
	// all of them. It always uses local cache if it's within the cache retention period. The
	// force boolean option will force a call to Azure.
	// See https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-list-rest
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if cacheNoGood || force {
		list = GetAzRoleAssignments(true, z) // Get the entire set from Azure, true = show progress
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	roleNameMap := GetIdMapRoleDefs(z) // Get all role definition id:name pairs
	for _, i := range list {           // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant roleDefinitions attributes
		xProp := x["properties"].(map[string]interface{})
		rdId := utl.Str(xProp["roleDefinitionId"])
		roleName := roleNameMap[utl.LastElem(rdId, "/")]
		principalId := utl.Str(xProp["principalId"])
		description := utl.Str(xProp["description"])
		principalType := utl.Str(xProp["principalType"])
		scope := utl.Str(xProp["scope"])
		if utl.SubString(utl.Str(x["name"]), filter) || utl.SubString(rdId, filter) ||
			utl.SubString(roleName, filter) || utl.SubString(principalId, filter) ||
			utl.SubString(description, filter) || utl.SubString(principalType, filter) ||
			utl.SubString(scope, filter) {
			matchingList = append(matchingList, x)
		}
	}
	return matchingList
}

func GetAzRoleAssignments(verbose bool, z Bundle) (list []interface{}) {
	// Get ALL roleAssignments in current Azure tenant AND save them to local cache file
	// Option to be verbose (true) or quiet (false), since it can take a while.
	// References:
	// - https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-list-rest
	// - https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-subscription
	list = nil         // We have to zero it out
	var uuIds []string // Keep track of each unique objects to eliminate inherited repeats
	k := 1             // Track number of API calls to provide progress
	mgGroupNameMap := GetIdMapMgGroups(z)
	subNameMap := GetIdMapSubs(z)                            // Get all subscription id:name pairs
	scopes := GetAzRbacScopes(z)                             // Get all RBAC hierarchy scopes to search for all role assignments
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	for _, scope := range scopes {
		scopeName := scope // Default scope name is the whole scope string
		if strings.HasPrefix(scope, "/providers") {
			scopeName = mgGroupNameMap[scope] // If it's an MG, just use its name
		} else if strings.HasPrefix(scope, "/subscriptions") {
			scopeName = subNameMap[utl.LastElem(scope, "/")] // If it's a sub, user its name
		}
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		ApiErrorCheck("GET", url, utl.Trace(), r)
		if r["value"] != nil {
			assignmentsUnderThisScope := r["value"].([]interface{})
			u := 0 // Keep track of assignments in this scope
			for _, i := range assignmentsUnderThisScope {
				x := i.(map[string]interface{})
				uuid := utl.Str(x["name"]) // Note that 'name' is actually the role assignment UUID
				if utl.ItemInList(uuid, uuIds) {
					continue // Role assignments DO repeat! Skip if it's already been added.
				}
				list = append(list, x)      // This one is unique, append to growing list
				uuIds = append(uuIds, uuid) // Keep track of the UUIDs we are seeing
				u++
			}
			if verbose {
				fmt.Printf("%s(API calls = %d) %d role assignments under scope %s", rUp, k, u, scopeName)
			}
		}
		k++
	}
	if verbose {
		fmt.Printf("\n") // Use newline now
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments.json")
	utl.SaveFileJson(list, cacheFile) // Update the local cache
	return list
}

func GetAzRoleAssignmentByObject(x map[string]interface{}, z Bundle) (y map[string]interface{}) {
	// Get Azure resource RBAC role assignment object by matching given objects: roleId, principalId,
	// and scope (the 3 parameters which make a role assignment unique)

	// First, make sure x is a searchable role assignment object
	if x == nil {
		return nil
	}
	xProp := x["properties"].(map[string]interface{})
	if xProp == nil {
		return nil
	}

	xRoleDefinitionId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
	xPrincipalId := utl.Str(xProp["principalId"])
	xScope := utl.Str(xProp["scope"])
	if xScope == "" {
		xScope = utl.Str(xProp["Scope"]) // Account for possibly capitalized key
	}
	if xScope == "" || xPrincipalId == "" || xRoleDefinitionId == "" {
		return nil
	}

	// Get all role assignments for xPrincipalId under xScope
	params := map[string]string{
		"api-version": "2022-04-01", // roleAssignments
		"$filter":     "principalId eq '" + xPrincipalId + "'",
	}
	url := ConstAzUrl + xScope + "/providers/Microsoft.Authorization/roleAssignments"
	r, _, _ := ApiGet(url, z.AzHeaders, params)
	//ApiErrorCheck("GET", url, utl.Trace(), r)
	if r != nil && r["value"] != nil {
		results := r["value"].([]interface{})
		fmt.Println(len(results))
		for _, i := range results {
			y = i.(map[string]interface{})
			yProp := y["properties"].(map[string]interface{})
			yScope := utl.Str(yProp["scope"])
			yRoleDefinitionId := utl.LastElem(utl.Str(yProp["roleDefinitionId"]), "/")
			if yScope == xScope && yRoleDefinitionId == xRoleDefinitionId {
				return y // As soon as we find it
			}
		}
	}
	return nil // If we get here, we didn't fine it, so return nil
}

func GetAzRoleAssignmentByUuid(uuid string, z Bundle) (x map[string]interface{}) {
	// Get Azure resource roleAssignment by Object UUID. Unfortunately we have to traverse
	// and search the ENTIRE Azure resource scope hierarchy, which can take time.
	x = nil
	scopes := GetAzRbacScopes(z)                             // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	for _, scope := range scopes {
		url := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		r, _, _ := ApiGet(url, z.AzHeaders, params)
		//ApiErrorCheck("GET", url, utl.Trace(), r) // DEBUG. Until ApiGet rewrite with nullable _ err
		if r != nil && r["value"] != nil {
			assignmentsUnderThisScope := r["value"].([]interface{})
			for _, i := range assignmentsUnderThisScope {
				x := i.(map[string]interface{})
				if utl.Str(x["name"]) == uuid {
					return x // Return immediately if found
				}
			}
		}
	}
	return x
}

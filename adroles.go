// adroles.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
)

func PrintAdRole(x map[string]interface{}, z Bundle) {
	// Print Azure AD role definition object in YAML-like format
	if x == nil {
		return
	}
	// Print the most important attributes first
	list := []string{"id", "displayName", "description"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s: %s\n", i, v)
		}
	}

	// Commenting this out for now. Too chatty. User can run -adj JSON to get the full list of perms.
	// // List permissions
	// if x["rolePermissions"] != nil {
	// 	rolePerms := x["rolePermissions"].([]interface{})
	// 	if len(rolePerms) > 0 {
	// 		// Unclear why rolePermissions is a list instead of the single entry that it usually is
	// 		perms := rolePerms[0].(map[string]interface{})
	// 		if perms["allowedResourceActions"] != nil && len(perms["allowedResourceActions"].([]interface{})) > 0 {
	// 			fmt.Printf("permissions:\n")
	// 			for _, i := range perms["allowedResourceActions"].([]interface{}) {
	// 				fmt.Printf("  %s\n", utl.Str(i))
	// 			}
	// 		}
	// 	}
	// }

	// Print members of this role
	// See https://github.com/microsoftgraph/microsoft-graph-docs/blob/main/api-reference/v1.0/api/directoryrole-list-members.md
	url := ConstMgUrl + "/v1.0/directoryRoles(roleTemplateId='" + utl.Str(x["templateId"]) + "')/members"
	r := ApiGet(url, z.MgHeaders, nil)
	if r["value"] != nil {
		members := r["value"].([]interface{})
		if len(members) > 0 {
			fmt.Printf("members:\n")
			for _, i := range members {
				m := i.(map[string]interface{})
				fmt.Printf("  %s  %-40s   %s\n", utl.Str(m["id"]), utl.Str(m["userPrincipalName"]), utl.Str(m["displayName"]))
			}
		} else {
			fmt.Printf("%s: %s\n", "members", "None")
		}
	} else {
		fmt.Printf("members:\n  No members in this role (templateId = %s). Maybe not yet activated?\n", utl.Str(x["templateId"]))
	}
}

func AdRolesCountLocal(z Bundle) int64 {
	// Return count of Azure AD directory role entries in local cache file
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_directoryRoles.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func AdRolesCountAzure(z Bundle) int64 {
	// Return count of Azure AD directory role entries in current tenant
	// Note that endpoint "/v1.0/directoryRoles" is for Activated AD roles, so it wont give us
	// the full count of all AD roles. Also, the actual role definitions, with what permissions
	// each has is at endpoint "/v1.0/roleManagement/directory/roleDefinitions", but because
	// we only care about their count it is easier to just call end point
	// "/v1.0/directoryRoleTemplates" which is a quicker API call and has the accurate count.
	// It's not clear why MSFT makes this so darn confusing.
	url := ConstMgUrl + "/v1.0/directoryRoleTemplates"
	r := ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck(r, utl.Trace())
	if r["value"] != nil {
		return int64(len(r["value"].([]interface{})))
	}
	return 0
}

func GetAdRoles(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all Azure AD role definitions whose searchAttributes match on 'filter'. An empty "" filter returns all.
	// Uses local cache if it's less than cachePeriod old. The 'force' option forces calling Azure query.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_directoryRoles.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if cacheNoGood || force {
		list = GetAzAdRoles(cacheFile, z.MgHeaders, true) // Get all from Azure and show progress (verbose = true)
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	searchAttributes := []string{"id", "displayName", "description", "templateId"}
	var ids []string // Keep track of each unique objects to eliminate repeats
	for _, i := range list {
		x := i.(map[string]interface{})
		id := utl.Str(x["id"])
		for _, i := range searchAttributes {
			if utl.SubString(utl.Str(x[i]), filter) && !utl.ItemInList(id, ids) {
				matchingList = append(matchingList, x)
				ids = append(ids, id)
			}
		}
	}
	return matchingList
}

func GetAzAdRoles(cacheFile string, headers map[string]string, verbose bool) (list []interface{}) {
	// Get all Azure AD role definitions in current tenant AND save them to local cache file.
	// Usually a short list, so verbose is ignored, and not used.
	// See https://learn.microsoft.com/en-us/graph/api/rbacapplication-list-roledefinitions

	// There's no API delta options for this object (too short a list?), so just one call
	url := ConstMgUrl + "/v1.0/roleManagement/directory/roleDefinitions"
	r := ApiGet(url, headers, nil)
	ApiErrorCheck(r, utl.Trace())
	if r["value"] == nil {
		return nil
	}
	list = r["value"].([]interface{})
	utl.SaveFileJson(list, cacheFile) // Update the local cache
	return list
}

func GetAzAdRoleByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure AD role definition by Object UUID, with extended attributes
	// Note that role definitions are under a different area, until they are activated
	baseUrl := ConstMgUrl + "/v1.0/roleManagement/directory/roleDefinitions"
	selection := "?$select=id,displayName,description,isBuiltIn,isEnabled,resourceScopes,"
	selection += "templateId,version,rolePermissions,inheritsPermissionsFrom"
	url := baseUrl + "/" + uuid + selection
	r := ApiGet(url, headers, nil)
	//ApiErrorCheck(r, utl.Trace()) // Commented out to do this quietly. Use for DEBUGging
	return r
}

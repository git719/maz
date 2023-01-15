// helper.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"os"
	"path/filepath"
	"time"
)

func GetObjectById(t, id string, z Bundle) (x map[string]interface{}) {
	// Retrieve Azure object by Object Id
	switch t {
	case "d":
		return GetAzRoleDefinitionById(id, z)
	case "a":
		return GetAzRoleAssignmentById(id, z)
	case "s":
		return GetAzSubscriptionById(id, z.AzHeaders)
	case "u":
		return GetAzUserById(id, z.MgHeaders)
	case "g":
		return GetAzGroupById(id, z.MgHeaders)
	case "sp":
		return GetAzSpById(id, z.MgHeaders)
	case "ap":
		return GetAzAppById(id, z.MgHeaders)
	case "ad":
		return GetAzAdRoleById(id, z.MgHeaders)
	}
	return nil
}

func GetAzRbacScopes(z Bundle) (scopes []string) {
	// NOT USED: LOOKING TO DELETE THIS
	// Get all scopes from the Azure RBAC hierarchy
	scopes = nil
	managementGroups := GetAzMgGroups(z) // Let's start with all managementGroups scopes
	for _, i := range managementGroups {
		x := i.(map[string]interface{})
		scopes = append(scopes, utl.Str(x["id"]))
	}
	subscriptions := GetAzSubscriptions(z) // Now add all the subscription scopes
	for _, i := range subscriptions {
		x := i.(map[string]interface{})
		// Skip legacy subscriptions, since they have no role definitions and calling them causes an error
		if utl.Str(x["displayName"]) == "Access to Azure Active Directory" {
			continue
		}
		scopes = append(scopes, utl.Str(x["id"]))
		// BELOW NOT REALLY NEEDED
		// // Now get/add all resourceGroups under this subscription
		// params := map[string]string{"api-version": "2021-04-01"} // resourceGroups
		// url := ConstAzUrl + utl.Str(x["id"]) + "/resourcegroups"
		// r := ApiGet(url, z.AzHeaders, params)
		// ApiErrorCheck(r, utl.Trace())
		// if r != nil && r["value"] != nil {
		// 	resourceGroups := r["value"].([]interface{})
		// 	for _, j := range resourceGroups {
		// 		y := j.(map[string]interface{})
		// 		scopes = append(scopes, utl.Str(y["id"]))
		// 	}
		// }
	}
	return scopes
}

func CheckLocalCache(cacheFile string, cachePeriod int64) (usable bool, cachedList []interface{}) {
	// Return locally cached list of objects if it exists *and* it is within the specified cachePeriod in seconds
	if utl.FileUsable(cacheFile) {
		cacheFileEpoc := int64(utl.FileModTime(cacheFile))
		cacheFileAge := int64(time.Now().Unix()) - cacheFileEpoc
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			if len(cachedList) > 0 && cacheFileAge < cachePeriod {
				return false, cachedList // Cache is usable, returning cached list
			}
		}
	}
	return true, nil // Cache is not usable, returning nil
}

func GetObjects(t, filter string, force bool, z Bundle) (list []interface{}) {
	// Generic function to get objects of type t whose attributes match on filter.
	// If filter is the "" empty string return ALL of the objects of this type.
	switch t {
	case "d":
		return GetRoleDefinitions(filter, force, z)
	case "a":
		return GetRoleAssignments(filter, force, z)
	case "s":
		return GetSubscriptions(filter, force, z)
	case "m":
		return GetMgGroups(filter, force, z)
	case "u":
		return GetUsers(filter, force, z)
	case "g":
		return GetGroups(filter, force, z)
	case "sp":
		return GetSps(filter, force, z)
	case "ap":
		return GetApps(filter, force, z)
	case "ad":
		return GetAdRoles(filter, force, z)
	}
	return nil
}

func GetAzObjects(url string, headers map[string]string, verbose bool) (deltaSet []interface{}, deltaLinkMap map[string]string) {
	// Generic Azure object deltaSet retriever function. Returns the set of changed or new items,
	// and a deltaLink for running the next future Azure query. Implements the pattern described at
	// https://docs.microsoft.com/en-us/graph/delta-query-overview
	k := 1 // Track number of API calls
	r := ApiGet(url, headers, nil)
	ApiErrorCheck(r, utl.Trace())
	for {
		// Infinite for-loop until deltalLink appears (meaning we're done getting current delta set)
		var thisBatch []interface{} = nil // Assume zero entries in this batch
		if r["value"] != nil {
			thisBatch = r["value"].([]interface{})
			if len(thisBatch) > 0 {
				deltaSet = append(deltaSet, thisBatch...) // Continue growing deltaSet
			}
		}
		if verbose {
			// Progress count indicator. Using global var rUp to overwrite last line. Defer newline until done
			fmt.Printf("%s(API calls = %d) %d objects in set %d", rUp, k, len(thisBatch), k)
		}
		if r["@odata.deltaLink"] != nil {
			deltaLinkMap := map[string]string{"@odata.deltaLink": utl.Str(r["@odata.deltaLink"])}
			if verbose {
				fmt.Printf("\n")
			}
			return deltaSet, deltaLinkMap // Return immediately after deltaLink appears
		}
		r = ApiGet(utl.Str(r["@odata.nextLink"]), headers, nil) // Get next batch
		ApiErrorCheck(r, utl.Trace())
		k++
	}
	if verbose {
		fmt.Printf("\n")
	}
	return nil, nil
}

func RemoveCacheFile(t string, z Bundle) {
	switch t {
	case "t":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TokenFile))
	case "d":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions.json"))
	case "a":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments.json"))
	case "s":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_subscriptions.json"))
	case "m":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_managementGroups.json"))
	case "u":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_users.json"))
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_users_deltaLink.json"))
	case "g":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_groups.json"))
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_groups_deltaLink.json"))
	case "sp":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_servicePrincipals.json"))
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_servicePrincipals_deltaLink.json"))
	case "ap":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_applications.json"))
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_applications_deltaLink.json"))
	case "ad":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_directoryRoles.json"))
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_directoryRoles_deltaLink.json"))
	case "all":
		// See https://stackoverflow.com/questions/48072236/remove-files-with-wildcard
		fileList, err := filepath.Glob(filepath.Join(z.ConfDir, z.TenantId+"_*.json"))
		if err != nil {
			panic(err)
		}
		for _, filePath := range fileList {
			utl.RemoveFile(filePath)
		}
	}
	os.Exit(0)
}

func GetObjectFromFile(filePath string) (formatType, t string, obj map[string]interface{}) {
	// Returns 3 values: File format type, single-letter object type, and the object itself

	// Because JSON is essentially a subset of YAML, we have to check JSON first
	// As an interesting aside regarding YAML & JSON, see https://news.ycombinator.com/item?id=31406473
	formatType = "JSON"                     // Pretend it's JSON
	objRaw, _ := utl.LoadFileJson(filePath) // Ignores the errors
	if objRaw == nil {                      // Ok, it's NOT JSON
		objRaw, _ = utl.LoadFileYaml(filePath) // See if it's YAML, ignoring the error
		if objRaw == nil {
			return "", "", nil // Ok, it's neither, let's return 3 null values
		}
		formatType = "YAML" // It is YAML
	}
	obj = objRaw.(map[string]interface{})

	// Continue unpacking the object to see what it is
	xProp, ok := obj["properties"].(map[string]interface{})
	if !ok { // Valid definition/assignments have a properties attribute
		return formatType, "", nil // It's not a valid object, return null for type and object
	}
	roleName := utl.Str(xProp["roleName"])       // Assert and assume it's a definition
	roleId := utl.Str(xProp["roleDefinitionId"]) // assert and assume it's an assignment

	if roleName != "" {
		return formatType, "d", obj // Role definition
	} else if roleId != "" {
		return formatType, "a", obj // Role assignment
	} else {
		return formatType, "", obj // Unknown
	}
}

func CompareSpecfileToAzure(filePath string, z Bundle) {
	if utl.FileNotExist(filePath) || utl.FileSize(filePath) < 1 {
		utl.Die("File does not exist, or is zero size\n")
	}
	formatType, t, x := GetObjectFromFile(filePath)
	if formatType != "JSON" && formatType != "YAML" {
		utl.Die("File is not in JSON nor YAML format\n")
	}
	if t != "d" && t != "a" {
		utl.Die("This " + formatType + " file is not a role definition nor an assignment specfile\n")
	}

	fmt.Printf("==== SPECFILE ============================\n")
	PrintObject(t, x, z) // Use generic print function
	fmt.Printf("==== AZURE ===============================\n")
	if t == "d" {
		y := GetAzRoleDefinition(x, z)
		if y == nil {
			fmt.Printf("Above definition does NOT exist in current Azure tenant\n")
		} else {
			PrintRoleDefinition(y, z) // Use specific role def print function
		}
	} else {
		y := GetAzRoleAssignment(x, z)
		if y == nil {
			fmt.Printf("Above assignment does NOT exist in current Azure tenant\n")
		} else {
			PrintRoleAssignment(y, z) // Use specific role assgmnt print function
		}
	}
	os.Exit(0)
}

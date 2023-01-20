// helper.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"os"
	"path/filepath"
	"time"
)

func UpsertAzObject(filePath string, z Bundle) {
	// Create or Update role definition or assignment based on given specfile
	if utl.FileNotExist(filePath) || utl.FileSize(filePath) < 1 {
		utl.Die("File does not exist, or it is zero size\n")
	}
	formatType, t, x := GetObjectFromFile(filePath)
	if formatType != "JSON" && formatType != "YAML" {
		utl.Die("File is not in JSON nor YAML format\n")
	}
	if t != "d" && t != "a" {
		utl.Die("File is not a role definition nor an assignment specfile\n")
	}
	switch t {
	case "d":
		UpsertAzRoleDefinition(x, z)
	case "a":
		CreateAzRoleAssignment(x, z)
	}
	os.Exit(0)
}

func DeleteAzObject(specifier string, z Bundle) {
	// Delete object based on string specifier (currently only supports roleDefinitions or Assignments)
	// String specifier can be either of 3: UUID, specfile, or displaName (only for roleDefinition)
	// 1) Search Azure by given identifier; 2) Grab object's Fully Qualified Id string;
	// 3) Print and prompt for confirmation; 4) Delete or abort
	if utl.ValidUuid(specifier) {
		list := FindAzObjectsByUuid(specifier, z) // Get all objects that may match this UUID, hopefully just one
		if len(list) > 1 {
			utl.Die(utl.Red("UUID collision? Run utility with UUID argument to see the list.\n"))
		}
		if len(list) < 1 {
			utl.Die("Object does not exist.\n")
		}
		y := list[0].(map[string]interface{}) // Single out the only object
		if y != nil && y["mazType"] != nil {
			t := utl.Str(y["mazType"])
			fqid := utl.Str(y["id"]) // Grab fully qualified object Id
			PrintObject(t, y, z)
			if utl.PromptMsg("DELETE above? y/n ") == 'y' {
				switch t {
				case "d":
					DeleteAzRoleDefinitionByFqid(fqid, z)
				case "a":
					DeleteAzRoleAssignmentByFqid(fqid, z)
				}
			} else {
				utl.Die("Aborted.\n")
			}
		}
	} else if utl.FileExist(specifier) {
		// Delete object defined in specfile
		formatType, t, x := GetObjectFromFile(specifier) // x is for the object in Specfile
		if formatType != "JSON" && formatType != "YAML" {
			utl.Die("File is not in JSON nor YAML format\n")
		}
		var y map[string]interface{} = nil
		switch t {
		case "d":
			y = GetAzRoleDefinitionByObject(x, z) // y is for the object from Azure
			fqid := utl.Str(y["id"])              // Grab fully qualified object Id
			if y == nil {
				utl.Die("Role definition does not exist.\n")
			} else {
				PrintRoleDefinition(y, z) // Use specific role def print function
				if utl.PromptMsg("DELETE above? y/n ") == 'y' {
					DeleteAzRoleDefinitionByFqid(fqid, z)
				} else {
					utl.Die("Aborted.\n")
				}
			}
		case "a":
			y = GetAzRoleAssignmentByObject(x, z)
			fqid := utl.Str(y["id"]) // Grab fully qualified object Id
			if y == nil {
				utl.Die("Role assignment does not exist.\n")
			} else {
				PrintRoleAssignment(y, z) // Use specific role assgmnt print function
				if utl.PromptMsg("DELETE above? y/n ") == 'y' {
					DeleteAzRoleAssignmentByFqid(fqid, z)
				} else {
					utl.Die("Aborted.\n")
				}
			}
		default:
			utl.Die("File " + formatType + " is not a role definition or assignment.\n")
		}
	} else {
		// Delete role definition by its displayName, if it exists. This only applies to definitions
		// since assignments do not have a displayName attribute. Also, other objects are not supported.
		y := GetAzRoleDefinitionByName(specifier, z)
		if y == nil {
			utl.Die("Role definition does not exist.\n")
		}
		fqid := utl.Str(y["id"]) // Grab fully qualified object Id
		PrintRoleDefinition(y, z)
		if utl.PromptMsg("DELETE above? y/n ") == 'y' {
			DeleteAzRoleDefinitionByFqid(fqid, z)
		} else {
			utl.Die("Aborted.\n")
		}
	}
}

func FindAzObjectsByUuid(uuid string, z Bundle) (list []interface{}) {
	// Returns list of Azure objects with this UUID. We are saying a list because potentially
	// this could find UUID collisions. Only checks for the maz limited set of Azure object types.
	list = nil
	mazTypes := []string{"d", "a", "s", "u", "g", "sp", "ap", "ad"}
	for _, t := range mazTypes {
		x := GetAzObjectByUuid(t, uuid, z)
		if x != nil && x["id"] != nil { // Valid objects have an 'id' attribute
			// Note, we are extending the object by adding a mazType as an additional FIELD
			x["mazType"] = t       // Found one of these types with this UUID
			list = append(list, x) // Add it to the list
		}
	}
	return list
}

func GetAzObjectByUuid(t, uuid string, z Bundle) (x map[string]interface{}) {
	// Retrieve Azure object by Object UUID
	switch t {
	case "d":
		return GetAzRoleDefinitionByUuid(uuid, z)
	case "a":
		return GetAzRoleAssignmentByUuid(uuid, z)
	case "s":
		return GetAzSubscriptionByUuid(uuid, z.AzHeaders)
	case "u":
		return GetAzUserByUuid(uuid, z.MgHeaders)
	case "g":
		return GetAzGroupByUuid(uuid, z.MgHeaders)
	case "sp":
		return GetAzSpByUuid(uuid, z.MgHeaders)
	case "ap":
		return GetAzAppByUuid(uuid, z.MgHeaders)
	case "ad":
		return GetAzAdRoleByUuid(uuid, z.MgHeaders)
	}
	return nil
}

func GetAzRbacScopes(z Bundle) (scopes []string) {
	// Get all scopes in the entire Azure RBAC hierarchy: All MG scopes + All subscription scopes
	scopes = nil
	managementGroups := GetAzMgGroups(z) // Start by adding all the managementGroups scopes
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
		// SCOPES below subscriptions do not appear to be REALLY NEEDED
		// Most list search functions pull all objects in lower scopes.
		// ------------
		// // Now get/add all resourceGroups under this subscription
		// params := map[string]string{"api-version": "2021-04-01"} // resourceGroups
		// url := ConstAzUrl + utl.Str(x["id"]) + "/resourcegroups"
		// r, _, _ := ApiGet(url, z.AzHeaders, params)
		// ApiErrorCheck("GET", url, utl.Trace(), r)
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
	r, _, _ := ApiGet(url, headers, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
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
		r, _, _ = ApiGet(utl.Str(r["@odata.nextLink"]), headers, nil) // Get next batch
		ApiErrorCheck("GET", url, utl.Trace(), r)
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
		utl.Die("File " + formatType + " is not a role definition or assignment.\n")
	}

	fmt.Printf("==== SPECFILE ============================\n")
	PrintObject(t, x, z) // Use generic print function
	fmt.Printf("==== AZURE ===============================\n")
	if t == "d" {
		y := GetAzRoleDefinitionByObject(x, z)
		if y == nil {
			fmt.Printf("Role definition does not exist.\n")
		} else {
			PrintRoleDefinition(y, z) // Use specific role def print function
		}
	} else {
		y := GetAzRoleAssignmentByObject(x, z)
		if y == nil {
			fmt.Printf("Role assignment does not exist.\n")
		} else {
			PrintRoleAssignment(y, z) // Use specific role assgmnt print function
		}
	}
	os.Exit(0)
}

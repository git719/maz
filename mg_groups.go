// mg_groups.go
// MS Graph groups

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
)

func PrintGroup(x map[string]interface{}, z Bundle) {
	// Print group object in YAML-like format
	if x == nil {
		return
	}
	id := utl.Str(x["id"])

	// Print the primary keys first
	keys := []string{"id", "displayName", "description", "isAssignableToRole"}
	for _, i := range keys {
		if v := utl.Str(x[i]); v != "" { // Print only non-empty keys
			fmt.Printf("%s: %s\n", utl.Blu(i), utl.Gre(v))
		}
	}

	// Print owners of this group
	url := ConstMgUrl + "/v1.0/groups/" + id + "/owners"
	r, statusCode, _ := ApiGet(url, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		owners := r["value"].([]interface{}) // Assert as JSON array type
		if len(owners) > 0 {
			fmt.Printf(utl.Blu("owners") + ":\n")
			for _, i := range owners {
				o := i.(map[string]interface{}) // Assert as JSON object type
				fmt.Printf("  %-50s %s\n", utl.Gre(utl.Str(o["userPrincipalName"])), utl.Gre(utl.Str(o["id"])))
			}
		}
	}

	// Print all groups and roles it is a member of
	url = ConstMgUrl + "/v1.0/groups/" + id + "/transitiveMemberOf"
	r, statusCode, _ = ApiGet(url, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}

	// Print members of this group
	//url = ConstMgUrl + "/v1.0/groups/" + id + "/members"  // Get nothing with this, so evidently still in beta
	url = ConstMgUrl + "/beta/groups/" + id + "/members" // beta works
	r, statusCode, _ = ApiGet(url, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		members := r["value"].([]interface{})
		if len(members) > 0 {
			fmt.Printf(utl.Blu("members") + ":\n")
			for _, i := range members {
				m := i.(map[string]interface{}) // Assert as JSON object type
				Type, Name := "-", "-"
				Type = utl.LastElem(utl.Str(m["@odata.type"]), ".")
				switch Type {
				case "group", "servicePrincipal":
					Name = utl.Str(m["displayName"])
				default:
					Name = utl.Str(m["userPrincipalName"])
				}
				fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name), utl.Gre(utl.Str(m["id"])), utl.Gre(Type))
			}
		}
	}
}

func GroupsCountLocal(z Bundle) int64 {
	// Return number of entries in local cache file
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_groups."+ConstCacheFileExtension)
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJsonGzip(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func GroupsCountAzure(z Bundle) int64 {
	// Return number of entries in Azure tenant
	z.MgHeaders["ConsistencyLevel"] = "eventual"
	url := ConstMgUrl + "/v1.0/groups/$count"
	r, _, _ := ApiGet(url, z, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] != nil {
		return r["value"].(int64) // Expected result is a single int64 value for the count
	}
	return 0
}

func GetIdMapGroups(z Bundle) (nameMap map[string]string) {
	// Return groups id:name map
	nameMap = make(map[string]string)
	groups := GetGroups("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range groups {
		x := i.(map[string]interface{})
		if x["id"] != nil && x["displayName"] != nil {
			nameMap[utl.Str(x["id"])] = utl.Str(x["displayName"])
		}
	}
	return nameMap
}

func GetGroups(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all groups matching on 'filter'; return entire list if filter is empty ""

	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_groups."+ConstCacheFileExtension)
	cacheFileAge := utl.FileAge(cacheFile)
	if utl.InternetIsAvailable() && (force || cacheFileAge == 0 || cacheFileAge > ConstMgCacheFileAgePeriod) {
		// If Internet is available AND (force was requested OR cacheFileAge is zero (meaning does not exist)
		// OR it is older than ConstMgCacheFileAgePeriod) then query Azure directly to get all objects
		// and show progress while doing so (true = verbose below)
		list = GetAzGroups(z, true)
	} else {
		// Use local cache for all other conditions
		list = GetCachedObjects(cacheFile)
	}

	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	searchKeys := []string{"id", "displayName", "description"}
	var ids []string // Keep track of each unique objects to eliminate repeats
	for _, i := range list {
		x := i.(map[string]interface{})
		id := utl.Str(x["id"])
		for _, i := range searchKeys {
			if utl.SubString(utl.Str(x[i]), filter) && !utl.ItemInList(id, ids) {
				matchingList = append(matchingList, x)
				ids = append(ids, id)
			}
		}
	}
	return matchingList
}

func GetAzGroups(z Bundle, verbose bool) (list []interface{}) {
	// Get all groups from Azure and sync to local cache; show progress if verbose = true

	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_groups."+ConstCacheFileExtension)
	deltaLinkFile := filepath.Join(z.ConfDir, z.TenantId+"_groups_deltaLink."+ConstCacheFileExtension)

	baseUrl := ConstMgUrl + "/beta/groups"
	// Get delta updates only if/when selection attributes are modified
	selection := "?$select=displayName,description,isAssignableToRole"
	url := baseUrl + "/delta" + selection + "&$top=999"
	list = GetCachedObjects(cacheFile) // Get current cache
	if len(list) < 1 {
		// These are only needed on initial cache run
		z.MgHeaders["Prefer"] = "return=minimal" // Tells API to focus only on $select attributes deltas
		z.MgHeaders["deltaToken"] = "latest"
		// https://graph.microsoft.com/v1.0/users/delta?$deltatoken=latest
	}

	// Prep to do a delta query if it is possible
	var deltaLinkMap map[string]interface{} = nil
	if utl.FileUsable(deltaLinkFile) && utl.FileAge(deltaLinkFile) < (3660*24*27) && len(list) > 0 {
		// Note that deltaLink file age has to be within 30 days (we do 27)
		tmpVal, _ := utl.LoadFileJsonGzip(deltaLinkFile)
		deltaLinkMap = tmpVal.(map[string]interface{})
		url = utl.Str(utl.Str(deltaLinkMap["@odata.deltaLink"]))
		// Base URL is now the cached Delta Link URL
	}

	// Now go get Azure objects using the updated URL (either a full or a delta query)
	var deltaSet []interface{} = nil
	deltaSet, deltaLinkMap = GetAzObjects(url, z, verbose) // Run generic deltaSet retriever function

	// Save new deltaLink for future call, and merge newly acquired delta set with existing list
	utl.SaveFileJsonGzip(deltaLinkMap, deltaLinkFile)
	list = NormalizeCache(list, deltaSet) // Run our MERGE LOGIC with new delta set
	utl.SaveFileJsonGzip(list, cacheFile) // Update the local cache
	return list
}

func GetAzGroupByUuid(uuid string, z Bundle) map[string]interface{} {
	// Get Azure AD group by Object UUID, with all attributes
	baseUrl := ConstMgUrl + "/beta/groups"
	selection := "?$select=*"
	url := baseUrl + "/" + uuid + selection
	r, _, _ := ApiGet(url, z, nil)
	return r
}

func PrintPags(z Bundle) {
	// List all cached Privileged Access Groups
	groups := GetGroups("", false, z) // Get all groups, false = don't hit Azure
	for _, i := range groups {
		x := i.(map[string]interface{})
		if x["isAssignableToRole"] != nil {
			if x["isAssignableToRole"].(bool) {
				PrintTersely("g", x) // Pring group tersely
			}
		}
	}
}
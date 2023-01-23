// groups.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
	"time"
)

func PrintGroup(x map[string]interface{}, z Bundle) {
	// Print group object in YAML-like format
	if x == nil {
		return
	}
	id := utl.Str(x["id"])

	// First, print the most important attributes of this group
	co := utl.Red(":") // Colorize ":" text to Red
	list := []string{"id", "displayName", "description", "isAssignableRole", "isAssignableToRole", "mailEnabled", "mailNickname"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s %s\n", utl.Cya(i)+co, v)
		}
	}

	// Print owners of this group
	url := ConstMgUrl + "/beta/groups/" + id + "/owners"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		owners := r["value"].([]interface{}) // Assert as JSON array type
		if len(owners) > 0 {
			fmt.Printf(utl.Cya("owners") + co + "\n")
			for _, i := range owners {
				o := i.(map[string]interface{}) // Assert as JSON object type
				fmt.Printf("  %-50s %s\n", utl.Str(o["userPrincipalName"]), utl.Str(o["id"]))
			}
		}
	}

	// Print all groups and roles it is a member of
	url = ConstMgUrl + "/v1.0/groups/" + id + "/transitiveMemberOf"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}

	// Print members of this group
	url = ConstMgUrl + "/beta/groups/" + id + "/members"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		members := r["value"].([]interface{})
		if len(members) > 0 {
			fmt.Printf(utl.Cya("members") + co + "\n")
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
				fmt.Printf("  %-50s %s (%s)\n", Name, utl.Str(m["id"]), Type)
			}
		}
	}
}

func GroupsCountLocal(z Bundle) int64 {
	// Return number of entries in local cache file
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_groups.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
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
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
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
	// Get all Azure AD groups whose searchAttributes match on 'filter'. An empty "" filter returns all.
	// Uses local cache if it's less than cachePeriod old. The 'force' option forces calling Azure query.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_groups.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if cacheNoGood || force {
		list = GetAzGroups(cacheFile, z.MgHeaders, true) // Get all from Azure and show progress (verbose = true)
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	searchAttributes := []string{
		"id", "displayName", "userPrincipalName", "onPremisesSamAccountName",
		"onPremisesUserPrincipalName", "onPremisesDomainName",
	}
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

func GetAzGroups(cacheFile string, headers map[string]string, verbose bool) (list []interface{}) {
	// Get all Azure AD groups in current tenant AND save them to local cache file. Show progress if verbose = true.

	// We will first try doing a delta query. See https://docs.microsoft.com/en-us/graph/delta-query-overview
	var deltaLinkMap map[string]string = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))
	baseUrl := ConstMgUrl + "/v1.0/groups"
	// Get delta updates only when below selection of attributes are modified
	selection := "?$select=id,displayName"
	url := baseUrl + "/delta" + selection + "&$top=999"
	headers["Prefer"] = "return=minimal" // This tells API to focus only on specific 'select' attributes

	// But first, double-check the base set again to avoid running a delta query on an empty set
	listIsEmpty, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if utl.FileUsable(deltaLinkFile) && deltaAge < (3660*24*27) && listIsEmpty == false {
		// Note that deltaLink file age has to be within 30 days (we do 27)
		tmpVal, _ := utl.LoadFileJson(deltaLinkFile)
		deltaLinkMap = tmpVal.(map[string]string)
		url = utl.Str(deltaLinkMap["@odata.deltaLink"]) // Base URL is now the cached Delta Link
	}

	// Now go get azure objects using the updated URL (either a full query or a deltaLink query)
	var deltaSet []interface{} = nil
	deltaSet, deltaLinkMap = GetAzObjects(url, headers, verbose) // Run generic deltaSet retriever function

	// Save new deltaLink for future call, and merge newly acquired delta set with existing list
	utl.SaveFileJson(deltaLinkMap, deltaLinkFile)
	list = NormalizeCache(list, deltaSet) // Run our MERGE LOGIC with new delta set
	utl.SaveFileJson(list, cacheFile)     // Update the local cache
	return list
}

func GetAzGroupByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure AD group by Object UUID, with extended attributes
	baseUrl := ConstMgUrl + "/v1.0/groups"
	selection := "?$select=id,createdDateTime,description,displayName,groupTypes,id,isAssignableToRole,"
	selection += "mail,mailNickname,onPremisesLastSyncDateTime,onPremisesProvisioningErrors,"
	selection += "onPremisesSecurityIdentifier,onPremisesSyncEnabled,renewedDateTime,securityEnabled,"
	selection += "securityIdentifier,memberOf,members,owners"
	url := baseUrl + "/" + uuid + selection
	r, _, _ := ApiGet(url, headers, nil)
	//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
	return r
}

func PrintPags(z Bundle) {
	// List all Privileged Access Groups
	groups := GetGroups("", false, z) // Get all groups, false = not need to hit Azure
	for _, i := range groups {
		x := i.(map[string]interface{})
		if x["isAssignableToRole"] != nil {
			if x["isAssignableToRole"].(bool) {
				PrintTersely("g", x) // Pring group tersely
			}
		}
	}
}

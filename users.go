// users.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
	"time"
)

func PrintUser(x map[string]interface{}, z Bundle) {
	// Print user object in YAML-like format
	if x == nil {
		return
	}
	id := utl.Str(x["id"])

	// Print the primary keys first
	keys := []string{"id", "displayName", "userPrincipalName", "onPremisesSamAccountName", "onPremisesDomainName"}
	for _, i := range keys {
		if v := utl.Str(x[i]); v != "" { // Print only non-empty keys
			fmt.Printf("%s: %s\n", utl.Blu(i), utl.Gre(v))
		}
	}

	// Print all groups and roles it is a member of
	url := ConstMgUrl + "/v1.0/users/" + id + "/transitiveMemberOf"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}
}

func UsersCountLocal(z Bundle) int64 {
	// Return number of entries in local cache file
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_users.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func UsersCountAzure(z Bundle) int64 {
	// Return number of entries in Azure tenant
	z.MgHeaders["ConsistencyLevel"] = "eventual"
	url := ConstMgUrl + "/v1.0/users/$count"
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] != nil {
		return r["value"].(int64) // Expected result is a single int64 value for the count
	}
	return 0
}

func GetIdMapUsers(z Bundle) (nameMap map[string]string) {
	// Return users id:name map
	nameMap = make(map[string]string)
	users := GetUsers("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range users {
		x := i.(map[string]interface{})
		if x["id"] != nil && x["displayName"] != nil {
			nameMap[utl.Str(x["id"])] = utl.Str(x["displayName"])
		}
	}
	return nameMap
}

func GetUsers(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all Azure AD users that match on 'filter'. An empty "" filter returns all.
	// Uses local cache if it's less than cachePeriod old. The 'force' option forces calling Azure query.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_users.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if cacheNoGood || force {
		list = GetAzUsers(cacheFile, z, true) // Get all from Azure and show progress (verbose = true)
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	searchKeys := []string{
		"id", "displayName", "userPrincipalName", "onPremisesSamAccountName",
	}
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

func GetAzUsers(cacheFile string, z Bundle, verbose bool) (list []interface{}) {
	// Get all Azure AD users in current tenant AND save them to local cache file. Show progress if verbose = true.

	// We will first try doing a delta query. See https://docs.microsoft.com/en-us/graph/delta-query-overview
	var deltaLinkMap map[string]interface{} = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))

	baseUrl := ConstMgUrl + "/v1.0/users"
	// Get delta updates only if/when below attributes in $select are modified
	selection := "?$select=displayName,userPrincipalName,onPremisesSamAccountName,"
	url := baseUrl + "/delta" + selection + "&$top=999"
	headers := z.MgHeaders
	headers["Prefer"] = "return=minimal" // Tells API to focus only on $select attributes
	headers["deltaToken"] = "latest"

	// But first, double-check the base set again to avoid running a delta query on an empty set
	listIsEmpty, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if utl.FileUsable(deltaLinkFile) && deltaAge < (3660*24*27) && listIsEmpty == false {
		// Note that deltaLink file age has to be within 30 days (we use 27)
		tmpVal, _ := utl.LoadFileJson(deltaLinkFile)
		deltaLinkMap = tmpVal.(map[string]interface{})
		url = utl.Str(utl.Str(deltaLinkMap["@odata.deltaLink"]))
		// Base URL is now the cached Delta Link URL
	}

	// Now go get Azure objects using the updated URL (either a full or a deltaLink query)
	var deltaSet []interface{} = nil
	deltaSet, deltaLinkMap = GetAzObjects(url, z, verbose) // Run generic deltaSet retriever function

	// Save new deltaLink for future call, and merge newly acquired delta set with existing list
	utl.SaveFileJson(deltaLinkMap, deltaLinkFile)
	list = NormalizeCache(list, deltaSet) // Run our MERGE LOGIC with new delta set
	utl.SaveFileJson(list, cacheFile)     // Update the local cache
	return list
}

func GetAzUserByUuid(uuid string, z Bundle) map[string]interface{} {
	// Get Azure user by Object UUID, with extended attributes
	baseUrl := ConstMgUrl + "/v1.0/users"
	selection := "?$select=id,displayName,userPrincipalName,mailNickname,onPremisesSamAccountName," +
		"onPremisesDomainName,onPremisesUserPrincipalName,otherMails,identities,accountEnabled," +
		"createdDateTime,creationType,lastPasswordChangeDateTime,mail,onPremisesDistinguishedName," +
		"onPremisesExtensionAttributes,onPremisesImmutableId,onPremisesLastSyncDateTime," +
		"onPremisesProvisioningErrors,onPremisesSecurityIdentifier,onPremisesSyncEnabled," +
		"securityIdentifier,surname,tags,"
	url := baseUrl + "/" + uuid + selection
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
	//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
	return r
}

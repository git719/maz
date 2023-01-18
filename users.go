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

	// First, print the most important attributes for this user
	list := []string{"displayName", "id", "userPrincipalName", "mailNickname", "onPremisesSamAccountName",
		"onPremisesDomainName", "onPremisesUserPrincipalName"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s: %s\n", i, v)
		}
	}

	// Print other mails this user has configured
	if x["otherMails"] != nil {
		otherMails := x["otherMails"].([]interface{})
		if len(otherMails) > 0 {
			fmt.Printf("otherMails:\n")
			for _, i := range otherMails {
				email := i.(string)
				fmt.Printf("  %s\n", email)
			}
		} else {
			fmt.Printf("  %s: %s\n", "otherMails", "None")
		}
	}

	// Print all groups and roles it is a member of
	url := ConstMgUrl + "/v1.0/users/" + id + "/transitiveMemberOf"
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r != nil && r["value"] != nil {
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
	cacheNoGood, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if cacheNoGood || force {
		list = GetAzUsers(cacheFile, z.MgHeaders, true) // Get all from Azure and show progress (verbose = true)
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

func GetAzUsers(cacheFile string, headers map[string]string, verbose bool) (list []interface{}) {
	// Get all Azure AD users in current tenant AND save them to local cache file. Show progress if verbose = true.

	// We will first try doing a delta query. See https://docs.microsoft.com/en-us/graph/delta-query-overview
	var deltaLinkMap map[string]string = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))

	baseUrl := ConstMgUrl + "/v1.0/users"
	// Get delta updates only when below selection of attributes are modified
	selection := "?$select=displayName,mailNickname,userPrincipalName,onPremisesSamAccountName,"
	selection += "onPremisesDomainName,onPremisesUserPrincipalName"
	url := baseUrl + "/delta" + selection + "&$top=999"
	headers["Prefer"] = "return=minimal" // This tells API to focus only on specific 'select' attributes

	// But first, double-check the base set again to avoid running a delta query on an empty set
	listIsEmpty, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if utl.FileUsable(deltaLinkFile) && deltaAge < (3660*24*27) && listIsEmpty == false {
		// Note that deltaLink file age has to be within 30 days (we do 27)
		tmpVal, _ := utl.LoadFileJson(deltaLinkFile)
		deltaLinkMap = tmpVal.(map[string]string)
		url = utl.Str(deltaLinkMap["@odata.deltaLink"]) // Base URL is now the cached Delta Link URL
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

func GetAzUserByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure user by Object UUID, with extended attributes
	baseUrl := ConstMgUrl + "/v1.0/users"
	selection := "?$select=id,accountEnabled,createdDateTime,creationType,displayName,id,identities,"
	selection += "lastPasswordChangeDateTime,mail,mailNickname,onPremisesDistinguishedName,"
	selection += "onPremisesDomainName,onPremisesExtensionAttributes,onPremisesImmutableId,"
	selection += "onPremisesLastSyncDateTime,onPremisesProvisioningErrors,onPremisesSamAccountName,"
	selection += "onPremisesSecurityIdentifier,onPremisesSyncEnabled,onPremisesUserPrincipalName,"
	selection += "otherMails,securityIdentifier,surname,userPrincipalName"
	url := baseUrl + "/" + uuid + selection
	r, _, _ := ApiGet(url, headers, nil)
	//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
	return r
}

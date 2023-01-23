// sps.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
	"strings"
	"time"
)

func PrintSp(x map[string]interface{}, z Bundle) {
	// Print service principal object in YAML-like format
	if x == nil {
		return
	}
	id := utl.Str(x["id"])

	// Print the most important attributes
	co := utl.Red(":") // Colorize ":" text to Red
	list := []string{"id", "displayName", "appId", "accountEnabled", "servicePrincipalType", "appOwnerOrganizationId"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s %s\n", utl.Cya(i)+co, v)
		}
	}

	// Print certificates keys
	if x["keyCredentials"] != nil {
		PrintCertificateList(x["keyCredentials"].([]interface{}))
	}

	// Print secret expiry and other details. Not actual secretText, which cannot be retrieve anyway!
	if x["passwordCredentials"] != nil {
		PrintSecretList(x["passwordCredentials"].([]interface{}))
	}

	// Print owners
	url := ConstMgUrl + "/beta/servicePrincipals/" + id + "/owners"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		PrintOwners(r["value"].([]interface{}))
	}

	// Print members and their roles
	url = ConstMgUrl + "/beta/servicePrincipals/" + id + "/appRoleAssignedTo"
	r, _, _ = ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] != nil {
		members := r["value"].([]interface{}) // JSON array
		if len(members) > 0 {
			fmt.Printf(utl.Cya("members") + co + "\n")

			// Build roleMap
			roleMap := make(map[string]string)
			if x["appRoles"] != nil {
				objAppRoles := x["appRoles"].([]interface{})
				if len(objAppRoles) > 0 {
					for _, i := range objAppRoles {
						ar := i.(map[string]interface{})
						roleMap[utl.Str(ar["id"])] = utl.Str(ar["displayName"])
					}
				}
			}
			// Add Default Access role
			roleMap["00000000-0000-0000-0000-000000000000"] = "Default Access"

			for _, i := range members {
				rm := i.(map[string]interface{}) // JSON object
				principalName := utl.Str(rm["principalDisplayName"])
				roleName := roleMap[utl.Str(rm["appRoleId"])] // Reference role name
				principalId := utl.Str(rm["principalId"])
				principalType := utl.Str(rm["principalType"])
				fmt.Printf("  %-50s %-20s %s (%s)\n", principalName, roleName, principalId, principalType)
			}
		}
	}

	// Print all groups and roles it is a member of
	url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/transitiveMemberOf"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}

	// Print API permissions
	url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/oauth2PermissionGrants"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		fmt.Printf(utl.Cya("api_permissionsi") + co + "\n")
		apiPerms := r["value"].([]interface{}) // Assert as JSON array

		// Print OAuth 2.0 scopes for each API
		for _, i := range apiPerms {
			api := i.(map[string]interface{}) // Assert as JSON object
			apiName := "Unknown"
			id := utl.Str(api["resourceId"]) // Get API's SP to get its displayName
			url2 := ConstMgUrl + "/v1.0/servicePrincipals/" + id
			r2, _, _ := ApiGet(url2, z.MgHeaders, nil)
			if r2["appDisplayName"] != nil {
				apiName = utl.Str(r2["appDisplayName"])
			}
			ApiErrorCheck("GET", url2, utl.Trace(), r2)

			// Print each delegated claim for this API
			scope := strings.TrimSpace(utl.Str(api["scope"]))
			claims := strings.Split(scope, " ")
			for _, j := range claims {
				fmt.Printf("  %-50s %s\n", apiName, j)
			}
		}
	}
}

func SpsCountLocal(z Bundle) (native, microsoft int64) {
	// Retrieves counts of all SPs in local cache, 2 values: Native ones to this tenant, and all others
	var nativeList []interface{} = nil
	var microsoftList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_servicePrincipals.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList := rawList.([]interface{})
			for _, i := range cachedList {
				x := i.(map[string]interface{})
				if utl.Str(x["appOwnerOrganizationId"]) == z.TenantId { // If owned by current tenant ...
					nativeList = append(nativeList, x)
				} else {
					microsoftList = append(microsoftList, x)
				}
			}
			return int64(len(nativeList)), int64(len(microsoftList))
		}
	}
	return 0, 0
}

func SpsCountAzure(z Bundle) (native, microsoft int64) {
	// Retrieves counts of all SPs in this Azure tenant, 2 values: Native ones to this tenant, and all others

	// First, get total number of SPs in tenant
	var all int64 = 0
	z.MgHeaders["ConsistencyLevel"] = "eventual"
	baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	url := baseUrl + "/$count"
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] == nil {
		return 0, 0 // Something went wrong, so return zero for both
	}
	all = r["value"].(int64)

	// Now get count of SPs registered and native to only this tenant
	params := map[string]string{"$filter": "appOwnerOrganizationId eq " + z.TenantId}
	params["$count"] = "true"
	url = baseUrl
	r, _, _ = ApiGet(url, z.MgHeaders, params)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] == nil {
		return 0, all // Something went wrong with native count, retun all as Microsoft ones
	}

	native = int64(r["@odata.count"].(float64))
	microsoft = all - native

	return native, microsoft
}

func GetIdMapSps(z Bundle) (nameMap map[string]string) {
	// Return service principals id:name map
	nameMap = make(map[string]string)
	sps := GetSps("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range sps {
		x := i.(map[string]interface{})
		if x["id"] != nil && x["displayName"] != nil {
			nameMap[utl.Str(x["id"])] = utl.Str(x["displayName"])
		}
	}
	return nameMap
}

func GetSps(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all Azure AD service principal whose searchAttributes match on 'filter'. An empty "" filter returns all.
	// Uses local cache if it's less than cachePeriod old. The 'force' option forces calling Azure query.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_servicePrincipals.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if cacheNoGood || force {
		list = GetAzSps(cacheFile, z.MgHeaders, true) // Get all from Azure and show progress (verbose = true)
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	searchAttributes := []string{"id", "displayName", "appId"}
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

func GetAzSps(cacheFile string, headers map[string]string, verbose bool) (list []interface{}) {
	// Get all Azure AD service principal in current tenant AND save them to local cache file. Show progress if verbose = true.

	// We will first try doing a delta query. See https://docs.microsoft.com/en-us/graph/delta-query-overview
	var deltaLinkMap map[string]string = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))
	baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	// Get delta updates only when below selection of attributes are modified
	selection := "?$id,select=id,displayName"
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

func GetAzSpByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure AD service principal by its Object UUID or by its appId, with extended attributes
	baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	selection := "?$select=id,displayName,appId,accountEnabled,servicePrincipalType,appOwnerOrganizationId,"
	selection += "appRoleAssignmentRequired,appRoles,disabledByMicrosoftStatus,addIns,alternativeNames,"
	selection += "appDisplayName,homepage,id,info,logoutUrl,notes,oauth2PermissionScopes,replyUrls,"
	selection += "resourceSpecificApplicationPermissions,servicePrincipalNames,tags"
	url := baseUrl + "/" + uuid + selection // First search is for direct Object Id
	r, _, _ := ApiGet(url, headers, nil)
	if r != nil && r["error"] != nil {
		// Second search is for this SP's application Client Id
		url = baseUrl + selection
		params := map[string]string{"$filter": "appId eq '" + uuid + "'"}
		r, _, _ := ApiGet(url, headers, params)
		//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
		if r != nil && r["value"] != nil {
			list := r["value"].([]interface{})
			count := len(list)
			if count == 1 {
				return list[0].(map[string]interface{}) // Return single value found
			} else if count > 1 {
				// Not sure this would ever happen, but just in case
				fmt.Printf("Found %d entries with this appId\n", count)
				return nil
			} else {
				return nil
			}
		}
	}
	return r
}

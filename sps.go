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
	list := []string{"id", "displayName", "appId", "accountEnabled", "servicePrincipalType",
		"appOwnerOrganizationId"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s: %s\n", utl.Blu(i), utl.Gre(v))
		}
	}

	// Print certificates keys
	url := ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/keyCredentials"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		keyCredentials := r["value"].([]interface{}) // Assert as JSON array
		if keyCredentials != nil {
			PrintCertificateList(keyCredentials)
		}
	}

	// Print secret expiry and other details. Not actual secretText, which cannot be retrieve anyway!
	url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/passwordCredentials"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		passwordCredentials := r["value"].([]interface{}) // Assert as JSON array
		if passwordCredentials != nil {
			PrintSecretList(passwordCredentials)
		}
	}

	// Print owners
	url = ConstMgUrl + "/beta/servicePrincipals/" + id + "/owners"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		PrintOwners(r["value"].([]interface{}))
	}

	// Print App Roles this resource SP allows
	roleNameMap := make(map[string]string)                                 // Used later when printing members and their roles
	roleNameMap["00000000-0000-0000-0000-000000000000"] = "Default Access" // Add Default Access role
	appRoles := x["appRoles"].([]interface{})
	if len(appRoles) > 0 {
		fmt.Printf(utl.Blu("app_roles") + ":\n")
		for _, i := range appRoles {
			a := i.(map[string]interface{})
			rId := utl.Str(a["id"])
			displayName := utl.Str(a["displayName"])
			roleNameMap[rId] = displayName
			if len(displayName) >= 60 {
				displayName = utl.FirstN(displayName, 57) + "..."
			}
			fmt.Printf("  %s  %-50s  %-60s\n", utl.Gre(rId), utl.Gre(utl.Str(a["value"])), utl.Gre(displayName))
		}
	}

	// Print app role assignment members and the specific role assigned
	//url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/appRoleAssignedTo"
	url = ConstMgUrl + "/beta/servicePrincipals/" + id + "/appRoleAssignedTo"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		appRoleAssignments := r["value"].([]interface{}) // Assert as JSON array
		if len(appRoleAssignments) > 0 {
			fmt.Printf(utl.Blu("appRoleAssignments") + ":\n")
			for _, i := range appRoleAssignments {
				ara := i.(map[string]interface{}) // JSON object
				principalId := utl.Str(ara["principalId"])
				principalType := utl.Str(ara["principalType"])
				principalName := utl.Str(ara["principalDisplayName"])
				roleName := roleNameMap[utl.Str(ara["appRoleId"])] // Reference roleNameMap now
				if len(roleName) >= 40 {
					roleName = utl.FirstN(roleName, 37) + "..."
				}
				principalName = utl.Gre(principalName)
				roleName = utl.Gre(roleName)
				principalId = utl.Gre(principalId)
				principalType = utl.Gre(principalType)
				fmt.Printf("  %-50s %-40s %s (%s)\n", principalName, roleName, principalId, principalType)
			}
		}
	}

	// Print all groups and roles it is a member of
	//url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/transitiveMemberOf"
	url = ConstMgUrl + "/beta/servicePrincipals/" + id + "/transitiveMemberOf"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}

	// Print API permissions
	url = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/oauth2PermissionGrants"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		fmt.Printf(utl.Blu("api_permissions") + ":\n")
		apiPerms := r["value"].([]interface{}) // Assert as JSON array

		// Print OAuth 2.0 scopes for each API
		for _, i := range apiPerms {
			api := i.(map[string]interface{}) // Assert as JSON object
			apiName := "Unknown"
			id := utl.Str(api["resourceId"]) // Get API's SP to get its displayName
			//url2 := ConstMgUrl + "/v1.0/servicePrincipals/" + id
			url2 := ConstMgUrl + "/beta/servicePrincipals/" + id
			r2, _, _ := ApiGet(url2, z.MgHeaders, nil)
			ApiErrorCheck("GET", url2, utl.Trace(), r2)
			if r2["appDisplayName"] != nil {
				apiName = utl.Str(r2["appDisplayName"])
			}

			// Print each delegated claim for this API
			scope := strings.TrimSpace(utl.Str(api["scope"]))
			claims := strings.Split(scope, " ")
			for _, j := range claims {
				fmt.Printf("  %-50s %s\n", utl.Gre(apiName), utl.Gre(j))
			}
		}
	}
}

func AddSpSecret(uuid, displayName, expiry string, z Bundle) {
	if !utl.ValidUuid(uuid) {
		utl.Die("Invalid SP UUID.\n")
	}
	var endDateTime string
	if utl.ValidDate(expiry, "2006-01-02") {
		var err error
		endDateTime, err = utl.ConvertDateFormat(expiry, "2006-01-02", time.RFC3339Nano)
		if err != nil {
			utl.Die("Error converting Expiry date format to RFC3339Nano/ISO8601 format.\n")
		}
	} else {
		// If expiry not a valid date, see if it's a valid integer number
		days, err := utl.StringToInt64(expiry)
		if err != nil {
			utl.Die("Error converting Expiry to valid integer number.\n")
		}
		maxDays := utl.GetDaysSinceOrTo("9999-12-31") // Maximum supported date
		if days > maxDays {
			days = maxDays
		}
		expiryTime := utl.GetDateInDays(utl.Int64ToString(days)) // Set expiryTime to 'days' from now
		expiry = expiryTime.Format("2006-01-02")                 // Convert it to yyyy-mm-dd format
		endDateTime = expiryTime.Format(time.RFC3339Nano)        // Convert to RFC3339Nano/ISO8601 format
	}

	payload := map[string]interface{}{
		"passwordCredential": map[string]string{
			"displayName": displayName,
			"endDateTime": endDateTime,
		},
	}
	url := ConstMgUrl + "/v1.0/servicePrincipals/" + uuid + "/addPassword"
	r, statusCode, _ := ApiPost(url, payload, z.MgHeaders, nil)
	if statusCode == 200 {
		fmt.Printf("%s: %s\n", utl.Blu("App_Object_Id"), utl.Gre(uuid))
		fmt.Printf("%s: %s\n", utl.Blu("New_Secret_Id"), utl.Gre(utl.Str(r["keyId"])))
		fmt.Printf("%s: %s\n", utl.Blu("New_Secret_Name"), utl.Gre(displayName))
		fmt.Printf("%s: %s\n", utl.Blu("New_Secret_Expiry"), utl.Gre(expiry))
		fmt.Printf("%s: %s\n", utl.Blu("New_Secret_Text"), utl.Gre(utl.Str(r["secretText"])))
	} else {
		e := r["error"].(map[string]interface{})
		utl.Die(e["message"].(string) + "\n")
	}
}

func RemoveSpSecret(uuid, keyId string, z Bundle) {
	if !utl.ValidUuid(uuid) {
		utl.Die("SP UUID is not a valid UUID.\n")
	}
	if !utl.ValidUuid(keyId) {
		utl.Die("Secret ID is not a valid UUID.\n")
	}

	// Get SP, display details and secret, and prompt for delete confirmation
	x := GetAzSpByUuid(uuid, z.MgHeaders)
	if x == nil || x["id"] == nil {
		utl.Die("There's no SP with this UUID.\n")
	}
	url := ConstMgUrl + "/v1.0/servicePrincipals/" + uuid + "/passwordCredentials"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	var passwordCredentials []interface{} = nil
	if statusCode == 200 && r != nil && r["value"] != nil && len(r["value"].([]interface{})) > 0 {
		passwordCredentials = r["value"].([]interface{}) // Assert as JSON array
	}
	if passwordCredentials == nil || len(passwordCredentials) < 1 {
		utl.Die("SP object has no secrets.\n")
	}
	var a map[string]interface{} = nil // Target keyId, Secret ID to be deleted
	for _, i := range passwordCredentials {
		targetKeyId := i.(map[string]interface{})
		if utl.Str(targetKeyId["keyId"]) == keyId {
			a = targetKeyId
			break
		}
	}
	if a == nil {
		utl.Die("SP object does not have this Secret ID.\n")
	}
	cId := utl.Str(a["keyId"])
	cName := utl.Str(a["displayName"])
	cHint := utl.Str(a["hint"]) + "********"
	cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die(utl.Trace() + err.Error() + "\n")
	}
	cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die(utl.Trace() + err.Error() + "\n")
	}

	// Prompt
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Str(x["id"]))
	fmt.Printf("%s: %s\n", utl.Blu("appId"), utl.Str(x["appId"]))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Str(x["displayName"]))
	fmt.Printf("%s:\n", utl.Blu("secret_to_be_deleted"))
	fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", utl.Gre(cId), utl.Gre(cName),
		utl.Gre(cHint), utl.Gre(cStart), utl.Gre(cExpiry))
	if utl.PromptMsg("DELETE above? y/n ") == 'y' {
		payload := map[string]interface{}{"keyId": keyId}
		url := ConstMgUrl + "/v1.0/servicePrincipals/" + uuid + "/removePassword"
		r, statusCode, _ := ApiPost(url, payload, z.MgHeaders, nil)
		if statusCode == 204 {
			utl.Die("Successfully deleted secret.\n")
		} else {
			e := r["error"].(map[string]interface{})
			utl.Die(e["message"].(string) + "\n")
		}
	} else {
		utl.Die("Aborted.\n")
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
	//baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	baseUrl := ConstMgUrl + "/beta/servicePrincipals"
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
	var deltaLinkMap map[string]interface{} = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))

	//baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	baseUrl := ConstMgUrl + "/beta/servicePrincipals"
	// Get delta updates only if/when below attributes in $select are modified
	selection := "?$select=displayName,appId,accountEnabled,servicePrincipalType,appOwnerOrganizationId"
	url := baseUrl + "/delta" + selection + "&$top=999"
	headers["Prefer"] = "return=minimal" // This tells API to focus only on specific 'select' attributes
	headers["deltaToken"] = "latest"

	// But first, double-check the base set again to avoid running a delta query on an empty set
	listIsEmpty, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if utl.FileUsable(deltaLinkFile) && deltaAge < (3660*24*27) && listIsEmpty == false {
		// Note that deltaLink file age has to be within 30 days (we do 27)
		tmpVal, _ := utl.LoadFileJson(deltaLinkFile)
		deltaLinkMap = tmpVal.(map[string]interface{})
		url = utl.Str(utl.Str(deltaLinkMap["@odata.deltaLink"]))
		// Base URL is now the cached Delta Link
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
	//baseUrl := ConstMgUrl + "/v1.0/servicePrincipals"
	baseUrl := ConstMgUrl + "/beta/servicePrincipals"
	selection := "?$select=id,displayName,appId,accountEnabled,servicePrincipalType,appOwnerOrganizationId,"
	selection += "appRoleAssignmentRequired,appRoles,disabledByMicrosoftStatus,addIns,alternativeNames,"
	selection += "appDisplayName,homepage,id,info,logoutUrl,notes,oauth2PermissionScopes,replyUrls,"
	selection += "resourceSpecificApplicationPermissions,servicePrincipalNames,tags,customSecurityAttributes"
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

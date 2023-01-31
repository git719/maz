// apps.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
	"time"
)

func PrintApp(x map[string]interface{}, z Bundle) {
	// Print application object in YAML-like format
	if x == nil {
		return
	}
	id := utl.Str(x["id"])

	// Print the most important attributes first
	co := utl.Red(":") // Colorize ":" text to Red
	list := []string{"id", "displayName", "appId"}
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

	// Print secret list & expiry details, not actual secretText (which cannot be retrieve anyway)
	if x["passwordCredentials"] != nil {
		PrintSecretList(x["passwordCredentials"].([]interface{}))
	}

	// Print federated IDs
	url := ConstMgUrl + "/v1.0/applications/" + id + "/federatedIdentityCredentials"
	r, statusCode, _ := ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		fedCreds := r["value"].([]interface{})
		if len(fedCreds) > 0 {
			fmt.Println(utl.Cya("federated_ids") + co)
			for _, i := range fedCreds {
				a := i.(map[string]interface{})
				fmt.Printf("  %-36s  %-30s  %-40s  %s\n", utl.Str(a["id"]), utl.Str(a["name"]),
					utl.Str(a["subject"]), utl.Str(a["issuer"]))
			}
		}
	}

	// Print owners
	url = ConstMgUrl + "/beta/applications/" + id + "/owners"
	r, statusCode, _ = ApiGet(url, z.MgHeaders, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		PrintOwners(r["value"].([]interface{}))
	}

	// Print API permissions
	// Just look under this object's 'requiredResourceAccess' attribute
	if x["requiredResourceAccess"] != nil && len(x["requiredResourceAccess"].([]interface{})) > 0 {
		fmt.Printf(utl.Cya("api_permissions") + co + "\n")
		APIs := x["requiredResourceAccess"].([]interface{}) // Assert to JSON array
		for _, a := range APIs {
			api := a.(map[string]interface{})
			// Getting this API's name and permission value such as Directory.Read.All is a 2-step process:
			// 1) Get all the roles for given API and put their id/value pairs in a map, then
			// 2) Use that map to enumerate and print them

			// Let's drill down into the permissions for this API
			if api["resourceAppId"] == nil {
				fmt.Printf("  %-50s %s\n", "Unknown API", "Missing resourceAppId")
				continue // Skip this API, move on to next one
			}
			resAppId := utl.Str(api["resourceAppId"])

			// Get this API's SP object with all relevant attributes
			url := ConstMgUrl + "/beta/servicePrincipals?filter=appId+eq+'" + resAppId + "'"
			r, _, _ := ApiGet(url, z.MgHeaders, nil)
			ApiErrorCheck("GET", url, utl.Trace(), r) // TODO: Get rid of this by using StatuCode checks, etc
			// Result is a list because this could be a multi-tenant app, having multiple SPs
			if r["value"] == nil {
				fmt.Printf("  %-50s %s\n", resAppId, "Unable to get Resource App object. Skipping this API.")
				continue
			}
			// TODO: Handle multiple SPs

			SPs := r["value"].([]interface{})
			if len(SPs) > 1 {
				utl.Die("  %-50s %s\n", resAppId, "Error. Multiple SPs for this AppId. Aborting.")
			}
			sp := SPs[0].(map[string]interface{}) // Currently only handling the expected single-tenant entry

			// 1. Put all API role id:name pairs into roleMap list
			roleMap := make(map[string]string)
			if sp["appRoles"] != nil { // These are for Application types
				for _, i := range sp["appRoles"].([]interface{}) { // Iterate through all roles
					role := i.(map[string]interface{})
					//utl.PrintJson(role) // DEBUG
					if role["id"] != nil && role["value"] != nil {
						roleMap[utl.Str(role["id"])] = utl.Str(role["value"]) // Add entry to map
					}
				}
			}
			if sp["publishedPermissionScopes"] != nil { // These are for Delegated types
				for _, i := range sp["publishedPermissionScopes"].([]interface{}) {
					role := i.(map[string]interface{})
					//utl.PrintJson(role) // DEBUG
					if role["id"] != nil && role["value"] != nil {
						roleMap[utl.Str(role["id"])] = utl.Str(role["value"])
					}
				}
			}
			if roleMap == nil {
				fmt.Printf("  %-50s %s\n", resAppId, "Error getting list of appRoles.")
				continue
			}

			// 2. Parse this app permissions, and use roleMap to display permission value
			if api["resourceAccess"] != nil && len(api["resourceAccess"].([]interface{})) > 0 {
				Perms := api["resourceAccess"].([]interface{})
				apiName := utl.Str(sp["displayName"]) // This API's name
				for _, i := range Perms {             // Iterate through perms
					perm := i.(map[string]interface{})
					pid := utl.Str(perm["id"]) // JSON string
					var pType string = "?"
					if utl.Str(perm["type"]) == "Role" {
						pType = "Application"
					} else {
						pType = "Delegated"
					}
					fmt.Printf("  %-50s %-40s %s\n", apiName, roleMap[pid], pType)
				}
			} else {
				fmt.Printf("  %-50s %s\n", resAppId, "Error getting list of appRoles.")
			}
		}
	}
}

func AddAppSecret(uuid, expiry, displayName string, z Bundle) {
	if !utl.ValidUuid(uuid) {
		utl.Die("Invalid App UUID.\n")
	}
	if !utl.ValidDate(expiry, "2006-01-02") {
		utl.Die("Expiry '" + expiry + "' is not in 'YYYY-MM-DD' format.\n")
	}
	endDateTime, err := utl.ConvertDateFormat(expiry, "2006-01-02", time.RFC3339Nano)
	if err != nil {
		utl.Die("Error converting Expiry to RFC3339Nano/ISO8601 format.\n")
	}

	payload := map[string]interface{}{
		"passwordCredential": map[string]string{
			"displayName": displayName,
			"endDateTime": endDateTime,
		},
	}
	url := ConstMgUrl + "/v1.0/applications/" + uuid + "/addPassword"
	r, statusCode, _ := ApiPost(url, payload, z.MgHeaders, nil)
	if statusCode == 200 {
		co := utl.Red(":")
		fmt.Printf("%s %s\n", utl.Cya("app_object_id")+co, uuid)
		fmt.Printf("%s %s\n", utl.Cya("new_secret_id")+co, utl.Str(r["keyId"]))
		fmt.Printf("%s %s\n", utl.Cya("new_secret_text")+co, utl.Str(r["secretText"]))
	} else {
		e := r["error"].(map[string]interface{})
		utl.Die(e["message"].(string) + "\n")
	}
}

func RemoveAppSecret(uuid, keyId string, z Bundle) {
	if !utl.ValidUuid(uuid) {
		utl.Die("App UUID is not a valid UUID.\n")
	}
	if !utl.ValidUuid(keyId) {
		utl.Die("Secret ID is not a valid UUID.\n")
	}

	// Get app, display details and secret, and prompt for delete confirmation
	x := GetAzAppByUuid(uuid, z.MgHeaders)
	if x == nil || x["id"] == nil {
		utl.Die("There's no App with this UUID.\n")
	}
	pwdCreds := x["passwordCredentials"].([]interface{})
	if pwdCreds == nil || len(pwdCreds) < 1 {
		utl.Die("App object has no secrets.\n")
	}
	var a map[string]interface{} = nil // Target keyId, Secret ID to be deleted
	for _, i := range pwdCreds {
		targetKeyId := i.(map[string]interface{})
		if utl.Str(targetKeyId["keyId"]) == keyId {
			a = targetKeyId
			break
		}
	}
	if a == nil {
		utl.Die("App object does not have this Secret ID.\n")
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
	co := utl.Red(":")
	fmt.Printf("%s %s\n", utl.Cya("id")+co, utl.Str(x["id"]))
	fmt.Printf("%s %s\n", utl.Cya("appId")+co, utl.Str(x["appId"]))
	fmt.Printf("%s %s\n", utl.Cya("displayName")+co, utl.Str(x["displayName"]))
	fmt.Printf("%s\n", utl.Cya("secret_to_be_deleted")+co)
	fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", utl.Cya2(cId), cName, cHint, cStart, cExpiry)
	if utl.PromptMsg("DELETE above? y/n ") == 'y' {
		payload := map[string]interface{}{"keyId": keyId}
		url := ConstMgUrl + "/v1.0/applications/" + uuid + "/removePassword"
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

func AppsCountLocal(z Bundle) int64 {
	// Return number of entries in local cache file
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_applications.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func AppsCountAzure(z Bundle) int64 {
	// Return number of entries in Azure tenant
	z.MgHeaders["ConsistencyLevel"] = "eventual"
	url := ConstMgUrl + "/v1.0/applications/$count"
	r, _, _ := ApiGet(url, z.MgHeaders, nil)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r["value"] != nil {
		return r["value"].(int64) // Expected result is a single int64 value for the count
	}
	return 0
}

func GetIdMapApps(z Bundle) (nameMap map[string]string) {
	// Return applications id:name map
	nameMap = make(map[string]string)
	apps := GetApps("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range apps {
		x := i.(map[string]interface{})
		if x["id"] != nil && x["displayName"] != nil {
			nameMap[utl.Str(x["id"])] = utl.Str(x["displayName"])
		}
	}
	return nameMap
}

func GetApps(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all Azure AD applications whose searchAttributes match on 'filter'. An empty "" filter returns all.
	// Uses local cache if it's less than cachePeriod old. The 'force' option forces calling Azure query.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_applications.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 86400) // cachePeriod = 1 day in seconds
	if cacheNoGood || force {
		list = GetAzApps(cacheFile, z.MgHeaders, true) // Get all from Azure and show progress (verbose = true)
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

func GetAzApps(cacheFile string, headers map[string]string, verbose bool) (list []interface{}) {
	// Get all Azure AD service principal in current tenant AND save them to local cache file. Show progress if verbose = true.

	// We will first try doing a delta query. See https://docs.microsoft.com/en-us/graph/delta-query-overview
	var deltaLinkMap map[string]interface{} = nil
	deltaLinkFile := cacheFile[:len(cacheFile)-len(filepath.Ext(cacheFile))] + "_deltaLink.json"
	deltaAge := int64(time.Now().Unix()) - int64(utl.FileModTime(deltaLinkFile))

	baseUrl := ConstMgUrl + "/v1.0/applications"
	// Get delta updates only if/when below attributes in $select are modified
	selection := "?$select=displayName,appId,requiredResourceAccess"
	url := baseUrl + "/delta" + selection + "&$top=999"
	headers["Prefer"] = "return=minimal" // This tells API to focus only on specific 'select' attributes

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

func GetAzAppByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure AD application by its Object UUID or by its appId, with extended attributes
	baseUrl := ConstMgUrl + "/v1.0/applications"
	selection := "?$select=id,addIns,api,appId,applicationTemplateId,appRoles,certification,createdDateTime,"
	selection += "deletedDateTime,disabledByMicrosoftStatus,displayName,groupMembershipClaims,id,identifierUris,"
	selection += "info,isDeviceOnlyAuthSupported,isFallbackPublicClient,keyCredentials,logo,notes,"
	selection += "oauth2RequiredPostResponse,optionalClaims,parentalControlSettings,passwordCredentials,"
	selection += "publicClient,publisherDomain,requiredResourceAccess,serviceManagementReference,"
	selection += "signInAudience,spa,tags,tokenEncryptionKeyId,verifiedPublisher,web"
	url := baseUrl + "/" + uuid + selection // First search is for direct Object Id
	r, _, _ := ApiGet(url, headers, nil)
	if r != nil && r["error"] != nil {
		// Second search is for this app's application Client Id
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

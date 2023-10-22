// az_subscriptions.go
// Azure resource Subscriptions

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
)

func PrintSubscription(x map[string]interface{}) {
	// Print subscription object in YAML-like
	if x == nil {
		return
	}
	list := []string{"subscriptionId", "displayName", "state", "tenantId"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s: %s\n", utl.Blu(i), utl.Gre(v))
		}
	}
}

func SubsCountLocal(z Bundle) int64 {
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions."+ConstCacheFileExtension)
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJsonGzip(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func SubsCountAzure(z Bundle) int64 {
	list := GetAzSubscriptions(z)
	return int64(len(list))
}

func GetAzSubscriptionsIds(z Bundle) (scopes []string) {
	// Get all subscription full IDs, i.e. "/subscriptions/UUID", which are commonly
	// used as scopes for Azure resource RBAC role definitions and assignments
	scopes = nil
	subscriptions := GetAzSubscriptions(z)
	for _, i := range subscriptions {
		x := i.(map[string]interface{})
		// Skip disabled and legacy subscriptions
		displayName := utl.Str(x["displayName"])
		state := utl.Str(x["state"])
		if state != "Enabled" || displayName == "Access to Azure Active Directory" {
			continue
		}
		subId := utl.Str(x["id"])
		scopes = append(scopes, subId)
	}
	return scopes
}

func GetIdMapSubs(z Bundle) (nameMap map[string]string) {
	// Return subscription id:name map
	nameMap = make(map[string]string)
	roleDefs := GetSubscriptions("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range roleDefs {
		x := i.(map[string]interface{})
		if x["subscriptionId"] != nil && x["displayName"] != nil {
			nameMap[utl.Str(x["subscriptionId"])] = utl.Str(x["displayName"])
		}
	}
	return nameMap
}

func GetSubscriptions(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all Azure subscriptions matching on 'filter'; return entire list if filter is empty ""

	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions."+ConstCacheFileExtension)
	cacheFileAge := utl.FileAge(cacheFile)
	if utl.InternetIsAvailable() && (force || cacheFileAge == 0 || cacheFileAge > ConstAzCacheFileAgePeriod) {
		// If Internet is available AND (force was requested OR cacheFileAge is zero (meaning does not exist)
		// OR it is older than ConstAzCacheFileAgePeriod) then query Azure directly to get all objects
		// and show progress while doing so (true = verbose below)
		list = GetAzSubscriptions(z)
	} else {
		// Use local cache for all other conditions
		list = GetCachedObjects(cacheFile)
	}

	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	for _, i := range list { // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant subscription attributes
		searchList := []string{"displayName", "subscriptionId", "state"}
		for _, i := range searchList {
			if utl.SubString(utl.Str(x[i]), filter) {
				matchingList = append(matchingList, x)
			}
		}
	}
	return matchingList
}

func GetAzSubscriptions(z Bundle) (list []interface{}) {
	// Get ALL subscription in current Azure tenant AND save them to local cache file
	list = nil                                               // We have to zero it out
	params := map[string]string{"api-version": "2022-09-01"} // subscriptions
	url := ConstAzUrl + "/subscriptions"
	r, _, _ := ApiGet(url, z, params)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r != nil && r["value"] != nil {
		objects := r["value"].([]interface{})
		list = append(list, objects...)
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions."+ConstCacheFileExtension)
	utl.SaveFileJsonGzip(list, cacheFile) // Update the local cache
	return list
}

func GetAzSubscriptionByUuid(uuid string, z Bundle) map[string]interface{} {
	// Get Azure subscription by Object UUID
	params := map[string]string{"api-version": "2022-09-01"} // subscriptions
	url := ConstAzUrl + "/subscriptions/" + uuid
	r, _, _ := ApiGet(url, z, params)
	//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
	return r
}

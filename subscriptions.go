// subscriptions.go

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
	co := utl.Red(":") // Colorize ":" text to Red
	list := []string{"subscriptionId", "displayName", "state", "tenantId"}
	for _, i := range list {
		v := utl.Str(x[i])
		if v != "" { // Only print non-null attributes
			fmt.Printf("%s %s\n", utl.Cya(i)+co, v)
		}
	}
}

func SubsCountLocal(z Bundle) int64 {
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
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
		// Skip legacy subscriptions, since they have no role definitions and calling them causes an error
		if utl.Str(x["displayName"]) == "Access to Azure Active Directory" {
			continue
		}
		scopes = append(scopes, utl.Str(x["id"]))
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
	// Get all subscriptions that match on provided filter. An empty "" filter means return
	// all subscription objects. It always uses local cache if it's within the cache retention
	// period, else it gets them from Azure. Also gets them from Azure if force is specified.
	// TODO: Make cachePeriod a configurable variable
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if cacheNoGood || force {
		list = GetAzSubscriptions(z) // Get the entire set from Azure
	}

	// Do filter matching
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
	r, _, _ := ApiGet(url, z.AzHeaders, params)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r != nil && r["value"] != nil {
		objects := r["value"].([]interface{})
		list = append(list, objects...)
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_subscriptions.json")
	utl.SaveFileJson(list, cacheFile) // Update the local cache
	return list
}

func GetAzSubscriptionByUuid(uuid string, headers map[string]string) map[string]interface{} {
	// Get Azure subscription by Object UUID
	params := map[string]string{"api-version": "2022-09-01"} // subscriptions
	url := ConstAzUrl + "/subscriptions/" + uuid
	r, _, _ := ApiGet(url, headers, params)
	//ApiErrorCheck("GET", url, utl.Trace(), r) // Commented out to do this quietly. Use for DEBUGging
	return r
}

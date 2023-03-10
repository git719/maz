// mgroups.go

package maz

import (
	"fmt"
	"github.com/git719/utl"
	"path/filepath"
)

func MgType(typeIn string) string {
	switch typeIn {
	case "Microsoft.Management/managementGroups":
		return "ManagementGroup"
	case "Microsoft.Management/managementGroups/subscriptions", "/subscriptions":
		return "Subscription"
	default:
		return "??"
	}
}

func PrintMgGroup(x map[string]interface{}) {
	// Print management group object in YAML-like
	if x == nil {
		return
	}
	co := utl.Red(":") // Colorize ":" text to Red
	xProp := x["properties"].(map[string]interface{})
	fmt.Printf("%-12s %s\n", utl.Cya("id")+co, utl.Str(x["name"]))
	fmt.Printf("%-12s %s\n", utl.Cya("displayName")+co, utl.Str(xProp["displayName"]))
	fmt.Printf("%-12s %s\n", utl.Cya("type")+co, MgType(utl.Str(x["type"])))
}

func MgGroupCountLocal(z Bundle) int64 {
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_managementGroups.json")
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile)
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

func MgGroupCountAzure(z Bundle) int64 {
	list := GetAzMgGroups(z)
	return int64(len(list))
}

func GetIdMapMgGroups(z Bundle) (nameMap map[string]string) {
	// Return management groups id:name map
	nameMap = make(map[string]string)
	mgGroups := GetMgGroups("", false, z) // false = don't force a call to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, i := range mgGroups {
		x := i.(map[string]interface{})
		nameMap[utl.Str(x["id"])] = utl.Str(x["name"])
	}
	return nameMap
}

func GetMgGroups(filter string, force bool, z Bundle) (list []interface{}) {
	// Get all managementGroups that match on provided filter. An empty "" filter means return
	// all of them. It always uses local cache if it's within the cache retention period. The force boolean
	// option will force a call to Azure.
	list = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_managementGroups.json")
	cacheNoGood, list := CheckLocalCache(cacheFile, 604800) // cachePeriod = 1 week in seconds
	if cacheNoGood || force {
		list = GetAzMgGroups(z) // Get the entire set from Azure
	}

	// Do filter matching
	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	for _, i := range list { // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant managementGroups attributes
		xProp := x["properties"].(map[string]interface{})
		if utl.SubString(utl.Str(x["name"]), filter) || utl.SubString(utl.Str(xProp["displayName"]), filter) {
			matchingList = append(matchingList, x)
		}
	}
	return matchingList
}

func GetAzMgGroups(z Bundle) (list []interface{}) {
	// Get ALL managementGroups in current Azure tenant AND save them to local cache file
	list = nil                                               // We have to zero it out
	params := map[string]string{"api-version": "2020-05-01"} // managementGroups
	url := ConstAzUrl + "/providers/Microsoft.Management/managementGroups"
	r, _, _ := ApiGet(url, z.AzHeaders, params)
	ApiErrorCheck("GET", url, utl.Trace(), r)
	if r != nil && r["value"] != nil {
		objects := r["value"].([]interface{})
		list = append(list, objects...)
	}
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_managementGroups.json")
	utl.SaveFileJson(list, cacheFile) // Update the local cache
	return list
}

func PrintMgChildren(indent int, children []interface{}) {
	// Recursively print managementGroups children (MGs and subscriptions)
	for _, i := range children {
		child := i.(map[string]interface{})
		Name := utl.Str(child["displayName"])
		Type := MgType(utl.Str(child["type"]))
		if Name == "Access to Azure Active Directory" && Type == "Subscription" {
			continue // Skip legacy subscriptions. We don't care
		}
		fmt.Printf("%*s", indent, " ") // Space padded indent
		padding := 38 - indent
		if padding < 12 {
			padding = 12
		}
		fmt.Printf("%-*s  %-38s  %s\n", padding, Name, utl.Str(child["name"]), Type)
		if child["children"] != nil {
			descendants := child["children"].([]interface{})
			PrintMgChildren(indent+4, descendants)
			// Using recursion here to print additional children
		}
	}
}

func PrintMgTree(z Bundle) {
	// Get current tenant managementGroups and subscriptions tree, and use
	// recursive function PrintMgChildren() to print the entire hierarchy
	url := ConstAzUrl + "/providers/Microsoft.Management/managementGroups/" + z.TenantId
	params := map[string]string{
		"api-version": "2020-05-01", // managementGroups
		"$expand":     "children",
		"$recurse":    "true",
	}
	r, _, _ := ApiGet(url, z.AzHeaders, params)
	ApiErrorCheck("GET", url, utl.Trace(), r) // DEBUG: Need to see when this is failing for some users
	if r["properties"] != nil {
		// Print everything under the hierarchy
		Prop := r["properties"].(map[string]interface{})
		fmt.Printf("%-38s  %-38s  TENANT\n", utl.Str(Prop["displayName"]), utl.Str(Prop["tenantId"]))
		if Prop["children"] != nil {
			children := Prop["children"].([]interface{})
			PrintMgChildren(4, children)
		}
	}
}

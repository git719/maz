// az_definitionsdiff.go
// Azure resource RBAC role definitions difference printer

package maz

import (
	"fmt"
	"github.com/git719/utl"
)

func DiffLists(list1, list2 []interface{}) (added, removed []interface{}, same bool) {
    // Compares two list of strings and returns added and removed items, and whether or not the
	// lists are the same. Note they come in as []interface{} but we know they are strings.

	// Create maps for quick lookup
	set1 := make(map[string]bool)
	for _, i := range list1 {
		set1[utl.Str(i)] = true  // Assert the value as strings, since we know they are strings
	}
	set2 := make(map[string]bool)
	for _, i := range list2 {
		set2[utl.Str(i)] = true
	}

    // Find added items
	for _, i := range list2 {
		if !set1[utl.Str(i)] {
			added = append(added, utl.Str(i))
		}
	}

    // Find removed items
	for _, i := range list1 {
		if !set2[utl.Str(i)] {
			removed = append(removed, utl.Str(i))
		}
	}	

	// Check if lists are the same
	if len(list1) == len(list2) {
		same = true
		for i := range list1 {
			if list1[i] != list2[i] {
				same = false
				break
			}
		}
	} else {
		same = false
	}

    return added, removed, same
}

func DiffRoleDefinitionSpecfileVsAzure(a, b map[string]interface{}, z Bundle) {
	// Prints differences between role definition in Specfile (a) vs what is in Azure (b). The
	// calling function must ensure that both a & b are valid role definition objects from the
	// specfile and from Azure. A generic DiffJsonObject() function would probably be better for this.

	// Gather the SPECFILE object values
	fileProp := a["properties"].(map[string]interface{})
	fileRoleName := utl.Str(fileProp["roleName"])
	fileDesc := utl.Str(fileProp["description"])
	var fileScopes []interface{} = nil
	if fileProp["assignableScopes"] != nil {
		fileScopes = fileProp["assignableScopes"].([]interface{})
	}
	var filePermSet []interface{} = nil
	var filePerms map[string]interface{} = nil
	var filePermsA []interface{} = nil
	var filePermsNA []interface{} = nil
	var filePermsDA []interface{} = nil
	var filePermsNDA []interface{} = nil
	if fileProp["permissions"] != nil {
		filePermSet = fileProp["permissions"].([]interface{})
		if len(filePermSet) == 1 {
			filePerms = filePermSet[0].(map[string]interface{})
			if filePerms["actions"] != nil {
				filePermsA = filePerms["actions"].([]interface{})
			}
			if filePerms["notActions"] != nil {
				filePermsNA = filePerms["notActions"].([]interface{})
			}
			if filePerms["dataActions"] != nil {
				filePermsDA = filePerms["dataActions"].([]interface{})
			}
			if filePerms["notDataActions"] != nil {
				filePermsNDA = filePerms["notDataActions"].([]interface{})
			}
		}
	}

	// Gather the Azure object values
	azureId := utl.Str(b["name"])
	azureProp := b["properties"].(map[string]interface{})
	azureRoleName := utl.Str(azureProp["roleName"])
	azureDesc := utl.Str(azureProp["description"])
	azureScopes := azureProp["assignableScopes"].([]interface{})
	azurePermSet := azureProp["permissions"].([]interface{})
	azurePerms := azurePermSet[0].(map[string]interface{})
	azurePermsA := azurePerms["actions"].([]interface{})
	azurePermsNA := azurePerms["notActions"].([]interface{})
	azurePermsDA := azurePerms["dataActions"].([]interface{})
	azurePermsNDA := azurePerms["notDataActions"].([]interface{})

	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(azureId))
	fmt.Println(utl.Blu("properties") + ":")

	// Display differences

	// roleName
	fmt.Printf("  %s: %s\n", utl.Blu("roleName"), utl.Gre(azureRoleName))
	if fileRoleName != azureRoleName {
		fmt.Printf("  %s: %s\n", utl.Blu("roleName"), utl.Red(fileRoleName))
	}
	
	// description
	fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Gre(azureDesc))
	if fileDesc != azureDesc {
		fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Red(fileDesc))
	}
	
	// scopes
	fmt.Printf("  %s:\n", utl.Blu("assignableScopes"))
	added, removed, _ := DiffLists(fileScopes, azureScopes)
	for _, i := range azureScopes {
		fmt.Printf("    - %s\n", utl.Gre(i))
	}	
	for _, i := range added {
		fmt.Printf("    - %s  # Not in specfile, to be REMOVED\n", utl.Red(i))
	}
	for _, i := range removed {
		fmt.Printf("    - %s  # In specfile, to be ADDED\n", utl.Mag(i))
	}

	// permissionss
	fmt.Printf("  %s:\n", utl.Blu("permissions"))
	// actions
	if len(azurePermsA) > 0 {
		fmt.Printf("    - %s:\n", utl.Blu("actions"))
		added, removed, _ := DiffLists(filePermsA, azurePermsA)
		for _, i := range azurePermsA {
			fmt.Printf("        - %s\n", utl.Gre(i))
		}	
		for _, i := range added {
			fmt.Printf("        - %s  # Not in specfile, to be REMOVED\n", utl.Red(i))
		}
		for _, i := range removed {
			fmt.Printf("        - %s  # In specfile, to be ADDED\n", utl.Mag(i))
		}	
	}
	// notActions
	if len(azurePermsNA) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("notActions"))
		added, removed, _ := DiffLists(filePermsNA, azurePermsNA)
		for _, i := range azurePermsNA {
			fmt.Printf("        - %s\n", utl.Gre(i))
		}	
		for _, i := range added {
			fmt.Printf("        - %s  # Not in specfile, to be REMOVED\n", utl.Red(i))
		}
		for _, i := range removed {
			fmt.Printf("        - %s  # In specfile, to be ADDED\n", utl.Mag(i))
		}	
	}
	// dataActions
	if len(azurePermsDA) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("dataActions"))
		added, removed, _ := DiffLists(filePermsDA, azurePermsDA)
		for _, i := range azurePermsDA {
			fmt.Printf("        - %s\n", utl.Gre(i))
		}	
		for _, i := range added {
			fmt.Printf("        - %s  # Not in specfile, to be REMOVED\n", utl.Red(i))
		}
		for _, i := range removed {
			fmt.Printf("        - %s  # In specfile, to be ADDED\n", utl.Mag(i))
		}	
	}
	// notDataActions
	if len(azurePermsNDA) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("notDataActions"))
		added, removed, _ := DiffLists(filePermsNDA, azurePermsNDA)
		for _, i := range azurePermsNDA {
			fmt.Printf("        - %s\n", utl.Gre(i))
		}	
		for _, i := range added {
			fmt.Printf("        - %s  # Not in specfile, to be REMOVED\n", utl.Red(i))
		}
		for _, i := range removed {
			fmt.Printf("        - %s  # In specfile, to be ADDED\n", utl.Mag(i))
		}	
	}
}


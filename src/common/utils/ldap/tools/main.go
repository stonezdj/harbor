package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/goharbor/harbor/src/common/utils/ldap"

	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils/log"
)

//LDAPConfigAll ...
type LDAPConfigAll struct {
	LDAPConf      models.LdapConf      `json:"ldap_conf,omitempty"`
	LDAPGroupConf models.LdapGroupConf `json:"ldap_group_conf,omitempty"`
}

func main() {
	f, _ := os.Create("./test.log")
	log.SetOutput(f)

	configJSON := flag.String("config", "ldap.json", "LDAP json file")
	username := flag.String("username", "", "search this username in LDAP")
	flag.Parse()

	jsonFile, err := os.Open(*configJSON)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	var ldapConfigAll LDAPConfigAll
	json.Unmarshal([]byte(byteValue), &ldapConfigAll)

	log.Debugf("ldapConfigAll %#v\n", ldapConfigAll)

	fmt.Println("================================================")
	fmt.Println("Verify basic LDAP information")
	fmt.Println("Start to ping LDAP server")

	session, err := ldap.CreateWithAllConfig(ldapConfigAll.LDAPConf, ldapConfigAll.LDAPGroupConf)
	if err != nil {
		fmt.Printf("Error %+v\n", err)
		return
	}
	err = ldap.ConnectionTestWithAllConfig(ldapConfigAll.LDAPConf, ldapConfigAll.LDAPGroupConf)
	if err != nil {
		fmt.Printf("Error at connection test, %+v\n", err)
		return
	}
	DumpResult("Success to ping LDAP server")
	fmt.Println("Start to search users")
	session.Open()
	defer session.Close()
	session.Bind(ldapConfigAll.LDAPConf.LdapSearchDn, ldapConfigAll.LDAPConf.LdapSearchPassword)
	result, err := session.SearchUser("")
	if err != nil {
		DumpResult(fmt.Sprintf("Failed to search LDAP, error %v", err))
		return
	}
	if len(result) == 0 {
		DumpResult(fmt.Sprintf("No LDAP user found in current search conditions."))
	} else {
		DumpResult(fmt.Sprintf("Found %d LDAP users in current search conditions", len(result)))
	}
	if len(*username) > 0 {
		fmt.Printf("Trying to find user %v\n", *username)
		singleUser, err := session.SearchUser(*username)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(singleUser) == 0 {
			DumpResult(fmt.Sprintf("The user %v is not found!", *username))
		} else {
			DumpResult(fmt.Sprintf("User %v found!", *username))
			DumpResult(fmt.Sprintf("User in the group: %+v", singleUser[0].GroupDNList))

		}
	}
	fmt.Println("================================================")

	fmt.Println("Verify LDAP group configurations")
	if len(ldapConfigAll.LDAPGroupConf.LdapGroupBaseDN) > 0 {
		fmt.Println("Trying to search group in current search conditions.")
		groups, err := session.SearchGroupByName("")
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(groups) == 0 {
			DumpResult("No user group found!")
		} else {
			DumpResult(fmt.Sprintf("Found %v groups in current condition.", len(groups)))
		}
	}

	if len(ldapConfigAll.LDAPGroupConf.LdapGroupAdminDN) > 0 {
		fmt.Println("Trying to search the group with admin privileges.")
		groups, err := session.SearchGroupByDN(ldapConfigAll.LDAPGroupConf.LdapGroupAdminDN)
		if err != nil {
			fmt.Println(err)
			return
		}

		DumpResult(fmt.Sprintf("Found %v groups with admin privileges.", len(groups)))
		if len(groups) > 0 {
			fmt.Printf("Trying to find users in this group: %v\n", ldapConfigAll.LDAPGroupConf.LdapGroupAdminDN)
			count := 0
			userList, err := session.SearchUser("")
			if err != nil {
				fmt.Println(err)
			}
			for _, user := range userList {
				log.Debugf("username: %v, groupDNList=%+v", user.Username, user.GroupDNList)
				if stringInSlice(ldapConfigAll.LDAPGroupConf.LdapGroupAdminDN, user.GroupDNList) {
					count++
				}
			}
			DumpResult(fmt.Sprintf("Found %v users in this group", count))

		}

	}
	fmt.Println("================================================")
}
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// DumpResult ...
func DumpResult(msg string) {
	fmt.Println("==> " + msg)
}

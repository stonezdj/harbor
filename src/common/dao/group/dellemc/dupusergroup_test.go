//  Copyright Project Harbor Authors
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package dellemc

import (
	"bufio"
	"fmt"
	"github.com/astaxie/beego/orm"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/dao/group"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/lib/log"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	orm.Debug = true
	// databases := []string{"mysql", "sqlite"}
	databases := []string{"postgresql"}
	for _, database := range databases {
		log.Infof("run test cases for database: %s", database)

		result := 1
		switch database {
		case "postgresql":
			dao.PrepareTestForPostgresSQL()
		default:
			log.Fatalf("invalid database: %s", database)
		}

		result = m.Run()

		if result != 0 {
			os.Exit(result)
		}
	}

}

func TestInsertUserGroup(t *testing.T) {
	file, err := os.Open("/path/usergroup_database.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	r := 0
	scanner := bufio.NewScanner(file)
	fmt.Println("=====================")
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		r = r + 1
		if r < 3 {
			continue
		}
		line := scanner.Text()
		//fmt.Printf("row=%v, %s\n",r, scanner.Text())
		InsertUserGroup(line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("=====================")
}
func InsertUserGroup(line string) {
	groupName, _, groupDN := splitLine(line)
	if len(groupName) == 0 {
		return
	}
	ug := &models.UserGroup{
		GroupName:   groupName,
		GroupType:   1,
		LdapGroupDN: groupDN,
	}
	o := orm.NewOrm()
	_, err := o.Insert(ug)
	if err != nil {
		fmt.Printf("error: %v, groupdn:%v\n", err, ug.LdapGroupDN)
	}
}

func splitLine(line string) (groupName string, groupType int, ldapgroupDN string) {
	parts := strings.SplitN(line, "|", 6)
	if len(parts) != 6 {
		fmt.Printf("==========skip current row======%v", line)
		return "", 1, ""
	}
	groupName = strings.TrimSpace(parts[1])
	groupType = 1
	ldapgroupDN = strings.TrimSpace(parts[3])
	return
}

func TestQueryUsergroup(t *testing.T) {
	ugs := []models.UserGroup{
		models.UserGroup{LdapGroupDN: strings.ToLower("CN=DAA Devops Communications,OU=Distribution Lists,DC=amer,DC=dell,DC=com"), GroupName: "DAA Devops Communications", GroupType: 1},
	}
	ids, err := group.PopulateGroup(ugs)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("result: %+v", ids)
	}
}
func TestCheckOnBoard(t *testing.T) {

	file, err := os.Open("/path/grouplist1.txt")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	r := 0
	scanner := bufio.NewScanner(file)
	fmt.Println("=====================")
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		r = r + 1
		if r < 3 {
			continue
		}
		line := scanner.Text()
		groupName, dn := parseGroupNameAndDN(line)
		ug := models.UserGroup{GroupName: groupName, GroupType: 1, LdapGroupDN: dn}
		ugs := []models.UserGroup{ug}
		ids, err := group.PopulateGroup(ugs)
		if err != nil {
			fmt.Printf("error %v, userGroup:%+v", err, ugs)
		} else {
			fmt.Printf("%+v", ids)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
func parseGroupNameAndDN(line string) (groupName, dn string) {
	parts := strings.Split(line, "[")
	parts2 := strings.Split(parts[1], "]")
	groupDN := parts2[0]
	fmt.Println(parts2[0])
	if strings.HasPrefix(groupDN, "CN=") {
		ndn := strings.TrimPrefix(groupDN, "CN=")
		fmt.Println(ndn)
		dnParts := strings.SplitN(ndn, ",", 2)
		groupName = dnParts[0]
	}
	return groupName, groupDN
}

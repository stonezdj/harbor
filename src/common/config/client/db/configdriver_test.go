package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/goharbor/harbor/src/common/config"
	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/utils/log"
)

var testingMetaDataArray = []config.Item{
	{Name: "ldap_search_scope", Type: "int", Scope: "system", Group: "ldapbasic", HasDefaultValue: true, DefaultValue: "3"},
	{Name: "ldap_search_dn", Type: "string", Scope: "user", Group: "ldapbasic", HasDefaultValue: true, DefaultValue: "cn=admin,dc=example,dc=com"},
	{Name: "ulimit", Type: "int64", Scope: "user", Group: "ldapbasic", HasDefaultValue: true, DefaultValue: "99999"},
	{Name: "ldap_verify_cert", Type: "bool", Scope: "user", Group: "ldapbasic", HasDefaultValue: true, DefaultValue: "true"},
	{Name: "sample_map_setting", Type: "map", Scope: "user", Group: "ldapbasic", HasDefaultValue: false},
}

func TestMain(m *testing.M) {
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
		result = testForAll(m)

		if result != 0 {
			os.Exit(result)
		}
	}
}

func testForAll(m *testing.M) int {

	rc := m.Run()
	clearAll()
	return rc
}

func clearAll() {
	tables := []string{"project_member",
		"project_metadata", "access_log", "repository", "replication_policy",
		"replication_target", "replication_job", "replication_immediate_trigger", "img_scan_job",
		"img_scan_overview", "clair_vuln_timestamp", "project", "harbor_user"}
	for _, t := range tables {
		if err := dao.ClearTable(t); err != nil {
			log.Errorf("Failed to clear table: %s,error: %v", t, err)
		}
	}
}

func TestDBDriver_Load(t *testing.T) {
	cd := NewDBConfigureStore()
	cd.InitFromArray(testingMetaDataArray)
	cd.Load()
	cfgValue, err := cd.GetSettingByGroup("ldapbasic")
	if err != nil {
		t.Errorf("Error occurred when : %v", err)
	}
	for _, item := range cfgValue {
		fmt.Printf("config value is %+v", item.GetString())
	}
}

func TestDBDriver_Save(t *testing.T) {
	cd := NewDBConfigureStore()
	config.MetaData.InitMetaDataFromArray(testingMetaDataArray)
	cd.InitFromArray(testingMetaDataArray)
	cd.Load()
	cd.UpdateConfigValue("ldap_search_dn", "cn=administrator,dc=vmware,dc=com")
	cd.UpdateConfigValue("ldap_verify_cert", "true")
	cd.UpdateConfigValue("ldap_search_scope", "2")
	cd.Save()
}

package config

import (
	"testing"
)

func TestInitMetaDataFromJsonString(t *testing.T) {
	jsonString := `[
		{"name":"ldap_search_scope", "type":"int", "scope":"system", "group":"ldapbasic"},
		{"name":"ldap_search_dn", "type":"string", "scope":"user", "group":"ldapbasic"}
	]`
	InitMetaDataFromJSONString(jsonString)

	if item, ok := ConfigureMetaData["ldap_search_scope"]; !ok {
		t.Error("failed to find ldap_search_scope!")
	} else {
		if item.Type != IntType {
			t.Errorf("Failed to get the type,expect int, actual type %v", item.Type)
		}
	}
	if item, ok := ConfigureMetaData["ldap_search_dn"]; !ok {
		t.Error("failed to find ldap_search_dn")
	} else {
		if item.Type != StringType {
			t.Errorf("Failed to get type string, actual int type %v", item.Type)
		}
	}
}

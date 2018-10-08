package usersetting

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/utils/log"
)

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
	return rc
}

func TestMethod(t *testing.T) {
	fmt.Println("Testing message")
}

func TestUserSettingManager_Load(t *testing.T) {
	tests := []struct {
		name    string
		usm     *UserSettingManager
		want    map[string]interface{}
		wantErr bool
	}{
		{"normal", &UserSettingManager{}, map[string]interface{}{"sample": 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usm := &UserSettingManager{}
			got, err := usm.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("UserSettingManager.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UserSettingManager.Load() = %v, want %v", got, tt.want)
			}
		})
	}
}

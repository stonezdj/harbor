package configuration

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/goharbor/harbor/src/common/dao"
	"github.com/goharbor/harbor/src/common/models"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/spf13/viper"
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
	viper.SetConfigType("properties")
	viper.SetConfigFile("/Users/daojunz/Documents/goworkdir/src/github.com/goharbor/harbor/make/harbor.cfg")
	err := viper.ReadInConfig()
	if err != nil {
		t.Errorf("Error occurred when : %v", err)
	}
	fmt.Printf("hostname=%v\n", viper.Get("hostname"))

	allkeys := viper.AllKeys()
	configEntries := make([]models.ConfigEntry, 0)
	for _, item := range allkeys {
		fmt.Printf("key=%v, value=%v\n", item, viper.Get(item))
		entry := models.ConfigEntry{Key: item, Value: strings.TrimSpace(viper.GetString(item))}
		configEntries = append(configEntries, entry)
	}

	err = dao.SaveConfigEntries(configEntries)
	if err != nil {
		t.Errorf("Error occurred when save confg : %v", err)
	}
	fmt.Println("Done")
}

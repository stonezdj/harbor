package manager

import (
	"fmt"
	commonhttp "github.com/goharbor/harbor/src/common/http"
	"net/http"
	"reflect"
	"testing"
)

func TestNewDefaultManger(t *testing.T) {
	tests := []struct {
		name string
		want *DefaultManager
	}{
		{want: &DefaultManager{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDefaultManger(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDefaultManger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExample(t *testing.T) {
	skipCertVerify := true
	address := "http://10.160.218.91:9999"
	req, err := http.NewRequest(http.MethodPost, address, nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Transport: commonhttp.GetHTTPTransport(skipCertVerify),
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	fmt.Printf("policy test success with address %s, skip cert verify :%v\n", address, skipCertVerify)

}

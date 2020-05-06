package repoproxy

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// MyResponseWriter ...
type MyResponseWriter struct {
	http.ResponseWriter
	Buf *bytes.Buffer
}

func (mrw *MyResponseWriter) Write(p []byte) (int, error) {
	mrw.Buf.Write(p)
	return mrw.ResponseWriter.Write(p)
}

func GetManifest(manifestUrl string, token *BearerToken) []byte {
	req2, err := http.NewRequest("GET", manifestUrl, nil)
	if err != nil {
		fmt.Println(err)
	}
	bt := fmt.Sprintf("Bearer %s", token.Token)
	fmt.Println(bt)
	req2.Header.Add("Authorization", bt)
	req2.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	client := getHttpClient()
	resp2, err := client.Do(req2)
	if err != nil {
		fmt.Println(err)
	}
	body2, err := ioutil.ReadAll(resp2.Body)
	if err != nil {
		return []byte{}
	}
	return body2
}

func GetAuthorization(name, password string) string {
	return string(base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", name, password))))
}

func RetrieveBearerToken(hostname string, username string, password string, repoName string, opType string) *BearerToken {
	client := getHttpClient()
	urlString := fmt.Sprintf("https://%s/service/token?account=%s&scope=repository:%s:%s&service=harbor-registry", hostname, username, repoName, opType)
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", GetAuthorization(username, password))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Printf("%v\n", string(body))
	token := &BearerToken{}
	json.Unmarshal(body, token)
	return token
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: tr,
	}
	return client
}

func ExtractDigest(body2 []byte) []string {
	fmt.Printf("The manifest content1 is %q\n", string(body2))
	digestList := make([]string, 0)
	rawIn := json.RawMessage(body2)
	bytes, err := rawIn.MarshalJSON()
	if err != nil {
		panic(err)
	}
	m := &Manifest{}
	fmt.Printf("The manifest content2 is %q\n", bytes)
	err = json.Unmarshal(bytes, m)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("manifest is : %#v", m)
	for _, l := range m.Layers {
		fmt.Printf("digest:%v\n", l["digest"])
		digestList = append(digestList, fmt.Sprintf("%v", l["digest"]))
	}
	return digestList
}

type BearerToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Manifest struct {
	SchemaVersion int                      `json:"schemaVersion"`
	MediaType     string                   `json:"mediaType"`
	Config        Config                   `json:"config"`
	Layers        []map[string]interface{} `json:"layers"`
}
type Config struct {
	MediaType string `json:"mediaType"`
}

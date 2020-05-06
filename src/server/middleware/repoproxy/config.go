package repoproxy

import "os"

type ProxyAuth struct {
	URL      string
	Username string
	Password string
}

var ProxyConfig = &ProxyAuth{
	URL:      os.Getenv("PROXY_URL"),
	Username: os.Getenv("PROXY_USERNAME"),
	Password: os.Getenv("PROXY_PASSWORD"),
}

package notifications

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/golang/glog"
)

const requestTimeout = 10 * time.Second

// The proxy schemes net/http knows what to do with. Anything else reaches the
// transport and fails every single notification, so it is worth catching here:
// "localhost:3128" parses happily as a scheme of its own with no host at all.
var proxySchemes = []string{"http", "https", "socks5", "socks5h"}

func usableProxy(proxy *url.URL) bool {
	return proxy.Host != "" && slices.Contains(proxySchemes, proxy.Scheme)
}

type Client struct {
	httpClient *http.Client
}

// CreateClient creates a client for the notification endpoint. An empty proxy
// means the endpoint is reached directly.
func CreateClient(proxy string) Client {
	// Notification endpoints are often internal services with a certificate
	// kube-monkey's image has no root for, and there is nothing secret in a
	// notification, so the certificate is not checked
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		switch {
		case err != nil:
			glog.Errorf("Ignoring the notifications proxy %s because it is not a valid URL. Error: %v", proxy, err)
		case !usableProxy(proxyURL):
			// Sending notifications straight out beats sending none at all
			glog.Errorf("Ignoring the notifications proxy %s because it is not a %s URL with a host, e.g. http://proxy:3128", proxy, strings.Join(proxySchemes, ", "))
		default:
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return Client{httpClient: &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}}
}

// Request posts the body to the endpoint. A response outside the 2xx range is
// an error.
//
// Errors leave the endpoint out because it can carry a secret token in its
// path, and callers log these errors
func (c Client) Request(endpoint string, requestBody string, headers map[string]string) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("new http request: %v", err)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %v", err)
	}
	defer resp.Body.Close()

	// Read the body either way: on failure it tells the user why, and on success
	// draining it lets the connection be reused
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("POST returned %d %s, expected 2xx", resp.StatusCode, strings.TrimSuffix(string(body), "\n"))
	}

	return nil
}

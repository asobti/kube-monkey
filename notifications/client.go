package notifications

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

// CreateClient creates a new client with a default timeout
func CreateClient(proxy *string) Client {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	transport := http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxy != nil {
		proxyUrl, _ := url.Parse(*proxy)
		transport.Proxy = http.ProxyURL(proxyUrl)
	}
	client.Transport = &transport
	return Client{httpClient: client}
}

// Request sends an http request and returns error also if response code is NOT 2XX
func (c Client) Request(endpoint string, requestBody string, headers map[string]string) error {
	body := bytes.NewBufferString(requestBody)

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return fmt.Errorf("new http request: %s %s: %v", "POST", endpoint, err)
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, _ := ioutil.ReadAll(resp.Body) // try to read response body as well to give user more info why request failed
		return fmt.Errorf("%s %s returned %d %s, expected 2xx",
			"POST", endpoint, resp.StatusCode, strings.TrimSuffix(string(b), "\n"))
	}

	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		return fmt.Errorf("read response body: %s %s: %v", "POST", endpoint, err)
	}
	return nil
}

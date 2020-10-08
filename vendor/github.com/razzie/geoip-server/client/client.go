package client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/razzie/geoip-server/geoip"
)

// Client is a lightweight http client to request location data from geoip-server
type Client struct {
	ServerAddress string
}

// DefaultClient is the default client
var DefaultClient geoip.Client = NewClient("https://geoip.gorzsony.com")

// NewClient returns a new client
func NewClient(serverAddr string) *Client {
	return &Client{ServerAddress: serverAddr}
}

// Provider returns the provider this client is requesting locations from
func (c *Client) Provider() string {
	u, err := url.Parse(c.ServerAddress)
	if err != nil {
		return c.ServerAddress
	}
	return u.Host
}

// GetLocation retrieves the location data of an IP or hostname from geoip-server
func (c *Client) GetLocation(ctx context.Context, hostname string) (*geoip.Location, error) {
	req, _ := http.NewRequest("GET", c.ServerAddress+"/"+hostname, nil)
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var loc geoip.Location
	return &loc, json.Unmarshal(result, &loc)
}

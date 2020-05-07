package client

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/razzie/geoip-server/geoip"
)

// Client is a lightweight http client to request location data from geoip-server
type Client struct {
	ServerAddress string
}

// DefaultClient is the default client
var DefaultClient = *NewClient()

// NewClient returns a new client
func NewClient() *Client {
	return &Client{ServerAddress: "https://geoip.gorzsony.com"}
}

// GetLocation requests the location data of an IP or hostname from geoip-server
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

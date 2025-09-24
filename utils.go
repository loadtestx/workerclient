package workerclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/caio/go-tdigest/v4"
)

type TDNode struct {
	Mean  float64 `json:"mean"`
	Count uint64  `json:"count"`
}

func SerializeTDigest(td *tdigest.TDigest) []TDNode {
	rslice := []TDNode{}
	td.ForEachCentroid(func(mean float64, count uint64) bool {
		rslice = append(rslice, TDNode{
			Mean:  mean,
			Count: count,
		})
		return true
	})
	return rslice
}

func UnserializeTDigest(nodes []TDNode) *tdigest.TDigest {
	td, _ := tdigest.New()
	for _, n := range nodes {
		td.AddWeighted(n.Mean, n.Count)
	}
	return td
}

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	return &HTTPClient{
		client: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
	}
}

func (c *HTTPClient) PostJSON(url string, requestBody interface{}, responseBody interface{}) error {
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if responseBody != nil {
		if err := json.Unmarshal(respBytes, responseBody); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

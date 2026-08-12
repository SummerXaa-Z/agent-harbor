package httpapi

import "net/http"

func doUpstreamRequest(request *http.Request) (*http.Response, error) {
	// Upstream endpoints are validated before they are stored. Following a
	// redirect would cross that boundary and could expose credentials or reach
	// an otherwise blocked private destination.
	client := &http.Client{
		Transport: http.DefaultClient.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return client.Do(request)
}

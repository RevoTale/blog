package gql

import (
	"net/http"
	"time"

	"blog/internal/config"
	genqlientgraphql "github.com/Khan/genqlient/graphql"
)

func NewClient(cfg config.Config) genqlientgraphql.Client {
	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &authTransport{
			base:  http.DefaultTransport,
			token: cfg.GraphQLAuthToken,
		},
	}

	return genqlientgraphql.NewClient(cfg.GraphQLEndpoint, client)
}

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token == "" {
		return t.base.RoundTrip(req)
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "JWT "+t.token)
	return t.base.RoundTrip(clone)
}

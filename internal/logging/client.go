package logging

import (
	"context"

	"cloud.google.com/go/logging/logadmin"
)

type Client struct {
	client    *logadmin.Client
	projectID string
}

func NewClient(ctx context.Context, projectID string) (*Client, error) {
	client, err := logadmin.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:    client,
		projectID: projectID,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) ProjectID() string {
	return c.projectID
}

func (c *Client) LogAdminClient() *logadmin.Client {
	return c.client
}

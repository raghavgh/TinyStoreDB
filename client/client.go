package client

import (
	"context"

	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TinyStoreClient struct {
	conn   *grpc.ClientConn
	client tspb.TinyStoreServiceClient
}

func NewTinyStoreClient(addr string) (*TinyStoreClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // ✅ FIXED
	)
	if err != nil {
		return nil, err
	}

	client := tspb.NewTinyStoreServiceClient(conn)
	return &TinyStoreClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *TinyStoreClient) Set(ctx context.Context, key, value string) error {
	_, err := c.client.Set(ctx, &tspb.SetRequest{
		Key:   key,
		Value: value,
	})
	return err
}

func (c *TinyStoreClient) Get(ctx context.Context, key string) (string, error) {
	resp, err := c.client.Get(ctx, &tspb.GetRequest{Key: key})
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

func (c *TinyStoreClient) Delete(ctx context.Context, key string) (bool, error) {
	resp, err := c.client.Delete(ctx, &tspb.DeleteRequest{Key: key})
	if err != nil {
		return false, err
	}

	return resp.Success, nil
}

func (c *TinyStoreClient) Compact(ctx context.Context) error {
	_, err := c.client.Compact(ctx, &tspb.CompactRequest{})
	if err != nil {
		return err
	}
	return nil
}

func (c *TinyStoreClient) Close() error {
	return c.conn.Close()
}

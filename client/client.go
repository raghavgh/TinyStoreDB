package client

import (
	"context"

	tspb "github.com/raghavgh/TinyStoreDB/server/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type TinyStoreClient struct {
	conn   *grpc.ClientConn
	client tspb.TinyStoreServiceClient
}

func New(addr string, token *string) (*TinyStoreClient, error) {
	authInterceptor := grpc.WithUnaryInterceptor(func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.New(map[string]string{
			"authorization": *token,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	})

	var (
		conn *grpc.ClientConn
		err  error
	)

	if token != nil {
		conn, err = grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			authInterceptor,
		)
	} else {
		conn, err = grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if err != nil {
		return nil, err
	}

	client := tspb.NewTinyStoreServiceClient(conn)
	return &TinyStoreClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *TinyStoreClient) Set(ctx context.Context, key, value string, ttl *uint64) error {
	_, err := c.client.Set(ctx, &tspb.SetRequest{
		Key:   key,
		Value: value, Ttl: ttl,
	})
	return err
}

func (c *TinyStoreClient) Exist(ctx context.Context, key string) (bool, error) {
	exist, err := c.client.Exist(ctx, &tspb.ExistRequest{
		Key: key,
	})
	if err != nil {
		return false, err
	}

	return exist.Value, nil
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

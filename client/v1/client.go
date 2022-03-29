package clientv1

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	corev1 "kaiko.com/go/grpc/core/v1"
	"kaiko.com/go/grpc/kaikosdk"
	marketupdatev1 "kaiko.com/go/grpc/marketupdate/v1"
)

type (
	// Kaiko is a Kaiko client.
	Client struct {
		authCtx func(context.Context) context.Context
		conn    *grpc.ClientConn
	}
	// Instrument is a financial Instrument.
	Instrument struct {
		cli  *Client
		crit *corev1.InstrumentCriteria
	}
	// MarketUpdate is a market update.
	MarketUpdate struct {
		*marketupdatev1.Response

		Commodity    string
		TSCollection time.Time
		TSEvent      time.Time
		TSExchange   time.Time
		UpdateType   string
	}
)

// NewClient returns a new Kaiko Client.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	cfg := newConfig(opts...)

	endp := "gateway-v0-grpc.kaiko.ovh:443"
	dopts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})),
	}

	if cfg.ICEEnabled {
		endp = "ice.kaiko.com:80"
		dopts = dopts[:0]
	}

	conn, err := grpc.Dial(endp, append(dopts, cfg.GRPCDialOptions...)...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC connection: %w", err)
	}

	return &Client{
		authCtx: func(ctx context.Context) context.Context {
			return metadata.AppendToOutgoingContext(ctx, "authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))
		},
		conn: conn,
	}, nil
}

// Close closes c.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close gRPC connection: %w", err)
	}

	return nil
}

// Instrument returns an instrument for the combination of exchange ex, class cl and code co.
func (c *Client) Instrument(exc, cla, cod string) *Instrument {
	return &Instrument{
		cli: c,
		crit: &corev1.InstrumentCriteria{
			Code:            cod,
			Exchange:        exc,
			InstrumentClass: cla,
		},
	}
}

// StreamMarketUpdates streams market updates for instrument i, calling handle for each one.
func (i *Instrument) StreamMarketUpdates(ctx context.Context, handle func(*MarketUpdate), cs ...string) error {
	cdts := make([]marketupdatev1.Commodity, 0, len(cs))

	for _, c := range cs {
		cdt := marketupdatev1.Commodity_value[fmt.Sprintf("COMMODITY_%s", c)]
		cdts = append(cdts, marketupdatev1.Commodity(cdt))
	}

	sub, err := kaikosdk.NewStreamMarketUpdateServiceV1Client(i.cli.conn).
		Subscribe(i.cli.authCtx(ctx), &marketupdatev1.Request{
			InstrumentCriteria: i.crit,
			Commodities:        cdts,
		})
	if err != nil {
		return fmt.Errorf("failed to subscribe to market updates: %w", err)
	}

	for {
		res, err := sub.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}

			return fmt.Errorf("failed to receive market updates: %w", err)
		}

		var mu MarketUpdate

		mu.fromProtobuf(res)
		handle(&mu)
	}
}

func (mu *MarketUpdate) fromProtobuf(res *marketupdatev1.Response) {
	mu.Response = res
	mu.Commodity = strings.TrimPrefix(marketupdatev1.Commodity_name[int32(res.Commodity)], "COMMODITY_")
	mu.TSCollection = res.TsCollection.Value.AsTime()
	mu.TSEvent = res.TsEvent.AsTime()
	mu.TSExchange = res.TsExchange.Value.AsTime()
	mu.UpdateType = strings.TrimPrefix(marketupdatev1.Response_Type_name[int32(res.UpdateType)], "TYPE_")
}

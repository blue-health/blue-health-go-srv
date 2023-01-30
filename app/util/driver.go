package util

import (
	"context"
	"database/sql/driver"
	"net"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/lib/pq"
	"golang.org/x/net/proxy"
)

type CloudSQLDriver struct {
	n string
	d *cloudsqlconn.Dialer
}

var (
	_ driver.Driver       = (*CloudSQLDriver)(nil)
	_ proxy.Dialer        = (*CloudSQLDriver)(nil)
	_ proxy.ContextDialer = (*CloudSQLDriver)(nil)
)

func NewCloudSQLDriver(ctx context.Context, cloudName string, opts ...cloudsqlconn.Option) (*CloudSQLDriver, func() error, error) {
	d, err := cloudsqlconn.NewDialer(ctx, opts...)
	if err != nil {
		return nil, nil, err
	}

	return &CloudSQLDriver{n: cloudName, d: d}, func() error { return d.Close() }, nil
}

func (d *CloudSQLDriver) Open(name string) (driver.Conn, error) {
	return pq.DialOpen(d, name)
}

func (d *CloudSQLDriver) Dial(_, _ string) (net.Conn, error) {
	return d.d.Dial(context.Background(), d.n)
}

func (d *CloudSQLDriver) DialTimeout(_, _ string, _ time.Duration) (net.Conn, error) {
	return d.d.Dial(context.Background(), d.n)
}

func (d *CloudSQLDriver) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	return d.d.Dial(ctx, d.n)
}

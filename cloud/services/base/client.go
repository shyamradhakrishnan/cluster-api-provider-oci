package base

import (
	"context"
)

type BaseClient interface {
	GenerateToken(ctx context.Context, clusterID string) (string, error)
}

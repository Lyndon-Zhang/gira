package proj

import (
	"context"
	"path"

	"github.com/Lyndon-Zhang/gira"
)

func LoadResource(ctx context.Context, loader gira.ResourceLoader, compress bool) error {
	// 连接resourcedb
	if resourceDbClient, err := NewResourceDbClient(ctx); err != nil {
		return err
	} else if err := loader.LoadResource(ctx, resourceDbClient, path.Join(Dir.ResourceDir, "conf"), compress); err != nil {
		return err
	} else {
		return nil
	}
}

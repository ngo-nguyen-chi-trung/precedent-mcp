package clients

import (
	"sync"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"

	"github.com/ngo/precedent_mcp/internal/env"
)

var (
	weaviateInstance *weaviate.Client
	weaviateOnce     sync.Once
)

func GetWeaviateInstance() *weaviate.Client {
	weaviateOnce.Do(func() {
		var err error
		weaviateInstance, err = weaviate.NewClient(weaviate.Config{
			Host:           env.WeaviateHost.GetValue(),
			Scheme:         env.WeaviateScheme.GetValue(),
			StartupTimeout: env.DurationWeaviateStartup,
		})
		if err != nil {
			panic(err)
		}
	})
	return weaviateInstance
}

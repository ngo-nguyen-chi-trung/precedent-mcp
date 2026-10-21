package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

type Capabilities struct {
	Tools      struct{}            `json:"tools"`
	Resources  struct{}            `json:"resources"`
	Extensions map[string]struct{} `json:"extensions"`
}

type DiscoverResult struct {
	ResultBase
	CacheHints
	SupportedVersions []string     `json:"supportedVersions"`
	Capabilities      Capabilities `json:"capabilities"`
}

func (r *Response) FromDiscoverResponse(res *pb.DiscoverResponse) {
	extensions := make(map[string]struct{}, len(res.GetExtensions()))
	for _, extension := range res.GetExtensions() {
		extensions[extension] = struct{}{}
	}
	r.Result = DiscoverResult{
		ResultBase:        complete,
		CacheHints:        publicCache,
		SupportedVersions: res.GetSupportedVersions(),
		Capabilities:      Capabilities{Extensions: extensions},
	}
}

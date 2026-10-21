package model

import pb "github.com/ngo/precedent_mcp/gen/proto/v1"

type Annotations struct {
	Audience []string `json:"audience"`
}

type Resource struct {
	URI         string       `json:"uri"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	MimeType    string       `json:"mimeType"`
	Size        int64        `json:"size,omitempty"`
	Annotations *Annotations `json:"annotations,omitempty"`
}

type ResourcesListResult struct {
	ResultBase
	CacheHints
	Resources []Resource `json:"resources"`
}

func (r *Response) FromResourceListResponse(res *pb.ResourceListResponse) {
	resources := make([]Resource, 0, len(res.GetResources()))
	for _, resource := range res.GetResources() {
		item := Resource{
			URI:         resource.GetUri(),
			Name:        resource.GetName(),
			Description: resource.GetDescription(),
			MimeType:    resource.GetMimeType(),
			Size:        resource.GetSize(),
		}
		if a := resource.GetAnnotations(); a != nil {
			item.Annotations = &Annotations{Audience: a.GetAudience()}
		}
		resources = append(resources, item)
	}
	r.Result = ResourcesListResult{ResultBase: complete, CacheHints: publicCache, Resources: resources}
}

type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type ResourceReadResult struct {
	ResultBase
	CacheHints
	Contents []ResourceContents `json:"contents"`
}

func (r *Response) FromResourceReadResponse(res *pb.ResourceReadResponse) {
	contents := make([]ResourceContents, 0, len(res.GetContents()))
	for _, content := range res.GetContents() {
		contents = append(contents, ResourceContents{URI: content.GetUri(), MimeType: content.GetMimeType(), Text: content.GetText()})
	}
	r.Result = ResourceReadResult{ResultBase: complete, CacheHints: publicCache, Contents: contents}
}

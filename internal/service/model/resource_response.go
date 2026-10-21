package model

import (
	"encoding/json"
	"fmt"
	"net/url"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ResourceListResponse struct {
	*pb.ResourceListResponse
}

func (r *ResourceListResponse) FromAssets(mds []assetmodel.Metadata, names []env.SkillName) {
	r.ResourceListResponse = &pb.ResourceListResponse{Resources: make([]*pb.Resource, 0, len(mds)+len(names))}
	for _, md := range mds {
		r.Resources = append(r.Resources, &pb.Resource{
			Uri:         (&url.URL{Scheme: string(env.SchemeSkill), Host: md.Name}).String(),
			Name:        md.Name,
			Description: md.Description,
			MimeType:    string(env.MimeMarkdown),
			Size:        md.Size,
		})
	}
	for _, name := range names {
		r.Resources = append(r.Resources, &pb.Resource{
			Uri:         (&url.URL{Scheme: string(env.SchemeExamples), Host: string(name)}).String(),
			Name:        string(name),
			Description: fmt.Sprintf(string(env.DescriptionExamples), name),
			MimeType:    string(env.MimeMarkdown),
			Annotations: &pb.Annotations{Audience: []string{string(env.AudienceUser)}},
		})
	}
}

type ResourceReadResponse struct {
	*pb.ResourceReadResponse
}

func (r *ResourceReadResponse) FromText(uri, text string) {
	r.ResourceReadResponse = &pb.ResourceReadResponse{Contents: []*pb.ResourceContents{{Uri: uri, MimeType: string(env.MimeMarkdown), Text: text}}}
}

type ResourceURIData struct {
	URI string `json:"uri"`
}

func ResourceNotFoundError(uri string) error {
	return resourceURIError("resource not found", uri)
}

func InvalidResourceURIError(uri string) error {
	return resourceURIError("invalid resource URI", uri)
}

func UnsupportedResourceSchemeError(uri string) error {
	return resourceURIError("unsupported resource scheme", uri)
}

func MissingResourceNameError(uri string) error {
	return resourceURIError("resource URI has no name", uri)
}

func ResourcePathNotAllowedError(uri string) error {
	return resourceURIError("resource URI must not have a path", uri)
}

func resourceURIError(message, uri string) error {
	data, err := json.Marshal(ResourceURIData{URI: uri})
	if err != nil {
		return err
	}
	st, err := status.New(codes.InvalidArgument, message).WithDetails(&pb.RPCError{
		Code:    int32(env.CodeInvalidParams),
		Message: message,
		Data:    data,
	})
	if err != nil {
		return err
	}
	return st.Err()
}

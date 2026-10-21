package validator

import (
	"context"
	"net/url"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
)

func SkillURI() Step[*model.ResourceURIRequest, *pb.SkillGetResponse] {
	return func(_ context.Context, req *model.ResourceURIRequest) (*pb.SkillGetResponse, error) {
		parsed, err := url.Parse(req.URI)
		if err != nil || !parsed.IsAbs() {
			return nil, model.InvalidResourceURIError(req.URI)
		}
		if parsed.Scheme != string(env.SchemeSkill) {
			return nil, model.UnsupportedResourceSchemeError(req.URI)
		}
		if parsed.Host == "" {
			return nil, model.MissingResourceNameError(req.URI)
		}
		if parsed.Path != "" {
			return nil, model.ResourcePathNotAllowedError(req.URI)
		}
		req.Parsed = parsed
		return nil, nil
	}
}

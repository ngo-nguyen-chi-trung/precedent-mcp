package gateway

import (
	"context"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ngo/precedent_mcp/gateway/model"
	"github.com/ngo/precedent_mcp/gateway/validator"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
	"google.golang.org/grpc"
)

type route func(c *gin.Context, req *model.Request, res *model.Response) *model.RPCError

type Gateway struct {
	pipe   validator.Pipeline
	routes map[env.RPCMethod]route
}

func NewGateway(capabilities pb.CapabilityServiceClient, tools pb.ToolServiceClient, resources pb.ResourceServiceClient, skills pb.SkillServiceClient) *Gateway {
	return &Gateway{
		pipe: validator.NewBuilder().Host().Bind().Legacy().Params().Headers().Build(),
		routes: map[env.RPCMethod]route{
			env.MethodDiscover:      register(capabilities.Discover, (*model.Request).ToEmpty, (*model.Response).FromDiscoverResponse),
			env.MethodToolsList:     register(tools.ListTools, (*model.Request).ToEmpty, (*model.Response).FromToolListResponse),
			env.MethodToolsCall:     register(tools.CallTool, (*model.Request).ToToolCallRequest, (*model.Response).FromToolCallResponse),
			env.MethodResourcesList: register(resources.ListResources, (*model.Request).ToEmpty, (*model.Response).FromResourceListResponse),
			env.MethodResourcesRead: register(resources.ReadResource, (*model.Request).ToResourceReadRequest, (*model.Response).FromResourceReadResponse),
			env.MethodSkillsList:    register(skills.ListSkills, (*model.Request).ToEmpty, (*model.Response).FromSkillListResponse),
			env.MethodSkillsGet:     register(skills.GetSkill, (*model.Request).ToSkillGetRequest, (*model.Response).FromSkillGetResponse),
		},
	}
}

func register[In, Out any](call func(context.Context, In, ...grpc.CallOption) (Out, error), to func(*model.Request) In, from func(*model.Response, Out)) route {
	return func(c *gin.Context, req *model.Request, res *model.Response) *model.RPCError {
		out, err := call(c.Request.Context(), to(req))
		if err != nil {
			return model.FromStatus(err)
		}
		from(res, out)
		return nil
	}
}

func CORSConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowOrigins = env.GetAllowedOrigins()
	config.AllowWildcard = true
	config.AddAllowHeaders(string(env.HeaderProtocolVersion), string(env.HeaderMethod), string(env.HeaderName))
	return config
}

func (g *Gateway) Router() (*gin.Engine, error) {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	r.HandleMethodNotAllowed = true
	r.Use(cors.New(CORSConfig()))
	r.POST(string(env.RouteMCP), g.HandleMCP)
	return r, nil
}

func (g *Gateway) HandleMCP(c *gin.Context) {
	var req model.Request
	res := model.NewResponse()
	rerr := g.pipe.Run(c, &req)
	route, ok := g.routes[req.Method]
	if rerr == nil && !ok {
		rerr = model.MethodNotFound(req.Method)
	}
	if rerr == nil {
		rerr = route(c, &req, res)
	}
	res.SetID(req.ID)
	status := http.StatusOK
	if rerr != nil {
		res.FromError(rerr)
		status = rerr.HTTPStatus
	}
	c.JSON(status, res)
}

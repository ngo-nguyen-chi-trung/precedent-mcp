package main

import (
	"context"
	"log"

	"github.com/ngo/precedent_mcp/internal/asset"
	"github.com/ngo/precedent_mcp/internal/clients"
	"github.com/ngo/precedent_mcp/internal/db"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service"
)

var (
	capabilityService *service.CapabilityService
	toolService       *service.ToolService
	resourceService   *service.ResourceService
	skillService      *service.SkillService
)

func init() {
	if err := env.LoadEnv(); err != nil {
		log.Printf("warn: %v", err)
	}
	ctx := context.Background()
	wv := clients.GetWeaviateInstance()
	toolService = service.NewToolService(asset.NewToolStore(string(env.PathAssets)), db.NewContextRepo(ctx, wv), db.NewDecisionRepo(ctx, wv))
	err := toolService.Register(map[env.ToolName]service.ToolFunc{
		env.ToolAddContext:        service.Tool(toolService.AddContext),
		env.ToolRetrieveContext:   service.Tool(toolService.RetrieveContext),
		env.ToolGetContextThread:  service.Tool(toolService.GetContextThread),
		env.ToolRecordDecision:    service.Tool(toolService.RecordDecision),
		env.ToolRetrieveDecisions: service.Tool(toolService.RetrieveDecisions),
		env.ToolReviewDecision:    service.Tool(toolService.ReviewDecision),
	})
	if err != nil {
		log.Fatalf("register tools: %v", err)
	}
	skillStore := asset.NewSkillStore(string(env.PathAssets), asset.Skills)
	exampleStore := asset.NewExampleStore(string(env.PathAssets), asset.Examples)
	capabilityService = service.NewCapabilityService()
	resourceService = service.NewResourceService(skillStore, exampleStore)
	skillService = service.NewSkillService(skillStore)
}

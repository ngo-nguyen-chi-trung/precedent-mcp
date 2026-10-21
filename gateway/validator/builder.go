package validator

import (
	"github.com/gin-gonic/gin"
	"github.com/ngo/precedent_mcp/gateway/model"
)

type Step func(r *model.Request, c *gin.Context) *model.RPCError

type Pipeline []Step

func (p Pipeline) Run(c *gin.Context, r *model.Request) *model.RPCError {
	for _, step := range p {
		if rerr := step(r, c); rerr != nil {
			return rerr
		}
	}
	return nil
}

type Builder struct {
	steps Pipeline
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Host() *Builder {
	b.steps = append(b.steps, (*model.Request).BindHost)
	return b
}

func (b *Builder) Bind() *Builder {
	b.steps = append(b.steps, (*model.Request).Bind)
	return b
}

func (b *Builder) Legacy() *Builder {
	b.steps = append(b.steps, (*model.Request).BindLegacy)
	return b
}

func (b *Builder) Params() *Builder {
	b.steps = append(b.steps, (*model.Request).BindParams)
	return b
}

func (b *Builder) Headers() *Builder {
	b.steps = append(b.steps, (*model.Request).BindHeaders)
	return b
}

func (b *Builder) Build() Pipeline {
	return b.steps
}

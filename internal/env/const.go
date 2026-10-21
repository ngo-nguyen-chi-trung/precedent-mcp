package env

import "time"

type OriginList string

const (
	OriginDefault   OriginList = "http://localhost:*,http://127.0.0.1:*,http://[::1]:*"
	OriginSeparator OriginList = ","
)

type HostName string

const (
	HostLoopback HostName = "localhost"
)

type SkillName string

const (
	SkillPrecedentCapture SkillName = "precedent-capture"
	SkillPrecedentDecide  SkillName = "precedent-decide"
	SkillPrecedentRecall  SkillName = "precedent-recall"
	SkillPrecedentRefine  SkillName = "precedent-refine"
	SkillPrecedentReview  SkillName = "precedent-review"
)

type AssetPath string

const (
	PathAssets                   AssetPath = "assets"
	PathToolList                 AssetPath = "tools.yaml"
	PathPrecedentCaptureSkill    AssetPath = "skills/precedent-capture/SKILL.md"
	PathPrecedentCaptureMetadata AssetPath = "skills/precedent-capture/metadata.yaml"
	PathPrecedentCaptureExamples AssetPath = "skills/precedent-capture/examples.md"
	PathPrecedentDecideSkill     AssetPath = "skills/precedent-decide/SKILL.md"
	PathPrecedentDecideMetadata  AssetPath = "skills/precedent-decide/metadata.yaml"
	PathPrecedentDecideExamples  AssetPath = "skills/precedent-decide/examples.md"
	PathPrecedentRecallSkill     AssetPath = "skills/precedent-recall/SKILL.md"
	PathPrecedentRecallMetadata  AssetPath = "skills/precedent-recall/metadata.yaml"
	PathPrecedentRecallExamples  AssetPath = "skills/precedent-recall/examples.md"
	PathPrecedentRefineSkill     AssetPath = "skills/precedent-refine/SKILL.md"
	PathPrecedentRefineMetadata  AssetPath = "skills/precedent-refine/metadata.yaml"
	PathPrecedentRefineExamples  AssetPath = "skills/precedent-refine/examples.md"
	PathPrecedentReviewSkill     AssetPath = "skills/precedent-review/SKILL.md"
	PathPrecedentReviewMetadata  AssetPath = "skills/precedent-review/metadata.yaml"
	PathPrecedentReviewExamples  AssetPath = "skills/precedent-review/examples.md"
)

type URIScheme string

const (
	SchemeSkill    URIScheme = "skill"
	SchemeExamples URIScheme = "examples"
)

type Audience string

const (
	AudienceUser Audience = "user"
)

type DescriptionTemplate string

const (
	DescriptionExamples DescriptionTemplate = "Examples for the %s skill. Reference only; write examples that match the agent's task context under ## Examples (at most 15 lines) in SKILL.md."
)

type Extension string

const (
	ExtensionSkills Extension = "io.modelcontextprotocol/skills"
)

type Limit int

const (
	LimitSkillDescription Limit = 50
	LimitContextEntries   Limit = 10
	LimitTopN             Limit = 20
)

type CacheSize int

const (
	CacheSizeTools  CacheSize = 1
	CacheSizeSkills CacheSize = 10
)

const (
	DurationCache           time.Duration = 5 * time.Minute
	DurationWeaviateStartup time.Duration = 30 * time.Second
)

type RankParameter float64

const (
	RankReciprocalConstant RankParameter = 60
	RankNeutralReview      RankParameter = 3
	RankHybridAlpha        RankParameter = 0.5
)

type RankWeight float64

const (
	WeightReview RankWeight = 3
	WeightMatch  RankWeight = 2
)

type Version string

const (
	VersionJSONRPC  Version = "2.0"
	VersionProtocol Version = "2026-07-28"
)

type ServerInfo string

const (
	ServerInfoName    ServerInfo = "precedent-mcp"
	ServerInfoVersion ServerInfo = "1.0.0"
)

type RoutePath string

const (
	RouteMCP RoutePath = "/mcp"
)

type ContentType string

const (
	ContentText ContentType = "text"
)

type MimeType string

const (
	MimeMarkdown MimeType = "text/markdown"
)

type HeaderKey string

const (
	HeaderProtocolVersion HeaderKey = "MCP-Protocol-Version"
	HeaderMethod          HeaderKey = "Mcp-Method"
	HeaderName            HeaderKey = "Mcp-Name"
)

type RPCMethod string

const (
	MethodInitialize    RPCMethod = "initialize"
	MethodDiscover      RPCMethod = "server/discover"
	MethodToolsList     RPCMethod = "tools/list"
	MethodToolsCall     RPCMethod = "tools/call"
	MethodSkillsList    RPCMethod = "skills/list"
	MethodSkillsGet     RPCMethod = "skills/get"
	MethodResourcesList RPCMethod = "resources/list"
	MethodResourcesRead RPCMethod = "resources/read"
)

type ToolName string

const (
	ToolAddContext        ToolName = "add_context"
	ToolRetrieveContext   ToolName = "retrieve_context"
	ToolGetContextThread  ToolName = "get_context_thread"
	ToolRecordDecision    ToolName = "record_decision"
	ToolRetrieveDecisions ToolName = "retrieve_decisions"
	ToolReviewDecision    ToolName = "review_decision"
)

type DecisionStatus string

const (
	StatusProposed   DecisionStatus = "proposed"
	StatusAccepted   DecisionStatus = "accepted"
	StatusSuperseded DecisionStatus = "superseded"
)

type WeaviateClassName string

const (
	ClassContext  WeaviateClassName = "Context"
	ClassDecision WeaviateClassName = "Decision"
)

type ResultType string

const (
	ResultComplete ResultType = "complete"
)

type CacheScope string

const (
	CachePublic CacheScope = "public"
)

type ServiceErrorCode string

const (
	ServiceErrorInvalidArgument  ServiceErrorCode = "invalid_argument"
	ServiceErrorInternal         ServiceErrorCode = "internal"
	ServiceErrorDecisionNotFound ServiceErrorCode = "decision_not_found"
)

type AssetErrorCode string

const (
	AssetErrorSkillNotFound     AssetErrorCode = "skill_not_found"
	AssetErrorExamplesNotFound  AssetErrorCode = "examples_not_found"
	AssetErrorInvalidMetadata   AssetErrorCode = "invalid_metadata"
	AssetErrorSkillNameMismatch AssetErrorCode = "skill_name_mismatch"
	AssetErrorDescriptionLength AssetErrorCode = "description_length"
)

type RPCCode int

const (
	CodeParseError         RPCCode = -32700
	CodeInvalidRequest     RPCCode = -32600
	CodeMethodNotFound     RPCCode = -32601
	CodeInvalidParams      RPCCode = -32602
	CodeInternal           RPCCode = -32603
	CodeHeaderMismatch     RPCCode = -32020
	CodeUnsupportedVersion RPCCode = -32022
)

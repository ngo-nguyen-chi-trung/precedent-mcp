package env

import (
	"cmp"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type EnvKey string

const (
	WeaviateHost          EnvKey = "WEAVIATE_HOST"
	WeaviateScheme        EnvKey = "WEAVIATE_SCHEME"
	GRPCPort              EnvKey = "GRPC_PORT"
	GRPCAddress           EnvKey = "GRPC_ADDRESS"
	HTTPPort              EnvKey = "HTTP_PORT"
	AllowedOrigins        EnvKey = "ALLOWED_ORIGINS"
	WeaviateText2VecModel EnvKey = "WEAVIATE_TEXT2VEC_MODEL"
)

func (key EnvKey) GetValue() string {
	return os.Getenv(string(key))
}

func GetAllowedOrigins() []string {
	return strings.Split(cmp.Or(AllowedOrigins.GetValue(), string(OriginDefault)), string(OriginSeparator))
}

func LoadEnv() error {
	return godotenv.Load(".env")
}

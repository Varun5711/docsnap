package config

import "os"

type Config struct {
	Addr          string
	AppOrigin     string
	FlareMode     string
	FlareRPCURL   string
	ContractAddr  string
	OpenAIAPIKey  string
	StorageMode   string
	StoragePath   string
	DatabaseURL   string
}

func Load() Config {
	return Config{
		Addr:         env("DOCSNAP_API_ADDR", ":8080"),
		AppOrigin:    env("DOCSNAP_APP_ORIGIN", "http://localhost:3000"),
		FlareMode:    env("DOCSNAP_FLARE_MODE", "simulated"),
		FlareRPCURL:  env("DOCSNAP_FLARE_RPC_URL", "https://coston2-api.flare.network/ext/C/rpc"),
		ContractAddr: env("DOCSNAP_CONTRACT_ADDRESS", ""),
		OpenAIAPIKey: env("DOCSNAP_OPENAI_API_KEY", ""),
		StorageMode:  env("DOCSNAP_STORAGE_MODE", "local"),
		StoragePath:  env("DOCSNAP_STORAGE_PATH", "./tmp/evidence"),
		DatabaseURL:  env("DATABASE_URL", ""),
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}


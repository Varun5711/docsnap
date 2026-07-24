package config

import "os"

type Config struct {
	Addr         string
	AppOrigin    string
	FlareMode    string
	FlareRPCURL  string
	ContractAddr string
	FlareKey     string
	TEEKey       string
	GroqAPIKey   string
	GroqBaseURL  string
	GroqModel    string
	TEEURL       string
	StorageMode  string
	StoragePath  string
	S3Endpoint   string
	S3AccessKey  string
	S3SecretKey  string
	S3Bucket     string
	S3UseSSL     bool
	DatabaseURL  string
}

func Load() Config {
	return Config{
		Addr:         env("DOCSNAP_API_ADDR", ":8080"),
		AppOrigin:    env("DOCSNAP_APP_ORIGIN", "http://localhost:3000"),
		FlareMode:    env("DOCSNAP_FLARE_MODE", "simulated"),
		FlareRPCURL:  env("DOCSNAP_FLARE_RPC_URL", "https://coston2-api.flare.network/ext/C/rpc"),
		ContractAddr: env("DOCSNAP_CONTRACT_ADDRESS", ""),
		FlareKey:     env("DOCSNAP_FLARE_PRIVATE_KEY", ""),
		TEEKey:       env("DOCSNAP_TEE_REPORTER_PRIVATE_KEY", ""),
		GroqAPIKey:   env("GROQ_API_KEY", ""),
		GroqBaseURL:  env("GROQ_BASE_URL", "https://api.groq.com/openai/v1"),
		GroqModel:    env("GROQ_MODEL", "qwen/qwen3.6-27b"),
		TEEURL:       env("DOCSNAP_TEE_URL", ""),
		StorageMode:  env("DOCSNAP_STORAGE_MODE", "local"),
		StoragePath:  env("DOCSNAP_STORAGE_PATH", "./tmp/evidence"),
		S3Endpoint:   env("DOCSNAP_S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:  env("DOCSNAP_S3_ACCESS_KEY", "docsnap"),
		S3SecretKey:  env("DOCSNAP_S3_SECRET_KEY", "docsnap-secret"),
		S3Bucket:     env("DOCSNAP_S3_BUCKET", "docsnap-evidence"),
		S3UseSSL:     env("DOCSNAP_S3_USE_SSL", "false") == "true",
		DatabaseURL:  env("DATABASE_URL", "postgres://docsnap:docsnap@localhost:5432/docsnap?sslmode=disable"),
	}
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

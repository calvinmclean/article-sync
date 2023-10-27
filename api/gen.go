package api

//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=api -generate=client,types -o ./forem.gen.go https://raw.githubusercontent.com/forem/forem-docs/main/api_v1.json

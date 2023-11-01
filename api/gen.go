package api

// v0 API did not seem to work and v1 API has `tags` as a `string` instead of `[]string` so I had to download and modify
//go:generate go run -mod=mod github.com/deepmap/oapi-codegen/cmd/oapi-codegen --package=api -generate=client,types -o ./forem.gen.go ./api.json

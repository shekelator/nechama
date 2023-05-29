module github.com/shekelator/nechama

go 1.20

require github.com/hebcal/hdate v1.0.2

require (
	github.com/hebcal/greg v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/shekelator/nechama/internal/sefariawrap v0.0.0-00010101000000-000000000000 // indirect
	github.com/spf13/cobra v1.7.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1 // indirect
)

replace github.com/shekelator/nechama/internal/sefariawrap => ./internal/sefariawrap

module github.com/DataDog/appsec-internal-go

go 1.23.0

require (
	github.com/stretchr/testify v1.10.0
	go.uber.org/goleak v1.3.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

retract v1.10.0 // This version includes unintended breaking changes.

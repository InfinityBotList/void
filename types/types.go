// Package types provides a service struct
package types

type Service struct {
	// Name is the name of the service
	Name string `yaml:"name" validate:"required"`
	// Domain is the domain of the service
	Domain string `yaml:"domain" validate:"required,hostname"`
	// Support is the support server/location of the service
	Support string `yaml:"support" validate:"required"`
	// Status is the status page of the service
	Status string `yaml:"status" validate:"required,url"`
}

// The yaml document
type Document struct {
	Services []Service `yaml:"services"`
	APIUrls  []string  `yaml:"apiUrls"`
}

// Some information about the maintenance server
type VoidInfo struct {
	Version string
	Commit  string
}

type HTMLCtx struct {
	MatchedService Service
	Path           string
	Hostname       string
	Info           VoidInfo
	Redirect       string
}

type APICtx struct {
	Message string   `json:"message"`
	Service Service  `json:"service"`
	Info    VoidInfo `json:"info"`
}

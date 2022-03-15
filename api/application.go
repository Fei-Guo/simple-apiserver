package api

import "time"

type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type Application struct {
	Title       string       `yaml:"title"`
	Version     string       `yaml:"version"`
	Maintainers []Maintainer `yaml:"maintainers,omitempty"`
	Company     string       `yaml:"company,omitempty"`
	Website     string       `yaml:"website,omitempty"`
	Source      string       `yaml:"source,omitempty"`
	License     string       `yaml:"license,omitempty"`
	Description string       `yaml:"description,omitempty"`

	// metadata
	CreateTimeStamp time.Time  `yaml:"createTimeStamp,omitempty"`
	DeleteTimeStamp *time.Time `yaml:"deleteTimeStamp,omitempty"`
	ResourceVersion string     `yaml:"resourceVersion,omitempty"`
}

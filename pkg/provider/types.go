package provider

import (
	"context"
)

// Resource represents a generic AWS entity instance.
type Resource struct {
	ID       string            // Unique identifier (e.g., ARN or InstanceID)
	Name     string            // Friendly name
	Status   string            // State (e.g., running, terminated, active)
	Metadata map[string]string // Key-value pairs for additional quick details
	Raw      interface{}       // Raw AWS SDK struct for custom operations
}

// ColumnDefinition defines how to extract and display table columns.
type ColumnDefinition struct {
	Header    string
	Width     int // 0 for flexible sizing
	ValueFunc func(r Resource) string
}

// CustomAction defines custom hotkeys/commands for a resource (e.g., "ssh", "download").
type CustomAction struct {
	Name        string
	Description string
	Hotkey      string // Key combination (e.g., 's', 'ctrl-d')
	Type        string // "text" (returns text content), "window" (opens UI window), "command" (standard callback)
	ActionFunc  func(ctx context.Context, r Resource) (string, error)
}

// ResourceProvider is the core interface every AWS entity handler must implement.
type ResourceProvider interface {
	// Metadata
	GetResourceType() string // e.g. "EC2 Instances"
	GetShortNames() []string // e.g. ["ec2", "instance"]
	GetCategory() string     // e.g. "Compute"

	// Data Fetching
	List(ctx context.Context, filters map[string]string) ([]Resource, error)
	Describe(ctx context.Context, id string) (string, error) // Details as a formatted string (YAML/JSON)
	Delete(ctx context.Context, id string) error

	// UI Layout & Action mappings
	GetColumns() []ColumnDefinition
	GetCustomActions() []CustomAction
}

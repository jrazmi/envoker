package bridgegen

// Config holds configuration for bridge generation
type Config struct {
	ModulePath     string
	OutputDir      string
	ForceOverwrite bool
}

// TemplateData holds all data needed for bridge template rendering
type TemplateData struct {
	// Package naming
	PackageName    string // e.g., "tasksrepobridge"
	RepoPackage    string // e.g., "tasksrepo"

	// Entity naming
	Entity           string // e.g., "Task" (alias for EntityName)
	EntityName       string // e.g., "Task"
	EntityNamePlural string // e.g., "Tasks"
	EntityNameLower  string // e.g., "task"
	EntityNameCamel  string // e.g., "task" (for JSON)

	// HTTP paths
	HTTPBasePath string // e.g., "/tasks"
	HTTPSingular string // e.g., "/task"

	// Primary key
	PKColumn     string // e.g., "task_id"
	PKGoName     string // e.g., "TaskID"
	PKJSONName   string // e.g., "taskId"
	PKGoType     string // e.g., "string"
	PKParamName  string // e.g., "taskID"
	PKURLParam   string // e.g., "task_id"

	// Fields for bridge model
	EntityFields []BridgeField
	CreateFields []BridgeField
	UpdateFields []BridgeField
	FilterFields []BridgeField

	// Foreign key methods
	ForeignKeys []FKBridgeMethod

	// Module path
	ModulePath string

	// Schema features
	BridgePackage   string // e.g., "tasksrepobridge"
	HasStatusColumn bool   // True if table has a status column for archive support
}

// BridgeField represents a field in the bridge model
type BridgeField struct {
	RepoName    string // Repository field name (PascalCase)
	BridgeName  string // Bridge field name (PascalCase)
	JSONName    string // JSON field name (camelCase)
	GoType      string // Go type
	DBColumn    string // Database column name
	IsPointer   bool   // Whether it's a pointer
	OmitEmpty   bool   // Whether to use omitempty in JSON
	IsTime      bool   // Whether it's a time field
	IsJSON      bool   // Whether it's a JSON field
}

// FKBridgeMethod represents a foreign key method in the bridge
type FKBridgeMethod struct {
	MethodName     string // e.g., "httpListByTaskID"
	RoutePath      string // e.g., "/{task_id}/executions"
	FKColumn       string // e.g., "task_id"
	FKGoName       string // e.g., "TaskID"
	FKParamName    string // e.g., "taskID"
	FKURLParam     string // e.g., "task_id"
	FKGoType       string // e.g., "string"
	RefEntityName  string // e.g., "Task"
}

// GenerateResult holds the results of bridge generation
type GenerateResult struct {
	BridgeFile      string   // Path to generated bridge.go
	HTTPRoutesFile  string   // Path to generated http.go (never overwritten)
	HTTPFile        string   // Path to generated http_gen.go
	ModelFile       string   // Path to generated model_gen.go
	MarshalFile     string   // Path to generated marshal_gen.go
	FOPFile         string   // Path to generated fop_gen.go
	Errors          []error  // Any errors encountered
	Warnings        []string // Any warnings
}

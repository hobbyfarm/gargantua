package property

type DataType string

var (
	DataTypeString  DataType = "string"
	DataTypeInteger DataType = "integer"
	DataTypeFloat   DataType = "float"
	DataTypeBoolean DataType = "boolean"
)

type ValueType string

var (
	ValueTypeScalar ValueType = "scalar"
	ValueTypeArray  ValueType = "array"
	ValueTypeMap    ValueType = "map"
)

// +k8s:deepcopy-gen=true

type Property struct {
	Name string `json:"name"`

	DataType  DataType  `json:"dataType"`
	ValueType ValueType `json:"variableType"`

	SettingValidation
}

// +k8s:deepcopy-gen=true

type SettingValidation struct {
	Required    bool
	Maximum     *float64
	Minimum     *float64
	MaxLength   *int64
	MinLength   *int64
	Format      *string
	Pattern     *string
	Enum        []string
	Default     *string
	UniqueItems bool
}

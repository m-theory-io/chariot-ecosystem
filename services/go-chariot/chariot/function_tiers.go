package chariot

import "fmt"

// FunctionTier defines different levels of Chariot functionality
type FunctionTier struct {
	Name        string
	Description string
	Functions   []string
}

// GetBasicChariotFunctions returns the basic Chariot function set
func GetBasicChariotFunctions() []string {
	return []string{
		// Core language functions (from chariot_test.go)
		"declare", "declareGlobal", "setq",
		"if", "else", "while", "switch", "case", "default", "break", "continue",
		"call", "func",

		// Math functions - arithmetic
		"abs", "add", "div", "mod", "mul", "sub",
		// Advanced math
		"ceiling", "ceil", "cos", "exp", "floor", "int", "pow", "round", "sin", "sqrt", "tan",
		// Logarithmic functions
		"log", "log10", "log2",
		// Math constants
		"pi", "e",
		// Statistics
		"avg", "max", "min", "sum",
		// Financial functions
		"amortize", "apr", "ballon", "depreciation", "fv", "irr", "nper", "npv", "pct", "pmt", "pv", "rate",

		// Random functions
		"random", "randomSeed", "randomString",
		"sum", "avg",

		// Logical and comparison functions
		"and", "or", "not", "bigger", "smaller", "biggerEq", "smallerEq", "equal", "unequal",

		// Array functions
		"addTo", "array", "lastIndex", "removeAt", "reverse", "slice", // Keep these for compatibility with existing array_funcs.go

		// Polymorphic functions (from dispatchers.go)
		"getAt", "setAt", "indexOf", "length", "contains", "split", "join",

		// String functions - clean names
		"append", "ascii", "atPos", "char", "charAt", "concat", "digits", "format", "hasPrefix", "hasSuffix", "interpolate", "join", "lastPos", "lower", "occurs", "padLeft", "padRight", "repeat", "replace", "right", "split", "sprintf", "string", "strlen", "substr", "substring", "trim", "trimLeft", "trimRight", "upper",

		// Type utilities
		"typeof", "isNull", "isNumber", "isString", "isBool", "isArray",

		// Date/time functions (basic set)
		"now", "dateFormat", "dateAdd", "dateDiff",
		"year", "month", "day", "hour", "minute", "second",

		// Basic Node support (from node_funcs.go)
		"create", "jsonNode", "mapNode",
		"addChild", "firstChild", "lastChild", "getName",
		"hasAttribute", "getAttribute", "setAttribute", "setAttributes",
		"getChildAt", "getChildByName", "childCount",
		"clear", "list", "nodeToString",

		// Node property access (from dispatchers.go)
		"getProp", "setProp", "getMeta", "setMeta", "getAllMeta",

		// Variable and scope management
		"get", "set", "exists", "delete",

		// I/O functions - clear distinction from math
		"print", "warn", "error", "debug", // Use print instead of log to avoid math conflict

		// Tree serialization
		"treeSave", "treeLoad",
	}
}

// GetDataAnalystChariotFunctions returns the data analyst function set
func GetDataAnalystChariotFunctions() []string {
	basic := GetBasicChariotFunctions()

	dataAnalystExtensions := []string{
		// SQL functions (sql_funcs.go)
		"sqlConnect", "sqlQuery", "sqlExecute", "sqlClose",
		"sqlTransaction", "sqlCommit", "sqlRollback",
		"sqlPrepare", "sqlBind", "sqlFetch",

		// Couchbase functions (couchbase_funcs.go)
		"cbConnect", "cbGet", "cbSet", "cbDelete", "cbQuery",
		"cbBucket", "cbCollection", "cbCluster",
		"cbN1QL", "cbFTS", "cbAnalytics",

		// ETL functions (etl_funcs.go)
		"extract", "transform", "load", "pipeline",
		"dataClean", "dataValidate", "dataAggregate",
		"dataJoin", "dataFilter", "dataSort", "dataGroup",

		// File functions (file_funcs.go)
		"fileRead", "fileWrite", "fileExists", "fileDelete",
		"fileList", "fileInfo", "fileCopy", "fileMove",
		"pathJoin", "pathDir", "pathBase", "pathExt",

		// CSV Node support
		"csvLoad", "csvSave", "csvParse", "csvFormat",
		"csvHeaders", "csvRows", "csvColumns",

		// YAML Node support
		"yamlLoad", "yamlSave", "yamlParse", "yamlFormat",
		"yamlMerge", "yamlExtract",

		// Advanced data structures
		"map", "mapKeys", "mapValues", "mapMerge", "mapFilter",
		"set", "setAdd", "setRemove", "setContains", "setUnion", "setIntersection",

		// Data analysis utilities
		"count", "sum", "avg", "median", "mode", "stddev", "variance",
		"distinct", "frequency", "percentile", "quartile",
	}

	return append(basic, dataAnalystExtensions...)
}

// GetScientificChariotFunctions returns the scientific computing function set
func GetScientificChariotFunctions() []string {
	dataAnalyst := GetDataAnalystChariotFunctions()

	scientificExtensions := []string{
		// Advanced mathematical functions
		"sin", "cos", "tan", "asin", "acos", "atan", "atan2",
		"sinh", "cosh", "tanh", "asinh", "acosh", "atanh",
		"exp", "exp2", "expm1", "ln", "log2", "log1p",
		"gamma", "lgamma", "factorial", "binomial",

		// Statistical functions
		"normalDist", "poissonDist", "binomialDist", "exponentialDist",
		"chiSquare", "tTest", "fTest", "correlation", "regression",
		"histogram", "zscore", "pvalue", "confidence",

		// Matrix operations
		"matrix", "matrixMul", "matrixAdd", "matrixSub", "matrixTranspose",
		"matrixInverse", "matrixDeterminant", "matrixEigenvalues",
		"matrixSVD", "matrixLU", "matrixQR",

		// Vector operations
		"vector", "vectorAdd", "vectorSub", "vectorMul", "vectorDot", "vectorCross",
		"vectorNorm", "vectorUnit", "vectorAngle", "vectorProject",

		// Numerical methods
		"integrate", "differentiate", "solve", "optimize", "interpolate",
		"fft", "ifft", "convolve", "filter", "smooth",

		// Complex numbers
		"complex", "real", "imag", "conjugate", "magnitude", "phase",
		"complexAdd", "complexMul", "complexDiv", "complexPow",

		// Random number generation
		"random", "randomSeed", "randomNormal", "randomUniform",
		"randomPoisson", "randomBinomial", "randomChoice", "shuffle",

		// Scientific constants
		"pi", "e", "euler", "golden", "lightSpeed", "planck", "avogadro",
	}

	return append(dataAnalyst, scientificExtensions...)
}

// GetAllFunctionTiers returns all defined function tiers
func GetAllFunctionTiers() []FunctionTier {
	return []FunctionTier{
		{
			Name:        "basic",
			Description: "Basic Chariot - Core language, math, arrays, strings, basic nodes",
			Functions:   GetBasicChariotFunctions(),
		},
		{
			Name:        "data-analyst",
			Description: "Data Analyst Chariot - Includes SQL, Couchbase, ETL, file operations",
			Functions:   GetDataAnalystChariotFunctions(),
		},
		{
			Name:        "scientific",
			Description: "Scientific Chariot - Advanced math, statistics, matrix operations",
			Functions:   GetScientificChariotFunctions(),
		},
	}
}

// GetFunctionTier returns a specific function tier by name
func GetFunctionTier(tierName string) *FunctionTier {
	tiers := GetAllFunctionTiers()
	for _, tier := range tiers {
		if tier.Name == tierName {
			return &tier
		}
	}
	return nil
}

// ValidateFunctionTier checks if all functions in a tier exist in the master registry
func ValidateFunctionTier(tierName string) ([]string, error) {
	tier := GetFunctionTier(tierName)
	if tier == nil {
		return nil, fmt.Errorf("function tier not found: %s", tierName)
	}

	var missing []string
	for _, funcName := range tier.Functions {
		if !HasFunction(funcName) {
			missing = append(missing, funcName)
		}
	}

	return missing, nil
}

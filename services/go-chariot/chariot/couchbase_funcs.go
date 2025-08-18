package chariot

import (
	"fmt"
	"strconv"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// RegisterCouchbaseFunctions registers all Couchbase-related functions
func RegisterCouchbaseFunctions(rt *Runtime) {
	// Connection management
	rt.Register("cbConnect", func(args ...Value) (Value, error) {
		if len(args) != 4 {
			return nil, fmt.Errorf("cbConnect requires 4 arguments: nodeName, connectionString, username, password")
		}

		nodeName := string(args[0].(Str))
		if nodeName == "" {
			nodeName = "default-couchbase-node"
		}
		connStr := string(args[1].(Str))
		username := string(args[2].(Str))
		password := string(args[3].(Str))

		// Create Couchbase node
		cbNode := NewCouchbaseNode(nodeName)

		// Connect to cluster
		if err := cbNode.Connect(connStr, username, password); err != nil {
			return nil, fmt.Errorf("failed to connect: %v", err)
		}
		// cbNode.SetDiag(true)

		// Store in runtime
		rt.objects[nodeName] = cbNode

		return Str("Connected to Couchbase cluster"), nil
	})

	// Bucket operations
	rt.Register("cbOpenBucket", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("cbOpenBucket requires 2 arguments: nodeName, bucketName")
		}

		nodeName := string(args[0].(Str))
		bucketName := string(args[1].(Str))

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		if err := cbNode.OpenBucket(bucketName); err != nil {
			return nil, fmt.Errorf("failed to open bucket: %v", err)
		}

		return Str(fmt.Sprintf("Opened bucket: %s", bucketName)), nil
	})

	// Scope/Collection operations
	rt.Register("cbSetScope", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("cbSetScope requires 3 arguments: nodeName, scopeName, collectionName")
		}

		nodeName := string(args[0].(Str))
		scopeName := string(args[1].(Str))
		collectionName := string(args[2].(Str))

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		if err := cbNode.SetScope(scopeName, collectionName); err != nil {
			return nil, fmt.Errorf("failed to set scope: %v", err)
		}

		return Str(fmt.Sprintf("Set scope: %s.%s", scopeName, collectionName)), nil
	})

	// N1QL Query
	rt.Register("cbQuery", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("cbQuery requires at least 2 arguments: nodeName, query, [params...]")
		}

		nodeName := string(args[0].(Str))
		query := string(args[1].(Str))

		// Auto-interpolate the query string
		queryStr, err := interpolateString(rt, query)
		if err != nil {
			return nil, fmt.Errorf("query interpolation failed: %v", err)
		}

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		// Parse parameters (if provided as JSONNode)
		var params map[string]interface{}
		if len(args) > 2 {
			switch p := args[2].(type) {
			case *JSONNode:
				if paramMap, ok := p.GetJSONValue().(map[string]interface{}); ok {
					params = paramMap
				}
			case *SimpleJSON:
				if paramMap, ok := p.value.(map[string]interface{}); ok {
					params = paramMap
				}
			case *MapNode:
				params = p.ToMap()
			case *MapValue:
				params = make(map[string]interface{})
				for k, v := range p.Values {
					params[k] = convertValueToNative(v)
				}
			case *ArrayValue:
				// Convert ArrayValue to slice of interfaces
				params = make(map[string]interface{})
				for i := 0; i < p.Length(); i++ {
					value := p.Get(i)
					// Use index as key for simplicity
					params[fmt.Sprintf("param%d", i)] = convertValueToNative(value)
				}
			default:
				// If no parameters provided, use empty map
				params = make(map[string]interface{})
			}
		}

		if err := cbNode.Query(queryStr, params); err != nil {
			return nil, fmt.Errorf("query failed: %v", err)
		}

		return &SimpleJSON{value: cbNode.QueryResults}, nil
	})

	rt.Register("cbInsert", func(args ...Value) (Value, error) {
		if len(args) < 3 || len(args) > 4 {
			return nil, fmt.Errorf("cbInsert requires 3-4 arguments: nodeName, documentId, document, [expiryDuration]")
		}

		nodeName := string(args[0].(Str))
		docId := string(args[1].(Str))
		document := args[2]

		// Optional expiry parameter (default: no expiry)
		var expiryDuration time.Duration
		if len(args) > 3 {
			switch exp := args[3].(type) {
			case Number:
				// If a number is provided, assume it's seconds
				expiryDuration = time.Duration(float64(exp)) * time.Second
			case Str:
				// Parse duration string like "24h", "7d", "30m", etc.
				var err error
				expiryDuration, err = time.ParseDuration(string(exp))
				if err != nil {
					return nil, fmt.Errorf("invalid expiry duration: %v", err)
				}
			default:
				return nil, fmt.Errorf("expiry must be a number (seconds) or a duration string")
			}
		}

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		// Stricter type checking
		var doc interface{}
		var originalDoc Value // Keep track of original document structure

		switch d := document.(type) {
		case *SimpleJSON:
			// Most efficient path - use native value directly
			doc = d.value
			originalDoc = d

		case *MapValue:
			// Convert MapValue to native map
			nativeMap := make(map[string]interface{})
			for k, v := range d.Values {
				nativeMap[k] = convertValueToNative(v)
			}
			doc = nativeMap
			originalDoc = d

		case *MapNode:
			// Convert MapNode to native map
			nativeMap := make(map[string]interface{})
			for k, v := range d.ToMap() {
				nativeMap[k] = convertValueToNative(v)
			}
			doc = nativeMap
			originalDoc = d

		case *JSONNode:
			// Allow JSONNode but return the ORIGINAL node with metadata
			doc = d.GetJSONValue()
			originalDoc = d // Keep the original JSONNode

		default:
			return nil, fmt.Errorf("unsupported document type: %T - use SimpleJSON, MapValue, or JSONNode", document)
		}

		// Insert document
		docMeta, err := cbNode.Insert(docId, doc, expiryDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to insert document: %v", err)
		}

		// Return document WITH metadata
		var result Value

		// Create appropriate return value based on input type
		switch d := originalDoc.(type) {
		case *SimpleJSON:
			result = d
		case *MapValue:
			// Create SimpleJSON wrapping MapValue data
			result = &SimpleJSON{value: doc}
		case *JSONNode:
			// Return the original JSONNode
			result = d // Simply use the original node
		}

		// Apply metadata to whatever result type we have
		setDocumentMeta(result, uint64(docMeta.Cas()), expiryDuration)

		return result, nil
	})

	rt.Register("cbUpsert", func(args ...Value) (Value, error) {
		if len(args) < 3 || len(args) > 4 {
			return nil, fmt.Errorf("cbUpsert requires 3-4 arguments: nodeName, documentId, document, [expiryDuration]")
		}

		nodeName := string(args[0].(Str))
		docId := string(args[1].(Str))
		document := args[2]

		// Optional expiry parameter (default: no expiry)
		var expiryDuration time.Duration
		if len(args) == 4 {
			switch exp := args[3].(type) {
			case Number:
				// If a number is provided, assume it's seconds
				expiryDuration = time.Duration(float64(exp)) * time.Second
			case Str:
				// Parse duration string like "24h", "7d", "30m", etc.
				var err error
				expiryDuration, err = time.ParseDuration(string(exp))
				if err != nil {
					return nil, fmt.Errorf("invalid expiry duration: %v", err)
				}
			default:
				return nil, fmt.Errorf("expiry must be a number (seconds) or a duration string")
			}
		}

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		// Stricter type checking
		var doc interface{}
		var originalDoc Value // Keep track of original document structure

		switch d := document.(type) {
		case *SimpleJSON:
			// Most efficient path - use native value directly
			doc = d.value
			originalDoc = d

		case *MapValue:
			// Convert MapValue to native map
			nativeMap := make(map[string]interface{})
			for k, v := range d.Values {
				nativeMap[k] = convertValueToNative(v)
			}
			doc = nativeMap
			originalDoc = d

		case *JSONNode:
			// Allow JSONNode but return the ORIGINAL node with metadata
			doc = d.GetJSONValue()
			originalDoc = d // Keep the original JSONNode

		default:
			return nil, fmt.Errorf("unsupported document type: %T - use SimpleJSON, MapValue, or JSONNode", document)
		}

		// Upsert document
		docMeta, err := cbNode.Upsert(docId, doc, expiryDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert document: %v", err)
		}

		// Return document WITH metadata
		var result Value

		// Create appropriate return value based on input type
		switch d := originalDoc.(type) {
		case *SimpleJSON:
			result = d
		case *MapValue:
			// Create SimpleJSON wrapping MapValue data
			result = &SimpleJSON{value: doc}
		case *JSONNode:
			// Return SimpleJSON
			result = d
		}

		// Apply metadata to whatever result type we have
		setDocumentMeta(result, uint64(docMeta.Cas()), expiryDuration)

		return result, nil
	})

	rt.Register("cbGet", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("cbGet requires 2 arguments: nodeName, documentId")
		}

		nodeName := string(args[0].(Str))
		docId := string(args[1].(Str))

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		docRes, err := cbNode.Collection.Get(docId, nil) // ← Use Collection.Get directly
		if err != nil {
			return nil, fmt.Errorf("failed to get document: %v", err)
		}

		var document interface{}
		if err = docRes.Content(&document); err != nil {
			return nil, fmt.Errorf("failed to decode document content: %v", err)
		}

		// Create SimpleJSON with metadata directly
		result := &SimpleJSON{value: document}
		result.SetMeta("cas", Str(strconv.FormatUint(uint64(docRes.Cas()), 10)))

		// Handle expiry
		if expiry := docRes.Expiry(); expiry != nil {
			result.SetMeta("expiry", Number(expiry.Seconds()))
		} else {
			result.SetMeta("expiry", Number(0)) // No expiry time set
		}

		return result, nil
	})

	rt.Register("cbRemove", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("cbRemove requires 2 arguments: nodeName, documentId")
		}

		// Add nil checks
		if args[0] == nil {
			return nil, fmt.Errorf("nodeName cannot be nil")
		}
		if args[1] == nil {
			return nil, fmt.Errorf("documentId cannot be nil")
		}

		nodeName := string(args[0].(Str))
		docId := string(args[1].(Str))

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		if err := cbNode.Remove(docId); err != nil {
			return nil, fmt.Errorf("failed to remove document: %v", err)
		}

		return Str("Document removed"), nil
	})

	// Utility to close connection
	rt.Register("cbClose", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("cbClose requires 1 argument: nodeName")
		}

		nodeName := string(args[0].(Str))

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		cbNode.Close()
		delete(rt.objects, nodeName)

		return Str("Connection closed"), nil
	})

	rt.Register("cbReplace", func(args ...Value) (Value, error) {
		if len(args) < 3 || len(args) > 5 {
			return nil, fmt.Errorf("cbReplace requires 3-5 arguments: nodeName, documentId, document, [cas], [expiryDuration]")
		}

		nodeName := string(args[0].(Str))
		docId := string(args[1].(Str))
		document := args[2]

		// Optional CAS
		var cas string
		if len(args) >= 4 && args[3] != nil && args[3] != DBNull {
			if str, ok := args[3].(Str); ok {
				cas = string(str)
			} else {
				return nil, fmt.Errorf("cas must be a string")
			}
		}

		// Optional expiry parameter (default: no expiry)
		var expiryDuration time.Duration
		if len(args) == 5 {
			switch exp := args[4].(type) {
			case Number:
				// If a number is provided, assume it's seconds
				expiryDuration = time.Duration(float64(exp)) * time.Second
			case Str:
				// Parse duration string
				var err error
				expiryDuration, err = time.ParseDuration(string(exp))
				if err != nil {
					return nil, fmt.Errorf("invalid expiry duration: %v", err)
				}
			default:
				return nil, fmt.Errorf("expiry must be a number (seconds) or a duration string")
			}
		}

		cbNode, err := getCouchbaseNode(rt, nodeName)
		if err != nil {
			return nil, err
		}

		// Stricter type checking
		var doc interface{}
		var originalDoc Value // Keep track of original document structure

		switch d := document.(type) {
		case *SimpleJSON:
			// Most efficient path - use native value directly
			doc = d.value
			originalDoc = d

		case *MapValue:
			// Convert MapValue to native map
			nativeMap := make(map[string]interface{})
			for k, v := range d.Values {
				nativeMap[k] = convertValueToNative(v)
			}
			doc = nativeMap
			originalDoc = d

		case *JSONNode:
			// Allow JSONNode but return the ORIGINAL node with metadata
			doc = d.GetJSONValue()
			originalDoc = d // Keep the original JSONNode

		default:
			return nil, fmt.Errorf("unsupported document type: %T - use SimpleJSON, MapValue, or JSONNode", document)
		}

		// Replace document
		docMeta, err := cbNode.Replace(docId, doc, cas, expiryDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to replace document: %v", err)
		}

		// Return document WITH metadata
		var result Value

		// Create appropriate return value based on input type
		switch d := originalDoc.(type) {
		case *SimpleJSON:
			result = d
		case *MapValue:
			// Create SimpleJSON wrapping MapValue data
			result = &SimpleJSON{value: doc}
		case *JSONNode:
			// Return the original JSONNode
			result = d // Simply use the original node
		}

		// Apply metadata to whatever result type we have
		setDocumentMeta(result, uint64(docMeta.Cas()), expiryDuration)

		return result, nil
	})

	rt.Register("newID", func(args ...Value) (Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("newID requires 1, 2 or 3 arguments")
		}
		var rowSQL map[string]interface{}

		format := "short" // Default format
		prefix := "doc"

		if tvar, ok := args[0].(Str); ok {
			prefix = string(tvar)
		}
		if tvar, ok := args[1].(Str); ok {
			format = string(tvar)
		}

		result := generateDocId(prefix, format, rowSQL)
		return Str(result), nil
	})

}

// Helper function to get Couchbase node from runtime
func getCouchbaseNode(rt *Runtime, nodeName string) (*CouchbaseNode, error) {
	obj, exists := rt.objects[nodeName]
	if !exists {
		return nil, fmt.Errorf("couchbase node '%s' not found", nodeName)
	}

	cbNode, ok := obj.(*CouchbaseNode)
	if !ok {
		return nil, fmt.Errorf("object '%s' is not a Couchbase node", nodeName)
	}

	return cbNode, nil
}

// Helper function to convert ArrayValue to []interface{}
func convertArrayToInterface(arr *ArrayValue) []interface{} {
	result := make([]interface{}, arr.Length())

	for i := 0; i < arr.Length(); i++ {
		value := arr.Get(i)
		switch v := value.(type) {
		case Number:
			result[i] = float64(v)
		case Str:
			result[i] = string(v)
		case Bool:
			result[i] = bool(v)
		case *JSONNode:
			result[i] = v.GetJSONValue()
		case *MapNode:
			result[i] = v.ToMap()
		case *ArrayValue:
			result[i] = convertArrayToInterface(v) // Recursive
		case *MapValue:
			// Convert MapValue to native map
			nativeMap := make(map[string]interface{})
			for k, v := range v.Values {
				nativeMap[k] = convertValueToNative(v)
			}
			result[i] = nativeMap
		case map[string]Value:
			// Convert map[string]Value to native map
			nativeMap := make(map[string]interface{})
			for k, v := range v {
				nativeMap[k] = convertValueToNative(v)
			}
			result[i] = nativeMap
		case nil:
			result[i] = nil
		default:
			result[i] = fmt.Sprintf("%v", v)
		}
	}

	return result
}

// setDocumentMeta applies standard metadata to any value that supports it
func setDocumentMeta(val Value, cas uint64, expiry time.Duration) {
	// Use type assertion to MetadataHolder interface
	if holder, ok := val.(interface {
		SetMeta(key string, value Value)
	}); ok {
		// Set CAS as string to preserve precision
		cfg.ChariotLogger.Debug(fmt.Sprintf("Setting CAS %d on document", cas))
		holder.SetMeta("cas", Str(strconv.FormatUint(cas, 10)))

		// Set expiry if non-zero
		if expiry > 0 {
			holder.SetMeta("expiry", Number(expiry.Seconds()))
		}
	}
}

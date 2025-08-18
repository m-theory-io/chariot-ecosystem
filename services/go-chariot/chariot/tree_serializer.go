package chariot

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Register node types for serialization
func init() {
	gob.Register(GobReference{})
	gob.Register(GobTreeNode{})
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	// Register TreeNode concrete types
	gob.Register(&TreeNodeImpl{})
	gob.Register(&XMLNode{})
	gob.Register(&MapNode{})
	gob.Register(&JSONNode{})
	gob.Register(&CSVNode{})
	gob.Register(&SQLNode{})
	gob.Register(&YAMLNode{})
	gob.Register(&CouchbaseNode{})
	// Register value types
	gob.Register(Number(0))
	gob.Register(Str(""))
	gob.Register(Bool(false))
	gob.Register(&ArrayValue{})
	gob.Register(&MapValue{})
	gob.Register(&TableValue{})
	gob.Register(&SimpleJSON{})
	gob.Register(&OfferVariable{})

}

// Update the options structures to use Key IDs
type SecureSerializationOptions struct {
	EncryptionKeyID   string // Key ID in Azure Key Vault
	SigningKeyID      string // Signing key ID in Azure Key Vault
	VerificationKeyID string // Verification key ID (optional)
	Checksum          bool
	Watermark         string
	CompressionLevel  int
	AuditTrail        bool
}

type SecureDeserializationOptions struct {
	DecryptionKeyID   string // Key ID in Azure Key Vault
	VerificationKeyID string // Verification key ID in Azure Key Vault
	RequireSignature  bool
	AuditTrail        bool
}

// Update container to store key IDs instead of raw keys
type SignedAgentContainer struct {
	Version           string                 `gob:"version"`
	Timestamp         time.Time              `gob:"timestamp"`
	Watermark         string                 `gob:"watermark"`
	Checksum          []byte                 `gob:"checksum"`
	EncryptedData     []byte                 `gob:"encrypted_data"`
	Signature         []byte                 `gob:"signature"`
	SigningKeyID      string                 `gob:"signing_key_id"`      // Store key ID, not raw key
	VerificationKeyID string                 `gob:"verification_key_id"` // For key rotation support
	Metadata          map[string]interface{} `gob:"metadata"`
}

type TreeSerializerService struct {
	serializer  *TreeNodeSerializer
	requestChan chan *SerializationRequest
	workerCount int
	logger      *logs.ZapLogger
}

type SerializationRequest struct {
	Type       string   // "save", "load"
	Node       TreeNode // For saves
	Filename   string
	Format     string                // "json", "xml", "yaml", "gob"
	Options    *SerializationOptions // Additional options
	Runtime    *Runtime              // For loads (function reconstruction)
	ResultChan chan *SerializationResult
}

type SerializationOptions struct {
	Compression bool
	PrettyPrint bool
	Format      string
}

type SerializationResult struct {
	Success bool
	Data    TreeNode // For loads
	Error   error
}

// Global service instance
var globalTreeSerializer *TreeSerializerService
var initOnce sync.Once

// Lazy initialization
func getTreeSerializerService() *TreeSerializerService {
	initOnce.Do(func() {
		globalTreeSerializer = NewTreeSerializerService(5, 50)
	})
	return globalTreeSerializer
}

// exported version
func GetTreeSerializerService() *TreeSerializerService {
	return getTreeSerializerService()
}

func NewTreeSerializerService(workers, bufferSize int) *TreeSerializerService {
	service := &TreeSerializerService{
		serializer:  NewTreeNodeSerializer(), // Use existing serializer
		requestChan: make(chan *SerializationRequest, bufferSize),
		workerCount: workers,
		logger:      cfg.ChariotLogger,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go service.worker(i)
	}

	return service
}

func (ts *TreeSerializerService) worker(id int) {
	ts.logger.Info("Worker started", zap.Int("id", id))

	for req := range ts.requestChan {
		result := ts.processRequest(req)

		// Send result back (non-blocking)
		select {
		case req.ResultChan <- result:
			// Success
		default:
			ts.logger.Error("Worker failed to send result", zap.Int("id", id), zap.String("filename", req.Filename))
		}
	}
}

func (ts *TreeSerializerService) processRequest(req *SerializationRequest) *SerializationResult {
	// Configure the serializer for this request
	if req.Options != nil {
		ts.serializer.Format = req.Options.Format
		ts.serializer.Compression = req.Options.Compression
		ts.serializer.PrettyPrint = req.Options.PrettyPrint
	} else if req.Format != "" {
		ts.serializer.Format = req.Format
	}

	switch req.Type {
	case "save":
		err := ts.serializer.SaveTree(req.Node, req.Filename)
		if err != nil {
			ts.logger.Error("save failed", zap.String("filename", req.Filename), zap.Error(err))
			return &SerializationResult{Success: false, Error: err}
		}
		ts.logger.Info("Save successful", zap.String("filename", req.Filename), zap.String("format", ts.serializer.Format))
		return &SerializationResult{Success: true}

	case "load":
		node, err := ts.serializer.LoadTree(req.Filename)
		if err != nil {
			ts.logger.Error("Load failed", zap.String("filename", req.Filename), zap.Error(err))
			return &SerializationResult{Success: false, Error: err}
		}
		ts.logger.Info("Load successful", zap.String("filename", req.Filename))
		return &SerializationResult{Success: true, Data: node}

	default:
		err := fmt.Errorf("unknown request type: %s", req.Type)
		ts.logger.Error("Invalid request", zap.Error(err))
		return &SerializationResult{Success: false, Error: err}
	}
}

// Public async API methods
func (ts *TreeSerializerService) SaveAsync(node TreeNode, filename string, options *SerializationOptions, timeout time.Duration) error {
	resultChan := make(chan *SerializationResult, 1)

	req := &SerializationRequest{
		Type:       "save",
		Node:       node,
		Filename:   filename,
		Options:    options,
		ResultChan: resultChan,
	}

	// Try to queue request (non-blocking)
	select {
	case ts.requestChan <- req:
		// Queued successfully
	default:
		return errors.New("serialization queue is full")
	}

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		if !result.Success {
			return result.Error
		}
		return nil
	case <-time.After(timeout):
		return errors.New("serialization timeout")
	}
}

func (ts *TreeSerializerService) LoadAsync(filename string, rt *Runtime, timeout time.Duration) (TreeNode, error) {
	resultChan := make(chan *SerializationResult, 1)

	req := &SerializationRequest{
		Type:       "load",
		Filename:   filename,
		Runtime:    rt,
		ResultChan: resultChan,
	}

	select {
	case ts.requestChan <- req:
		// Queued successfully
	default:
		return nil, errors.New("serialization queue is full")
	}

	select {
	case result := <-resultChan:
		if !result.Success {
			return nil, result.Error
		}
		return result.Data, nil
	case <-time.After(timeout):
		return nil, errors.New("serialization timeout")
	}
}

// ValidateSecureAgent validates a secure agent file without loading/decrypting it
func (ts *TreeSerializerService) ValidateSecureAgent(filename string, verificationKeyID string) (bool, error) {
	ts.logger.Info("Validating secure agent", zap.String("file", filename), zap.String("verification_key_id", verificationKeyID))

	// Load container without decrypting
	container, err := ts.loadSignedContainer(filename)
	if err != nil {
		return false, fmt.Errorf("failed to load container: %v", err)
	}

	// Verify signature
	if err := ts.verifySignature(container, verificationKeyID); err != nil {
		ts.logger.Error("Signature verification failed", zap.Error(err))
		return false, nil // Return false instead of error for validation
	}

	// Verify checksum if present
	if container.Checksum != nil {
		crypto := getCryptoManager()
		hash := crypto.HashSHA256(container.EncryptedData)
		if !bytes.Equal(hash, container.Checksum) {
			ts.logger.Error("Checksum verification failed")
			return false, nil // Return false instead of error for validation
		}
	}

	ts.logger.Info("Secure agent validation successful")
	return true, nil
}

// GetMetadata retrieves metadata from a secure agent file without decryption
func (ts *TreeSerializerService) GetMetadata(filename string) (map[string]interface{}, error) {
	ts.logger.Info("Getting metadata", zap.String("file", filename))

	// Load container without decrypting
	container, err := ts.loadSignedContainer(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to load container: %v", err)
	}

	// Return combined metadata
	metadata := make(map[string]interface{})

	// Add container-level metadata
	metadata["version"] = container.Version
	metadata["timestamp"] = container.Timestamp.Format(time.RFC3339)
	metadata["watermark"] = container.Watermark
	metadata["signing_key_id"] = container.SigningKeyID
	metadata["verification_key_id"] = container.VerificationKeyID
	metadata["has_signature"] = container.Signature != nil
	metadata["has_checksum"] = container.Checksum != nil
	metadata["data_size"] = len(container.EncryptedData)

	// Add stored metadata
	for k, v := range container.Metadata {
		metadata[k] = v
	}

	return metadata, nil
}

// TreeNodeSerializer handles persistence of TreeNode structures
type TreeNodeSerializer struct {
	Format      string // "json", "xml", "yaml", "gob", "binary"
	Compression bool   // Whether to use compression
	PrettyPrint bool   // Whether to format output for human readability
}

// NewTreeNodeSerializer creates a new serializer with default options
func NewTreeNodeSerializer() *TreeNodeSerializer {
	return &TreeNodeSerializer{
		Format:      cfg.ChariotConfig.TreeFormat,
		Compression: false,
		PrettyPrint: true,
	}
}

// SaveTree serializes a TreeNode tree to file
func (s *TreeNodeSerializer) SaveTree(node TreeNode, filePath string) error {
	var err error
	var fullPath string
	// Get serialization path from cfg
	if fullPath, err = getSecureFilePath(filePath, "tree"); err != nil {
		return err
	} else {
		cfg.ChariotLogger.Info("Using secure file path for tree", zap.String("path", fullPath))
	}
	/*
		serializationPath := cfg.ChariotConfig.TreePath
		fullPath := filepath.Join(serializationPath, filePath)
	*/
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfg.ChariotLogger.Info("Saving tree", zap.String("full_path", fullPath))
	f, err := os.Create(fullPath) // Create the file
	if err != nil {
		return err
	}
	defer f.Close()

	// Choose writer based on compression option
	var writer io.Writer = f
	if s.Compression {
		gzWriter := gzip.NewWriter(f)
		defer gzWriter.Close()
		writer = gzWriter
	}

	// Choose encoding based on format
	switch s.Format {
	case "json":
		var jsonData []byte
		var err error

		// For JSONNode, use its clean MarshalJSON
		if jsonNode, ok := node.(*JSONNode); ok {
			if s.PrettyPrint {
				jsonData, err = json.MarshalIndent(jsonNode.GetJSONValue(), "", "  ")
			} else {
				jsonData, err = jsonNode.MarshalJSON()
			}
		} else {
			// For other nodes, use map serialization
			if s.PrettyPrint {
				jsonData, err = json.MarshalIndent(s.serializeNodeToMap(node), "", "  ")
			} else {
				jsonData, err = json.Marshal(s.serializeNodeToMap(node))
			}
		}

		if err != nil {
			return err
		}
		_, err = writer.Write(jsonData)
		return err

	case "yaml":
		// Convert to map structure then to YAML
		nodeMap := s.serializeNodeToMap(node)
		yamlData, err := yaml.Marshal(nodeMap)
		if err != nil {
			return fmt.Errorf("failed to marshal to YAML: %v", err)
		}
		_, err = writer.Write(yamlData)
		return err

	case "xml":
		// Generate XML
		var xmlData string
		if s.PrettyPrint {
			xmlData, err = s.ToXMLPretty(node)
		} else {
			xmlData, err = s.ToXML(node)
		}
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte(xmlData))
		return err

	case "gob", "binary":
		serializer := NewStatefulGobSerializer()
		data, err := serializer.SerializeTree(node)
		if err != nil {
			return fmt.Errorf("failed to serialize tree with stateful GOB: %w", err)
		}
		_, err = writer.Write(data)
		return err

	default:
		return fmt.Errorf("unsupported format: %s", s.Format)
	}
}

// LoadTree deserializes a TreeNode tree from file
func (s *TreeNodeSerializer) LoadTree(filePath string) (TreeNode, error) {
	// Get serialization path from cfg
	serializationPath := cfg.ChariotConfig.TreePath
	fullPath := filepath.Join(serializationPath, filePath)
	cfg.ChariotLogger.Info("Loading tree", zap.String("full_path", fullPath))

	// Read file content
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Choose reader based on compression
	var reader io.Reader = f
	if s.Compression {
		gzReader, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// Auto-detect format from file extension if not specified
	format := s.Format
	if format == "" || !strings.HasSuffix(filePath, format) {
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".json":
			format = "json"
		case ".xml":
			format = "xml"
		case ".yaml", ".yml":
			format = "yaml"
		case ".gob":
			format = "gob"
		default:
			format = "binary" // Default if can't determine
		}
	}

	cfg.ChariotLogger.Info("Using format for loading", zap.String("format", format), zap.String("file", filePath))

	// Choose decoding based on format
	switch format {
	case "json":
		// Read all data
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return s.LoadTreeFromJSON(string(data))

	case "xml":
		// Read all data
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}
		return s.LoadTreeFromXML(string(data))

	case "yaml":
		// Read all data
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		// Parse YAML data into map[string]interface{}
		var yamlMap map[string]interface{}
		if err := yaml.Unmarshal(data, &yamlMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
		}

		// Convert map to TreeNode
		return s.deserializeNodeFromMap(yamlMap)

	case "gob", "binary":
		data, err := io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read GOB data: %w", err)
		}

		cfg.ChariotLogger.Info("Attempting stateful GOB deserialization", zap.Int("data_size", len(data)))

		// Try stateful deserialization first (this is what we saved with)
		deserializer := &StatefulGobDeserializer{}
		node, err := deserializer.DeserializeTree(data)
		if err != nil {
			cfg.ChariotLogger.Warn("Stateful GOB deserialization failed, trying regular GOB",
				zap.Error(err))

			// Fallback: Try to decode as a direct TreeNode (for backward compatibility)
			buf := bytes.NewBuffer(data)
			decoder := gob.NewDecoder(buf)

			var fallbackNode TreeNode
			err = decoder.Decode(&fallbackNode)
			if err != nil {
				return nil, fmt.Errorf("both stateful and regular GOB deserialization failed: %w", err)
			}
			return fallbackNode, nil
		}
		return node, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// serializeNodeToMap converts a node to a map for JSON serialization
func (s *TreeNodeSerializer) serializeNodeToMap(node TreeNode) map[string]interface{} {
	result := make(map[string]interface{})

	// Add name field for all nodes that have NameStr
	switch n := node.(type) {
	case *TreeNodeImpl:
		result["name"] = n.NameStr
	case *JSONNode:
		result["name"] = n.NameStr // JSONNode inherits NameStr
	case *MapNode:
		result["name"] = n.NameStr // MapNode inherits NameStr
	default:
		// Try to get name through interface if available
		result["name"] = "unknown"
	}

	// Get all existing metadata using the new API
	metaMap := node.GetAllMeta()
	if metaMap != nil && len(metaMap.Values) > 0 {
		// Convert MapValue to native map for serialization
		nativeMetaMap := make(map[string]interface{})
		for k, v := range metaMap.Values {
			nativeMetaMap[k] = convertValueToNative(v)
		}
		result["_meta"] = nativeMetaMap
	}

	// Add serialization-specific metadata to existing _meta
	node.SetMeta("serializer_type", reflect.TypeOf(node).String())
	node.SetMeta("serializer_version", "1.0")

	// Now get the updated metadata
	metaMap = node.GetAllMeta()
	if metaMap != nil && len(metaMap.Values) > 0 {
		// Convert metadata to native map for serialization
		nativeMetaMap := make(map[string]interface{})
		for k, v := range metaMap.Values {
			nativeMetaMap[k] = convertValueToNative(v)
		}
		result["_meta"] = nativeMetaMap
	}

	// Add attributes (clean data only)
	attributes := make(map[string]interface{})
	for k, v := range node.GetAttributes() {
		if k != "_meta" { // Don't duplicate _meta in attributes
			attributes[k] = convertValueToNative(v)
		}
	}
	if len(attributes) > 0 {
		result["attributes"] = attributes
	}

	// Add children
	children := node.GetChildren()
	if len(children) > 0 {
		childrenMaps := make([]map[string]interface{}, len(children))
		for i, child := range children {
			childrenMaps[i] = s.serializeNodeToMap(child)
		}
		result["children"] = childrenMaps
	}

	return result
}

// deserializeNodeFromMap reconstructs a node from a map for JSON deserialization
func (s *TreeNodeSerializer) deserializeNodeFromMap(data map[string]interface{}) (TreeNode, error) {
	// Get node type from _meta instead of direct type field
	var nodeType string
	if metaMap, ok := data["_meta"].(map[string]interface{}); ok {
		if serializerType, ok := metaMap["serializer_type"].(string); ok {
			nodeType = serializerType
		}
	}

	// Get name field
	nameVal, ok := data["name"]
	if !ok {
		return nil, fmt.Errorf("node data missing 'name' field")
	}

	name, ok := nameVal.(string)
	if !ok {
		return nil, fmt.Errorf("node name must be a string")
	}

	// Create appropriate node type
	var node TreeNode

	switch nodeType {
	case "*chariot.TreeNodeImpl": // ADD THIS CASE - it was missing!
		node = NewTreeNode(name)

	case "*chariot.Transform":
		node = NewTransform(name)

	case "*chariot.XMLNode":
		node = NewXMLNode(name)
		// Handle XML-specific data
		if xmlData, ok := data["xmlData"].(map[string]interface{}); ok {
			if ns, ok := xmlData["namespace"].(string); ok {
				node.(*XMLNode).Namespace = ns
			}
			if content, ok := xmlData["content"].(string); ok {
				node.(*XMLNode).Content = content
			}
			if isComment, ok := xmlData["isComment"].(bool); ok {
				node.(*XMLNode).IsComment = isComment
			}
			if isCDATA, ok := xmlData["isCDATA"].(bool); ok {
				node.(*XMLNode).IsCDATA = isCDATA
			}
		}

	case "*chariot.JSONNode":
		node = NewJSONNode(name)
		// Handle JSON-specific data
		if jsonValue, ok := data["jsonValue"]; ok {
			node.(*JSONNode).SetJSONValue(jsonValue)
		}

	// Add other type cases as needed
	case "*chariot.CSVNode":
		node = NewCSVNode(name)
		// Handle CSV-specific data

	case "*chariot.SQLNode":
		node = NewSQLNode(name)
		// Handle SQL-specific data

	case "*chariot.YAMLNode":
		node = NewYAMLNode(name)
		// Handle YAML-specific data

	case "*chariot.MapNode":
		node = NewMapNode(name)
		// Handle Map-specific data

	default:
		// Default to TreeNodeImpl
		node = NewTreeNode(name)
	}

	// Set clear attributes
	if attrMap, ok := data["attributes"].(map[string]interface{}); ok {
		for k, v := range attrMap {
			// Skip _meta attribute to avoid duplication
			if k == "_meta" {
				continue
			}
			// Use existing convertFromNativeValue function
			node.SetAttribute(k, convertFromNativeValue(v))
		}
	}

	// Set metadata
	if metaMap, ok := data["_meta"].(map[string]interface{}); ok {
		metaValue := &MapValue{Values: make(map[string]Value)}
		for k, v := range metaMap {
			// Convert value to appropriate type
			metaValue.Values[k] = convertToValue(v)
		}
		node.SetAllMeta(metaValue)
	}

	// Process children recursively
	if childrenData, ok := data["children"].([]interface{}); ok {
		for _, childData := range childrenData {
			if childMap, ok := childData.(map[string]interface{}); ok {
				child, err := s.deserializeNodeFromMap(childMap)
				if err != nil {
					return nil, err
				}
				node.AddChild(child)
			}
		}
	}

	return node, nil
}

// ToXML converts a TreeNode to XML representation
func (s *TreeNodeSerializer) ToXML(node TreeNode) (string, error) {
	var buffer bytes.Buffer

	// If root node, add XML declaration
	if node.Parent() == nil {
		buffer.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	}

	// Special handling for different node types
	switch n := node.(type) {
	case *XMLNode:
		// For XMLNodes, we can use existing data
		if n.IsComment {
			buffer.WriteString("<!-- ")
			buffer.WriteString(n.Content)
			buffer.WriteString(" -->")
			return buffer.String(), nil
		}

		if n.IsCDATA {
			buffer.WriteString("<")
			buffer.WriteString(n.Name())
			s.writeAttributes(&buffer, n)
			buffer.WriteString("><![CDATA[")
			buffer.WriteString(n.Content)
			buffer.WriteString("]]></")
			buffer.WriteString(n.Name())
			buffer.WriteString(">")
			return buffer.String(), nil
		}

		// Normal XML node
		buffer.WriteString("<")
		if n.Namespace != "" {
			buffer.WriteString(n.Namespace)
			buffer.WriteString(":")
		}
		buffer.WriteString(n.Name())
		s.writeAttributes(&buffer, n)

		if len(n.GetChildren()) == 0 && n.Content == "" {
			buffer.WriteString("/>")
			return buffer.String(), nil
		}

		buffer.WriteString(">")
		if n.Content != "" {
			buffer.WriteString(s.escapeXML(n.Content))
		}

	// In ToXML method, fix JSONNode handling
	case *JSONNode:
		// Convert JSON node to XML
		buffer.WriteString("<")
		buffer.WriteString(n.Name())
		s.writeAttributes(&buffer, n)

		// Get data from clean API
		jsonValue := n.GetJSONValue()

		if jsonValue == nil || (reflect.ValueOf(jsonValue).Kind() == reflect.Map &&
			reflect.ValueOf(jsonValue).Len() == 0) {
			buffer.WriteString("/>")
			return buffer.String(), nil
		}

		buffer.WriteString(">")

		// Convert JSON value to XML content
		switch data := jsonValue.(type) {
		case map[string]interface{}:
			for key, val := range data {
				buffer.WriteString("<")
				buffer.WriteString(key)
				buffer.WriteString(">")
				buffer.WriteString(s.valueToString(val))
				buffer.WriteString("</")
				buffer.WriteString(key)
				buffer.WriteString(">")
			}
		default:
			buffer.WriteString(s.valueToString(jsonValue))
		}

	default:
		// Generic TreeNode handling
		buffer.WriteString("<")
		buffer.WriteString(node.Name())
		s.writeAttributes(&buffer, node)

		if len(node.GetChildren()) == 0 {
			// Check for text content in attributes
			if textAttr, exists := node.GetAttribute("text"); exists {
				buffer.WriteString(">")
				buffer.WriteString(s.escapeXML(fmt.Sprintf("%v", textAttr)))
				buffer.WriteString("</")
				buffer.WriteString(node.Name())
				buffer.WriteString(">")
				return buffer.String(), nil
			}

			buffer.WriteString("/>")
			return buffer.String(), nil
		}

		buffer.WriteString(">")
	}

	// Process children for all node types
	for _, child := range node.GetChildren() {
		childXML, err := s.ToXML(child)
		if err != nil {
			return "", err
		}
		buffer.WriteString(childXML)
	}

	// Close the element
	buffer.WriteString("</")
	buffer.WriteString(node.Name())
	buffer.WriteString(">")

	return buffer.String(), nil
}

// ToXMLPretty converts a TreeNode to formatted XML with indentation
func (s *TreeNodeSerializer) ToXMLPretty(node TreeNode) (string, error) {
	// First get the raw XML
	rawXML, err := s.ToXML(node)
	if err != nil {
		return "", err
	}

	// Parse and reformat with indentation
	var buf bytes.Buffer
	//lint:ignore SA9005 why should I care about this?
	err = xml.Unmarshal([]byte(rawXML), &buf)
	if err != nil {
		// If unmarshaling fails, just return the raw XML
		return rawXML, nil
	}

	var prettyBuf bytes.Buffer
	encoder := xml.NewEncoder(&prettyBuf)
	encoder.Indent("", "  ")
	//lint:ignore SA9005 no downside
	if err := encoder.Encode(&buf); err != nil {
		// If pretty-printing fails, return the raw XML
		return rawXML, nil
	}

	return prettyBuf.String(), nil
}

// ToYAML converts a TreeNode to YAML representation
func (s *TreeNodeSerializer) ToYAML(node TreeNode) (string, error) {
	// Convert to a map structure
	nodeMap := s.serializeNodeToMap(node)

	// Marshal to YAML
	yamlData, err := yaml.Marshal(nodeMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal node to YAML: %v", err)
	}

	return string(yamlData), nil
}

// LoadTreeFromXML parses XML content into a TreeNode
func (s *TreeNodeSerializer) LoadTreeFromXML(xmlContent string) (TreeNode, error) {
	decoder := xml.NewDecoder(strings.NewReader(xmlContent))

	var stack []TreeNode
	var currentNode TreeNode

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Create new node
			newNode := NewXMLNode(t.Name.Local)
			if t.Name.Space != "" {
				newNode.Namespace = t.Name.Space
			}

			// Add attributes
			for _, attr := range t.Attr {
				newNode.SetAttribute(attr.Name.Local, Str(attr.Value))
			}

			// Add to parent if we have one
			if currentNode != nil {
				currentNode.AddChild(newNode)
			}

			// Push to stack
			stack = append(stack, newNode)
			currentNode = newNode

		case xml.EndElement:
			// Pop from stack
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
				currentNode = stack[len(stack)-1]
			}

		case xml.CharData:
			// Add text content to current node
			if currentNode != nil {
				text := string(t)
				trimmed := strings.TrimSpace(text)
				if trimmed != "" || len(text) > 0 {
					xmlNode, ok := currentNode.(*XMLNode)
					if ok {
						xmlNode.Content += text
					} else {
						// If not XMLNode, use attributes
						currentNode.SetAttribute("text", Str(trimmed))
					}
				}
			}

		case xml.Comment:
			// Create comment node
			commentNode := NewXMLNode("#comment")
			commentNode.IsComment = true
			commentNode.Content = string(t)

			if currentNode != nil {
				currentNode.AddChild(commentNode)
			} else {
				// Root comment
				stack = append(stack, commentNode)
			}

		case xml.ProcInst:
			// Processing instructions (like <?xml ... ?>) are ignored

		case xml.Directive:
			// Directives (like <!DOCTYPE...>) are ignored
		}
	}

	// Return root node
	if len(stack) > 0 {
		return stack[0], nil
	}

	return nil, errors.New("failed to parse XML: no root element found")
}

// LoadTreeFromJSON parses JSON content into a TreeNode
func (s *TreeNodeSerializer) LoadTreeFromJSON(jsonContent string) (TreeNode, error) {
	var jsonData interface{}

	if err := json.Unmarshal([]byte(jsonContent), &jsonData); err != nil {
		return nil, err
	}

	// Check if this is our serialized format
	if jsonMap, ok := jsonData.(map[string]interface{}); ok {
		if _, hasMeta := jsonMap["_meta"]; hasMeta {
			// Our serialized format
			return s.deserializeNodeFromMap(jsonMap)
		}
	}

	// Otherwise, treat as natural JSON
	node := NewJSONNode("root")
	node.SetJSONValue(jsonData) // ← Use clean API
	return node, nil
}

// Helper to create a TreeNode from a JSON object
func (s *TreeNodeSerializer) createNodeFromJSON(name string, obj interface{}) (TreeNode, error) {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Create a JSONNode for objects
		node := NewJSONNode(name)

		// Add all properties
		for key, val := range v {
			if nestedMap, ok := val.(map[string]interface{}); ok {
				// Nested object - create child node
				child, err := s.createNodeFromJSON(key, nestedMap)
				if err != nil {
					return nil, err
				}
				node.AddChild(child)
			} else if nestedArray, ok := val.([]interface{}); ok {
				// Array - create array node
				arrayNode := NewJSONNode(key)
				arrayNode.SetAttribute("type", Str("array"))

				// Add array items as children
				for i, item := range nestedArray {
					itemName := fmt.Sprintf("item_%d", i)
					itemNode, err := s.createNodeFromJSON(itemName, item)
					if err != nil {
						return nil, err
					}
					arrayNode.AddChild(itemNode)
				}

				node.AddChild(arrayNode)
			} else {
				// Simple value
				node.Set(key, val)
			}
		}

		return node, nil

	case []interface{}:
		// Create array node
		node := NewJSONNode(name)
		node.SetAttribute("type", Str("array"))

		// Add array items as children
		for i, item := range v {
			itemName := fmt.Sprintf("item_%d", i)
			itemNode, err := s.createNodeFromJSON(itemName, item)
			if err != nil {
				return nil, err
			}
			node.AddChild(itemNode)
		}

		return node, nil

	default:
		// Simple value - create leaf node
		node := NewTreeNode(name)
		node.SetAttribute("value", convertToValue(v))
		return node, nil
	}
}

// Helper to write attributes to XML
func (s *TreeNodeSerializer) writeAttributes(buffer *bytes.Buffer, node TreeNode) {
	// Write regular attributes
	for name, value := range node.GetAttributes() {
		// Skip internal attributes
		if name == "text" || name == "type" || name == "#text" {
			continue
		}

		buffer.WriteString(" ")
		buffer.WriteString(name)
		buffer.WriteString("=\"")
		buffer.WriteString(s.escapeXMLAttr(fmt.Sprintf("%v", value)))
		buffer.WriteString("\"")
	}

	// Optionally write metadata as attributes with meta: prefix
	metaMap := node.GetAllMeta()
	if metaMap != nil {
		for key, value := range metaMap.Values {
			buffer.WriteString(" meta:")
			buffer.WriteString(key)
			buffer.WriteString("=\"")
			buffer.WriteString(s.escapeXMLAttr(fmt.Sprintf("%v", value)))
			buffer.WriteString("\"")
		}
	}
}

// Helper to escape XML content
func (s *TreeNodeSerializer) escapeXML(str string) string {
	str = strings.ReplaceAll(str, "&", "&amp;")
	str = strings.ReplaceAll(str, "<", "&lt;")
	str = strings.ReplaceAll(str, ">", "&gt;")
	return str
}

// Helper to escape XML attribute values
func (s *TreeNodeSerializer) escapeXMLAttr(str string) string {
	str = strings.ReplaceAll(str, "&", "&amp;")
	str = strings.ReplaceAll(str, "<", "&lt;")
	str = strings.ReplaceAll(str, ">", "&gt;")
	str = strings.ReplaceAll(str, "\"", "&quot;")
	str = strings.ReplaceAll(str, "'", "&apos;")
	return str
}

// Helper to convert arbitrary value to string
func (s *TreeNodeSerializer) valueToString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return s.escapeXML(v)
	case nil:
		return ""
	default:
		return s.escapeXML(fmt.Sprintf("%v", v))
	}
}

func (ts *TreeSerializerService) encryptData(data []byte, keyID string) ([]byte, error) {
	crypto := getCryptoManager()
	return crypto.EncryptWithKey(keyID, data) // keyID first, data second
}

func (ts *TreeSerializerService) decryptData(ciphertext []byte, keyID string) ([]byte, error) {
	crypto := getCryptoManager()
	return crypto.DecryptWithKey(keyID, ciphertext) // keyID first, ciphertext second
}

func (ts *TreeSerializerService) signContainer(container *SignedAgentContainer, signingKeyID string) ([]byte, error) {
	crypto := getCryptoManager()

	// Create data to sign
	var buffer bytes.Buffer
	buffer.Write([]byte(container.Version))
	buffer.Write([]byte(container.Timestamp.Format(time.RFC3339)))
	buffer.Write([]byte(container.Watermark))
	buffer.Write(container.Checksum)
	buffer.Write(container.EncryptedData)

	// Use CryptoManager with Key Vault key ID (keyID first, data second)
	signature, err := crypto.SignWithKey(signingKeyID, buffer.Bytes())
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func (ts *TreeSerializerService) verifySignature(container *SignedAgentContainer, verificationKeyID string) error {
	if container.Signature == nil {
		return errors.New("no signature present")
	}

	crypto := getCryptoManager()

	// Use the verification key ID from container if not provided
	keyID := verificationKeyID
	if keyID == "" {
		keyID = container.VerificationKeyID
	}
	if keyID == "" {
		keyID = container.SigningKeyID // Fallback to signing key ID
	}
	if keyID == "" {
		return errors.New("no verification key ID available")
	}

	// Recreate signed data
	var buffer bytes.Buffer
	buffer.Write([]byte(container.Version))
	buffer.Write([]byte(container.Timestamp.Format(time.RFC3339)))
	buffer.Write([]byte(container.Watermark))
	buffer.Write(container.Checksum)
	buffer.Write(container.EncryptedData)

	// Use CryptoManager with Key Vault key ID (keyID first, data second, signature third)
	return crypto.VerifyWithKey(keyID, buffer.Bytes(), container.Signature)
}

func (ts *TreeSerializerService) SaveSecureAgent(node TreeNode, filename string, options *SecureSerializationOptions) error {
	ts.logger.Info("Starting secure agent serialization", zap.String("file", filename), zap.String("encryption_key_id", options.EncryptionKeyID))

	// Step 1: Serialize to GOB (unchanged)
	var gobBuffer bytes.Buffer
	encoder := gob.NewEncoder(&gobBuffer)
	if err := encoder.Encode(node); err != nil {
		ts.logger.Error("GOB encoding failed", zap.Error(err))
		return fmt.Errorf("GOB encoding failed: %v", err)
	}

	agentData := gobBuffer.Bytes()
	ts.logger.Info("GOB serialization complete", zap.Int("size_bytes", len(agentData)))

	// Step 2: Compress data (unchanged)
	var compressedBuffer bytes.Buffer
	level := options.CompressionLevel
	if level == 0 {
		level = gzip.DefaultCompression
	}

	gzWriter, err := gzip.NewWriterLevel(&compressedBuffer, level)
	if err != nil {
		return fmt.Errorf("compression writer creation failed: %v", err)
	}

	if _, err := gzWriter.Write(agentData); err != nil {
		return fmt.Errorf("compression failed: %v", err)
	}
	gzWriter.Close()

	compressedData := compressedBuffer.Bytes()
	ts.logger.Info("Compression complete",
		zap.Int("original_size", len(agentData)),
		zap.Int("compressed_size", len(compressedData)),
		zap.Float64("compression_ratio", float64(len(compressedData))/float64(len(agentData))))

	// Step 3: Encrypt using Key Vault key ID - FIXED PARAMETER ORDER
	finalData := compressedData
	if options.EncryptionKeyID != "" {
		encrypted, err := ts.encryptData(compressedData, options.EncryptionKeyID)
		if err != nil {
			ts.logger.Error("Encryption failed", zap.Error(err), zap.String("key_id", options.EncryptionKeyID))
			return fmt.Errorf("encryption failed with key %s: %v", options.EncryptionKeyID, err)
		}
		finalData = encrypted
		ts.logger.Info("Encryption complete", zap.Int("encrypted_size", len(finalData)), zap.String("key_id", options.EncryptionKeyID))
	}

	// Step 4: Create signed container
	container := &SignedAgentContainer{
		Version:           "1.0",
		Timestamp:         time.Now(),
		Watermark:         options.Watermark,
		EncryptedData:     finalData,
		SigningKeyID:      options.SigningKeyID,      // Store key ID
		VerificationKeyID: options.VerificationKeyID, // Store verification key ID
		Metadata: map[string]interface{}{
			"chariot_version":   "1.0",
			"node_type":         fmt.Sprintf("%T", node),
			"encrypted":         options.EncryptionKeyID != "",
			"encryption_key_id": options.EncryptionKeyID,
			"signing_key_id":    options.SigningKeyID,
			"compressed":        true,
			"compression_level": level,
		},
	}

	// Step 5: Calculate checksum - FIXED TO USE NEW SIGNATURE
	if options.Checksum {
		crypto := getCryptoManager()
		hash := crypto.HashSHA256(finalData) // Now returns []byte directly
		container.Checksum = hash
		ts.logger.Info("Checksum calculated", zap.String("hash", fmt.Sprintf("%x", hash[:8])))
	}

	// Step 6: Sign container using Key Vault - FIXED PARAMETER ORDER
	if options.SigningKeyID != "" {
		signature, err := ts.signContainer(container, options.SigningKeyID)
		if err != nil {
			ts.logger.Error("Signing failed", zap.String("error", err.Error()), zap.String("key_id", options.SigningKeyID))
			return fmt.Errorf("signing failed with key %s: %v", options.SigningKeyID, err)
		}
		container.Signature = signature
		ts.logger.Info("Digital signature applied", zap.String("key_id", options.SigningKeyID))
	}

	// Step 7: Save container (unchanged)
	return ts.saveSignedContainer(container, filename, options)
}

// Update the LoadSecureAgent method calls
func (ts *TreeSerializerService) LoadSecureAgent(filename string, options *SecureDeserializationOptions) (TreeNode, error) {
	ts.logger.Info("Starting secure agent deserialization", zap.String("file", filename), zap.String("decryption_key_id", options.DecryptionKeyID))

	// Step 1: Load signed container (unchanged)
	container, err := ts.loadSignedContainer(filename)
	if err != nil {
		ts.logger.Error("Failed to load container", zap.Error(err))
		return nil, fmt.Errorf("failed to load container: %v", err)
	}

	ts.logger.Info("Container loaded",
		zap.String("version", container.Version),
		zap.Time("timestamp", container.Timestamp),
		zap.String("watermark", container.Watermark),
		zap.String("signing_key_id", container.SigningKeyID))

	// Step 2: Verify signature using Key Vault
	if options.RequireSignature || container.Signature != nil {
		if err := ts.verifySignature(container, options.VerificationKeyID); err != nil {
			ts.logger.Error("Signature verification failed", zap.Error(err), zap.String("key_id", options.VerificationKeyID))
			return nil, fmt.Errorf("signature verification failed: %v", err)
		}
		ts.logger.Info("Digital signature verified", zap.String("key_id", options.VerificationKeyID))
	}

	// Step 3: Verify checksum - FIXED TO USE NEW SIGNATURE
	if container.Checksum != nil {
		crypto := getCryptoManager()
		hash := crypto.HashSHA256(container.EncryptedData) // Now returns []byte directly
		if !bytes.Equal(hash, container.Checksum) {
			ts.logger.Error("Checksum verification failed")
			return nil, fmt.Errorf("checksum verification failed - data may be corrupted")
		}
		ts.logger.Info("Checksum verified")
	}

	// Step 4: Decrypt using Key Vault - FIXED PARAMETER ORDER
	data := container.EncryptedData
	if options.DecryptionKeyID != "" {
		decrypted, err := ts.decryptData(data, options.DecryptionKeyID)
		if err != nil {
			ts.logger.Error("Decryption failed", zap.Error(err), zap.String("key_id", options.DecryptionKeyID))
			return nil, fmt.Errorf("decryption failed with key %s: %v", options.DecryptionKeyID, err)
		}
		data = decrypted
		ts.logger.Info("Decryption complete", zap.String("key_id", options.DecryptionKeyID))
	}

	// Steps 5-6: Decompress and deserialize (unchanged)
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decompression reader creation failed: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %v", err)
	}

	ts.logger.Info("Decompression complete", zap.Int("decompressed_size", len(decompressed)))

	decoder := gob.NewDecoder(bytes.NewReader(decompressed))
	var node TreeNode
	if err := decoder.Decode(&node); err != nil {
		ts.logger.Error("GOB decoding failed", zap.String("error", err.Error()))
		return nil, fmt.Errorf("GOB decoding failed: %v", err)
	}

	ts.logger.Info("Secure agent deserialization complete")
	return node, nil
}

// Add these missing methods to TreeSerializerService

func (ts *TreeSerializerService) saveSignedContainer(container *SignedAgentContainer, filename string, options *SecureSerializationOptions) error {
	// Create directory if needed
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0700); err != nil { // Restrictive permissions
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	// Create file with restrictive permissions
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	// Write GOB-encoded container
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(container); err != nil {
		return fmt.Errorf("failed to encode container: %v", err)
	}

	// Audit trail
	if options.AuditTrail {
		ts.logger.Info("Secure agent saved",
			zap.String("file", filename),
			zap.Time("timestamp", container.Timestamp),
			zap.String("watermark", container.Watermark),
			zap.Bool("signed", container.Signature != nil),
			zap.Bool("encrypted", options.EncryptionKeyID != ""),
			zap.String("encryption_key_id", options.EncryptionKeyID),
			zap.String("signing_key_id", options.SigningKeyID))
	}

	return nil
}

func (ts *TreeSerializerService) loadSignedContainer(filename string) (*SignedAgentContainer, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", filename)
	}

	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	// Decode GOB container
	decoder := gob.NewDecoder(file)
	var container SignedAgentContainer
	if err := decoder.Decode(&container); err != nil {
		return nil, fmt.Errorf("failed to decode container from %s: %v", filename, err)
	}

	ts.logger.Info("Signed container loaded",
		zap.String("file", filename),
		zap.String("version", container.Version),
		zap.Time("timestamp", container.Timestamp),
		zap.String("watermark", container.Watermark),
		zap.Bool("has_signature", container.Signature != nil),
		zap.Bool("has_checksum", container.Checksum != nil),
		zap.String("signing_key_id", container.SigningKeyID))

	return &container, nil
}

// Helper to convert Go values to Chariot Values
// Add a proper convertToValue function that handles the array wrappers
func convertToValue(v interface{}) Value {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string:
		return Str(val)
	case float64:
		return Number(val)
	case bool:
		return Bool(val)
	case map[string]interface{}:
		// Check if this is a wrapped ArrayValue
		if chariotType, exists := val["_chariot_type"]; exists {
			switch chariotType {
			case "ArrayValue":
				if elements, ok := val["_elements"].([]interface{}); ok {
					array := NewArray()
					for _, elem := range elements {
						array.Append(convertToValue(elem))
					}
					return array
				}
			case "ValueArray":
				if elements, ok := val["_elements"].([]interface{}); ok {
					result := make([]Value, len(elements))
					for i, elem := range elements {
						result[i] = convertToValue(elem)
					}
					return result
				}
			}
		}

		// Regular map conversion
		mapVal := &MapValue{Values: make(map[string]Value)}
		for k, v := range val {
			mapVal.Values[k] = convertToValue(v)
		}
		return mapVal
	case []interface{}:
		// Convert to ArrayValue by default
		array := NewArray()
		for _, elem := range val {
			array.Append(convertToValue(elem))
		}
		return array
	default:
		// For any other type, convert to string
		return Str(fmt.Sprintf("%v", v))
	}
}

// Replace deriveKey with your existing function
func deriveKey(password string, salt []byte) []byte {
	crypto := getCryptoManager()
	if salt == nil {
		salt = []byte("chariot-financial-security-2024")
	}
	return crypto.DeriveKeyPBKDF2(password, salt, 100000, 32)
}

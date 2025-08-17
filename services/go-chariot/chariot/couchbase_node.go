package chariot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	cfg "github.com/bhouse1273/go-chariot/configs"
	"github.com/bhouse1273/go-chariot/vault"
	"github.com/couchbase/gocb/v2"
)

type CouchbaseMeta struct {
	CAS    string
	Expiry time.Duration
}

// CouchbaseNode implements TreeNode for Couchbase operations
type CouchbaseNode struct {
	TreeNodeImpl
	// Couchbase-specific fields
	ConnectionString string           // Couchbase connection string
	Username         string           // Authentication username
	Password         string           // Authentication password
	BucketName       string           // Active bucket name
	ScopeName        string           // Active scope name (defaults to _default)
	CollectionName   string           // Active collection name (defaults to _default)
	Cluster          *gocb.Cluster    // Couchbase cluster connection
	Bucket           *gocb.Bucket     // Couchbase bucket connection
	Collection       *gocb.Collection // Couchbase collection reference
	QueryResults     []interface{}    // Results of the last query
	LastQuery        string           // Most recent N1QL query
	LastError        error            // Last error encountered
	connected        bool             // Connection state
}

// NewCouchbaseNode creates a new CouchbaseNode
func NewCouchbaseNode(name string) *CouchbaseNode {
	node := &CouchbaseNode{
		ScopeName:      "_default",
		CollectionName: "_default",
	}
	node.TreeNodeImpl = *NewTreeNode(name)
	return node
}

func (n *CouchbaseNode) SetDiag(flag bool) {
	// Set diagnostic mode (not used in CouchbaseNode)
	if flag {
		gocb.SetLogger(gocb.VerboseStdioLogger())
	} else {
		gocb.SetLogger(gocb.DefaultStdioLogger())
	}
}

// Connect establishes a connection to the Couchbase cluster
func (n *CouchbaseNode) Connect(connStr, username, password string) error {
	ctx := context.Background()
	// Validate args
	if n.ConnectionString != "" && n.Username != "" && n.Password != "" {
		// Use existing connection details if available
		connStr = n.ConnectionString
		username = n.Username
		password = n.Password
	} else {
		// If vault is configured, try to load connection details
		if vault.VaultClient != nil && cfg.ChariotConfig.VaultName != "" {
			secret, err := vault.GetOrgSecret(ctx, cfg.ChariotKey)
			if err != nil {
				return fmt.Errorf("failed to get secret from vault: %v", err)
			}
			if secret != nil {
				n.ConnectionString = secret.CBURL
				n.Username = secret.CBUser
				n.Password = secret.CBPassword
				n.ScopeName = secret.CBScope

				connStr = secret.CBURL
				username = secret.CBUser
				password = secret.CBPassword
			}
		}
	}
	// Test to see if connection values are populated
	if n.ConnectionString == "" || (n.Username == "" || n.Password == "") {
		return errors.New("connection string, username, and password are required")
	}

	// Create connection options
	opts := gocb.ClusterOptions{
		Authenticator: gocb.PasswordAuthenticator{
			Username: username,
			Password: password,
		},
		TimeoutsConfig: gocb.TimeoutsConfig{
			ConnectTimeout:    30 * time.Second, // Increase from default
			KVTimeout:         10 * time.Second,
			QueryTimeout:      75 * time.Second,
			AnalyticsTimeout:  75 * time.Second,
			SearchTimeout:     75 * time.Second,
			ManagementTimeout: 75 * time.Second,
		},
		OrphanReporterConfig: gocb.OrphanReporterConfig{
			Disabled: true, // Disable for testing
		},
	}

	if connStr == "" {
		return fmt.Errorf("connection string is required")
	}
	// Validate username and password
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	// Connect to cluster
	cluster, err := gocb.Connect(connStr, opts)
	if err != nil {
		n.LastError = err
		return err
	}

	n.Cluster = cluster
	n.connected = true
	return nil
}

// Add this method to your CouchbaseNode implementation
func (n *CouchbaseNode) ConnectMeta() error {
	connectionString := GetMetaString(n, "connectionString", "")
	if connectionString == "" {
		return fmt.Errorf("connectionString metadata is required")
	}

	username := GetMetaString(n, "username", "")
	password := GetMetaString(n, "password", "")

	// Connect to cluster first
	err := n.Connect(connectionString, username, password)
	if err != nil {
		return err
	}

	// Then open the bucket if specified
	bucket := GetMetaString(n, "bucket", "")
	if bucket != "" {
		err = n.OpenBucket(bucket)
		if err != nil {
			return err
		}

		// Set scope and collection if specified
		scope := GetMetaString(n, "scope", "_default")
		collection := GetMetaString(n, "collection", "_default")

		err = n.SetScope(scope, collection)
		if err != nil {
			return err
		}
	}

	return nil
}

// OpenBucket opens a specific bucket
func (n *CouchbaseNode) OpenBucket(bucketName string) error {
	if !n.connected {
		return errors.New("not connected to Couchbase")
	}

	// Get bucket reference
	bucket := n.Cluster.Bucket(bucketName)

	// Ensure bucket is accessible
	err := bucket.WaitUntilReady(15*time.Second, nil)
	if err != nil {
		n.LastError = err
		return err
	}

	n.Bucket = bucket
	n.BucketName = bucketName

	// Get default collection
	scope := bucket.Scope(n.ScopeName)
	collection := scope.Collection(n.CollectionName)
	n.Collection = collection

	return nil
}

// SetScope sets the active scope and collection
func (n *CouchbaseNode) SetScope(scopeName, collectionName string) error {
	if n.Bucket == nil {
		return errors.New("no bucket selected")
	}

	n.ScopeName = scopeName
	n.CollectionName = collectionName

	scope := n.Bucket.Scope(scopeName)
	collection := scope.Collection(collectionName)
	n.Collection = collection

	return nil
}

// Query executes a N1QL query
func (n *CouchbaseNode) Query(query string, params map[string]interface{}) error {
	if !n.connected {
		return errors.New("not connected to Couchbase")
	}

	// Save query for reference
	n.LastQuery = query

	// Create query options
	opts := &gocb.QueryOptions{
		NamedParameters: params,
	}

	// Execute query
	result, err := n.Cluster.Query(query, opts)
	if err != nil {
		n.LastError = err
		return err
	}

	// Clear previous results
	n.QueryResults = nil

	// Process results
	var allRows []interface{}
	for result.Next() {
		var row interface{}
		if err := result.Row(&row); err != nil {
			n.LastError = err
			return err
		}
		allRows = append(allRows, row)
	}

	// Check for any errors encountered during iteration
	if err := result.Err(); err != nil {
		n.LastError = err
		return err
	}

	n.QueryResults = allRows

	return nil
}

// GetQueryResults returns the cached query results as an ArrayValue of JSONNodes
func (n *CouchbaseNode) GetQueryResults() *ArrayValue {
	if n.QueryResults == nil {
		return NewArray()
	}

	// Create ArrayValue to hold the rows
	arrayResult := NewArray()

	// Convert each result to a JSONNode
	for i, row := range n.QueryResults {
		fmt.Printf("DEBUG: Processing query result row %d: %+v (type: %T)\n", i, row, row)

		rowNode := NewJSONNode(fmt.Sprintf("row_%d", i))
		rowNode.SetJSONValue(row)

		// Debug: Check what GetJSONValue returns
		retrievedValue := rowNode.GetJSONValue()
		fmt.Printf("DEBUG: After SetJSONValue, GetJSONValue returns: %+v (type: %T)\n", retrievedValue, retrievedValue)

		arrayResult.Append(rowNode)
	}

	fmt.Printf("DEBUG: Final array length: %d\n", arrayResult.Length())
	return arrayResult
}

// Get retrieves a document by ID
func (n *CouchbaseNode) Get(id string) (*JSONNode, error) {
	if n.Collection == nil {
		return nil, errors.New("no collection selected")
	}

	// Get document
	result, err := n.Collection.Get(id, nil)
	if err != nil {
		n.LastError = err
		return nil, err
	}

	// Create node
	docNode := NewJSONNode(id)

	// Extract document content
	var content interface{} // ← interface{}, not map[string]interface{}
	err = result.Content(&content)
	if err != nil {
		n.LastError = err
		return nil, err
	}

	// Set document content
	docNode.SetJSONValue(content)

	// Store CAS as STRING to preserve precision
	docNode.SetMeta("cas", fmt.Sprintf("%d", result.Cas()))
	docNode.SetMeta("expiry", 0)
	if result.Expiry() != nil {
		docNode.SetMeta("expiry", result.Expiry().Seconds())
	}

	return docNode, nil
}

// Insert adds a new document
func (n *CouchbaseNode) Insert(id string, document interface{}, expiry time.Duration) (*gocb.MutationResult, error) {
	if n.Collection == nil {
		return nil, errors.New("no collection selected")
	}

	// Convert document using helper
	doc := n.prepareDocument(document)

	// Create insert options with expiry
	opts := &gocb.InsertOptions{}
	if expiry > 0 {
		opts.Expiry = expiry
	}

	// Insert document
	docRes, err := n.Collection.Insert(id, doc, opts)
	if err != nil {
		n.LastError = err
		return nil, err
	}

	return docRes, nil
}

// Upsert creates or updates a document
func (n *CouchbaseNode) Upsert(id string, document interface{}, expiry time.Duration) (*gocb.MutationResult, error) {
	if n.Collection == nil {
		return nil, errors.New("no collection selected")
	}

	// Convert document using helper
	doc := n.prepareDocument(document)

	// Create upsert options with expiry
	opts := &gocb.UpsertOptions{}
	if expiry > 0 {
		opts.Expiry = expiry
	}

	// Upsert document
	docRes, err := n.Collection.Upsert(id, doc, opts)
	if err != nil {
		n.LastError = err
		return nil, err
	}

	return docRes, nil
}

// Remove deletes a document
func (n *CouchbaseNode) Remove(id string) error {
	var err error
	if n.Collection == nil {
		return errors.New("no collection selected")
	}
	// Remove document
	_, err = n.Collection.Remove(id, nil)
	if err != nil {
		n.LastError = err
		return err
	}

	return nil
}

// Replace updates an existing document
func (n *CouchbaseNode) Replace(id string, document interface{}, cas string, expiry time.Duration) (*gocb.MutationResult, error) {
	if n.Collection == nil {
		return nil, errors.New("no collection selected")
	}

	// Convert document using helper
	doc := n.prepareDocument(document)

	// Create replace options
	opts := &gocb.ReplaceOptions{}

	// Set CAS if provided
	if cas != "" {
		casBits, err := strconv.ParseUint(cas, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid CAS value: %v", err)
		}
		opts.Cas = gocb.Cas(casBits)
	}

	// Set expiry if provided
	if expiry > 0 {
		opts.Expiry = expiry
	}

	// Replace document
	docRes, err := n.Collection.Replace(id, doc, opts)
	if err != nil {
		n.LastError = err
		return nil, err
	}

	return docRes, nil
}

// Close closes the connection
func (n *CouchbaseNode) Close() {
	if n.Cluster != nil {
		n.Cluster.Close(nil)
		n.Cluster = nil
		n.Bucket = nil
		n.Collection = nil
		n.connected = false
	}
}

// Instead of manual type switching, use your standard converters
func (n *CouchbaseNode) prepareDocument(document interface{}) interface{} {
	if jsonNode, ok := document.(*JSONNode); ok {
		return jsonNode.GetJSONValue()
	}
	if mapNode, ok := document.(*MapNode); ok {
		return mapNode.ToMap()
	}
	return convertValueToNative(document) // ← Use comprehensive converter
}

// Helper functions that work with string CAS
func GetCouchbaseCAS(node TreeNode) (string, bool) {
	if cas, exists := node.GetMeta("cas"); exists {
		if casStr, ok := cas.(string); ok {
			return casStr, true
		}
	}
	return "", false
}

func GetCouchbaseExpiry(node TreeNode) (time.Duration, bool) {
	if expiry, exists := node.GetMeta("expiry"); exists {
		if expSeconds, ok := expiry.(float64); ok {
			return time.Duration(expSeconds) * time.Second, true
		}
	}
	return 0, false
}

func SetCouchbaseCAS(node TreeNode, cas string) {
	node.SetMeta("cas", cas)
}

func SetCouchbaseExpiry(node TreeNode, expiry time.Duration) {
	node.SetMeta("expiry", expiry.Seconds())
}

// Only convert to uint64 when needed for gocb SDK
func CASStringToUint64(casStr string) (uint64, error) {
	return strconv.ParseUint(casStr, 10, 64)
}

func CASUint64ToString(cas uint64) string {
	return fmt.Sprintf("%d", cas)
}

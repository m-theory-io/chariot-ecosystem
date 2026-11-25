package chariot

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	cfg "github.com/m-theory-io/chariot-ecosystem/services/go-chariot/configs"
)

func treePersistenceDir(t *testing.T) string {
	t.Helper()
	old := cfg.ChariotConfig.TreePath
	dir := t.TempDir()
	cfg.ChariotConfig.TreePath = dir
	t.Cleanup(func() { cfg.ChariotConfig.TreePath = old })
	return dir
}

func TestTreeLoadMislabeledJSON(t *testing.T) {
	dir := treePersistenceDir(t)
	// Match the real artifact: serialized tree JSON, not natural JSON or GOB.
	data := []byte(`{"name":"usersAgent","_meta":{"serializer_type":"*chariot.TreeNodeImpl"},"children":[{"name":"config","attributes":{"sessionTimeout":3600}}]}`)
	if err := os.WriteFile(filepath.Join(dir, "usersAgent.gob"), data, 0600); err != nil {
		t.Fatal(err)
	}
	node, err := NewTreeNodeSerializer().LoadTree("usersAgent.gob")
	if err != nil {
		t.Fatal(err)
	}
	if node.Name() != "usersAgent" || len(node.GetChildren()) != 1 || node.GetChildren()[0].GetAttributes()["sessionTimeout"] != Number(3600) {
		t.Fatal("serialized JSON tree structure or attribute types were lost")
	}
}

func TestTreeLoadWorkspaceUsersAgent(t *testing.T) {
	dir := treePersistenceDir(t)
	for _, source := range []string{
		"../data/trees/usersAgent.gob",
		"../../../data/go-chariot/trees/usersAgent.gob",
		"../../../infrastructure/docker/go-chariot/data/trees/usersAgent.gob",
	} {
		t.Run(source, func(t *testing.T) {
			data, err := os.ReadFile(source)
			if os.IsNotExist(err) {
				t.Skip("local artifact not present; synthetic regression runs independently")
			}
			if err != nil {
				t.Fatal(err)
			}
			if !json.Valid(data) {
				t.Fatal("artifact is no longer the observed JSON representation")
			}
			// Never load/save against the original path or print its contents.
			if err := os.WriteFile(filepath.Join(dir, "copy.gob"), data, 0600); err != nil {
				t.Fatal(err)
			}
			node, err := NewTreeNodeSerializer().LoadTree("copy.gob")
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, child := range node.GetChildren() {
				if child.Name() == "config" {
					found = len(child.GetAttributes()) > 0
				}
			}
			if node.Name() != "usersAgent" || !found {
				t.Fatal("agent/config structure not restored")
			}
			// Exercise the user's actual source, without printing the result.
			script, err := os.ReadFile("../../charioteer/files/applyTest.ch")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "usersAgent.gob"), data, 0600); err != nil {
				t.Fatal(err)
			}
			rt := classRuntime()
			RegisterTreeFunctions(rt)
			RegisterString(rt)
			RegisterNode(rt)
			result := dotExec(t, rt, string(script))
			if value, ok := result.(Str); !ok || len(value) == 0 {
				t.Fatal("applyTest did not return a nonempty string")
			}
		})
	}
}

func TestTreePersistenceEncodings(t *testing.T) {
	dir := treePersistenceDir(t)
	node := NewTreeNode("agent")
	node.SetAttribute("answer", Number(42))
	stateful, err := NewStatefulGobSerializer().SerializeTree(node)
	if err != nil {
		t.Fatal(err)
	}
	var direct bytes.Buffer
	var tree TreeNode = node
	if err := gob.NewEncoder(&direct).Encode(&tree); err != nil {
		t.Fatal(err)
	}
	zip := func(data []byte) []byte {
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	wrap := func(data []byte) []byte {
		var b bytes.Buffer
		if err := gob.NewEncoder(&b).Encode(data); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	serializedJSON := []byte(`{"name":"agent","_meta":{},"attributes":{"answer":42}}`)
	for name, data := range map[string][]byte{
		"stateful": stateful, "direct": direct.Bytes(),
		"gzip-stateful": zip(stateful), "gzip-direct": zip(direct.Bytes()),
		"wrapped-stateful": wrap(stateful), "wrapped-direct": wrap(direct.Bytes()),
		"wrapped-json": wrap(serializedJSON), "wrapped-gzip-json": wrap(zip(serializedJSON)),
		"gzip-json": zip(serializedJSON),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, "agent.gob"), data, 0600); err != nil {
				t.Fatal(err)
			}
			loaded, err := NewTreeNodeSerializer().LoadTree("agent.gob")
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Name() != "agent" || loaded.GetAttributes()["answer"] != Number(42) {
				t.Fatal("round trip changed tree")
			}
		})
	}
	for name, data := range map[string][]byte{
		"empty": nil, "truncated": stateful[:len(stateful)/2],
		"unsupported-gob": wrap([]byte("not a tree")), "invalid-json": []byte(`{"name":`),
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, "bad.gob"), data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := NewTreeNodeSerializer().LoadTree("bad.gob"); err == nil {
				t.Fatal("invalid payload accepted")
			}
		})
	}
}

func TestTreeSaveInferredFormatAndRequestIsolation(t *testing.T) {
	dir := treePersistenceDir(t)
	rt := classRuntime()
	RegisterTreeFunctions(rt)
	dotExec(t, rt, `treeSave(newTree("agent"), "inferred.gob")`)
	data, err := os.ReadFile(filepath.Join(dir, "inferred.gob"))
	if err != nil {
		t.Fatal(err)
	}
	if json.Valid(data) {
		t.Fatal("treeSave wrote JSON to inferred .gob file")
	}
	if _, err := (&StatefulGobDeserializer{}).DeserializeTree(data); err != nil {
		t.Fatal(err)
	}
	service := &TreeSerializerService{logger: cfg.ChariotLogger}
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			format := "json"
			if i%2 == 0 {
				format = "gob"
			}
			name := filepath.Base(t.Name()) + strings.Repeat("x", i) + "." + format
			node := NewTreeNode("isolated")
			result := service.processRequest(&SerializationRequest{Type: "save", Node: node, Filename: name, Options: &SerializationOptions{Format: format, Compression: i%3 == 0}})
			if result.Error != nil {
				t.Error(result.Error)
				return
			}
			result = service.processRequest(&SerializationRequest{Type: "load", Filename: name})
			if result.Error != nil {
				t.Error(result.Error)
				return
			}
			if result.Data.Name() != "isolated" {
				t.Error("request inherited another request's settings")
			}
		}(i)
	}
	wg.Wait()
}

func TestTreeSaveExplicitFormatAndCompressedSuffix(t *testing.T) {
	dir := treePersistenceDir(t)
	rt := classRuntime()
	RegisterTreeFunctions(rt)
	for _, test := range []struct {
		name, source string
	}{
		{"explicit.gob", `treeSave(newTree("agent"), "explicit.gob", "json")`},
		{"compressed.json.gz", `treeSave(newTree("agent"), "compressed.json.gz", "json", true)`},
		{"compressed.gob.gz", `treeSave(newTree("agent"), "compressed.gob.gz", "gob", true)`},
		{"alias.yml", `treeSave(newTree("agent"), "alias.yml")`},
	} {
		t.Run(test.name, func(t *testing.T) {
			dotExec(t, rt, test.source)
			if test.name == "explicit.gob" {
				data, err := os.ReadFile(filepath.Join(dir, test.name))
				if err != nil {
					t.Fatal(err)
				}
				if !json.Valid(data) {
					t.Fatal("explicit JSON override ignored")
				}
			}
			node, err := NewTreeNodeSerializer().LoadTree(test.name)
			if err != nil {
				t.Fatal(err)
			}
			if node.Name() != "agent" {
				t.Fatal("tree name changed")
			}
		})
	}
}

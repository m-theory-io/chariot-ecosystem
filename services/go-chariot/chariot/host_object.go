package chariot

import (
	"fmt"
	"reflect"
	"strings"
)

// HostObjectValue wraps a Go object as a Chariot Value
type HostObjectValue struct {
	Value interface{}
	Name  string // Optional name for better debugging
}

// HostObjectRegistrationResult represents the status of a host object registration
type HostObjectRegistrationResult int

const (
	HostObjectOK HostObjectRegistrationResult = iota
	HostObjectDuplicate
	HostObjectError
)

// Implement Value interface
func (h *HostObjectValue) Type() string {
	return "hostobject"
}

func (h *HostObjectValue) String() string {
	if h.Name != "" {
		return fmt.Sprintf("HostObject(%s)", h.Name)
	}
	return fmt.Sprintf("%v", h.Value)
}

// RegisterHostObject binds a Go object to the runtime under the specified name
// allowing Chariot scripts to call methods and access properties on the object.
func (rt *Runtime) RegisterHostObject(name string, obj interface{}, overwrite bool) HostObjectRegistrationResult {
	if _, exists := rt.objects[name]; exists {
		if overwrite {
			delete(rt.objects, name)
		} else {
			return HostObjectDuplicate
		}
	}

	// Initialize objects map if not initialized
	if rt.objects == nil {
		rt.objects = make(map[string]interface{})
	}

	// Register the object
	rt.objects[name] = obj
	return HostObjectOK
}

// BindObject exposes a Go object for method calls under a name.
// This is an alias for RegisterHostObject for backward compatibility
func (rt *Runtime) BindObject(name string, obj interface{}) {
	rt.RegisterHostObject(name, obj, true)
}

// CallHostMethod uses reflection to invoke a method on a bound object.
func (rt *Runtime) CallHostMethod(obj interface{}, methodName string, args []Value) (Value, error) {
	// Get the reflection value of the object
	objValue := reflect.ValueOf(obj)

	// Find the method on the object
	method := objValue.MethodByName(methodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("method not found: %s", methodName)
	}

	// Check if method is callable
	if method.Kind() != reflect.Func {
		return nil, fmt.Errorf("%s is not a method", methodName)
	}

	methodType := method.Type()
	numArgs := methodType.NumIn()

	// Check argument count
	if len(args) != numArgs {
		return nil, fmt.Errorf("incorrect number of arguments for %s (expected %d, got %d)",
			methodName, numArgs, len(args))
	}

	// Convert Chariot values to Go values
	goArgs := make([]reflect.Value, numArgs)
	for i := 0; i < numArgs; i++ {
		expectedType := methodType.In(i)
		goValue, err := rt.convertToGoValue(args[i], expectedType)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %v", i, err)
		}
		goArgs[i] = goValue
	}

	// Call the method
	results := method.Call(goArgs)

	// Handle the return value
	if len(results) == 0 {
		return nil, nil // Void method
	}

	// Convert the first return value to a Chariot value
	result, err := rt.convertToChariotValue(results[0])
	if err != nil {
		return nil, fmt.Errorf("error converting return value: %v", err)
	}

	// Check for error return value (if method returns multiple values and last is error)
	if len(results) > 1 && results[len(results)-1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !results[len(results)-1].IsNil() {
			err := results[len(results)-1].Interface().(error)
			return nil, err
		}
	}

	return result, nil
}

// SetObjectProperty sets a property on a host object
func (rt *Runtime) SetObjectProperty(objectName string, propertyName string, value Value) (Value, error) {
	obj, exists := rt.objects[objectName]
	if !exists {
		return nil, fmt.Errorf("object %s not found", objectName)
	}

	return rt.setObjectProperty(obj, propertyName, value)
}

// setObjectProperty is a helper function that sets a property on any object
func (rt *Runtime) setObjectProperty(obj interface{}, propertyName string, value Value) (Value, error) {
	// Handle map[string]Value objects
	if m, ok := obj.(map[string]Value); ok {
		m[propertyName] = value
		return value, nil
	}

	// Handle map[string]interface{} objects
	if m, ok := obj.(map[string]interface{}); ok {
		// Convert Chariot value to Go value
		goValue, err := rt.convertToGoValue(value, reflect.TypeOf((*interface{})(nil)).Elem())
		if err != nil {
			return nil, fmt.Errorf("error converting value: %v", err)
		}
		m[propertyName] = goValue
		return value, nil
	}

	// Handle structs using reflection
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not a struct or map, cannot set property '%s'", propertyName)
	}

	// Find field by name (case-insensitive)
	field := rv.FieldByName(propertyName)
	if !field.IsValid() {
		// Try case-insensitive search
		structType := rv.Type()
		for i := 0; i < structType.NumField(); i++ {
			if strings.EqualFold(structType.Field(i).Name, propertyName) {
				field = rv.Field(i)
				break
			}
		}
	}

	if !field.IsValid() {
		return nil, fmt.Errorf("field '%s' not found in object", propertyName)
	}

	if !field.CanSet() {
		return nil, fmt.Errorf("field '%s' is not settable", propertyName)
	}

	// Convert Chariot value to appropriate Go type
	goValue, err := rt.convertToGoValue(value, field.Type())
	if err != nil {
		return nil, fmt.Errorf("error converting value: %v", err)
	}

	val := reflect.ValueOf(goValue)
	if !val.Type().AssignableTo(field.Type()) {
		return nil, fmt.Errorf("cannot assign %s to field '%s' of type %s", val.Type(), propertyName, field.Type())
	}

	field.Set(val)
	return value, nil
}

// GetHostObject retrieves a registered host object
func (rt *Runtime) GetHostObject(name string) (interface{}, bool) {
	obj, exists := rt.objects[name]
	return obj, exists
}

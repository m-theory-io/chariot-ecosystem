package chariot

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// RegisterHostFunctions registers host object related functions
func RegisterHostFunctions(rt *Runtime) {
	// Register function to create a host object (alias for hostObject for compatibility)
	rt.Register("createHostObject", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("createHostObject requires 1 argument: name")
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("argument must be object name string")
		}

		// Create an empty map for the object
		rt.objects[string(name)] = make(map[string]Value)
		return &HostObjectValue{Value: rt.objects[string(name)], Name: string(name)}, nil
	})

	// Register function to set a property on a host object
	rt.Register("setHostProperty", func(args ...Value) (Value, error) {
		if len(args) != 3 {
			return nil, errors.New("setHostProperty requires 3 arguments: object, propertyName, value")
		}

		var obj interface{}
		switch hostObj := args[0].(type) {
		case *HostObjectValue:
			obj = hostObj.Value
		case Str:
			// Look up by name
			var exists bool
			obj, exists = rt.objects[string(hostObj)]
			if !exists {
				return nil, fmt.Errorf("host object '%s' not found", string(hostObj))
			}
		default:
			return nil, errors.New("first argument must be host object or object name")
		}

		propertyName, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("second argument must be property name string")
		}

		value := args[2]

		// Set the property based on object type
		if m, ok := obj.(map[string]Value); ok {
			m[string(propertyName)] = value
			return value, nil
		}

		return nil, errors.New("cannot set property on this object type")
	})

	// Register function to get a property from a host object
	rt.Register("getHostProperty", func(args ...Value) (Value, error) {
		if len(args) != 2 {
			return nil, errors.New("getHostProperty requires 2 arguments: object, propertyName")
		}

		var obj interface{}
		switch hostObj := args[0].(type) {
		case *HostObjectValue:
			obj = hostObj.Value
		case Str:
			// Look up by name
			var exists bool
			obj, exists = rt.objects[string(hostObj)]
			if !exists {
				return nil, fmt.Errorf("host object '%s' not found", string(hostObj))
			}
		default:
			return nil, errors.New("first argument must be host object or object name")
		}

		propertyName, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("second argument must be property name string")
		}

		// Get the property using the existing helper function
		result, err := getObjectProperty(obj, string(propertyName))
		if err != nil {
			return nil, err
		}

		// Convert the result to a Chariot Value
		if val, ok := result.(Value); ok {
			return val, nil
		}

		// Convert Go values to Chariot values
		return convertFromNativeValue(result), nil
	})

	// Register function to call a method on a host object (alias for callMethod)
	rt.Register("callHostMethod", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("callHostMethod requires at least 2 arguments: object, methodName, [args...]")
		}

		var obj interface{}
		switch hostObj := args[0].(type) {
		case *HostObjectValue:
			obj = hostObj.Value
		case Str:
			// Look up by name
			var exists bool
			obj, exists = rt.objects[string(hostObj)]
			if !exists {
				return nil, fmt.Errorf("host object '%s' not found", string(hostObj))
			}
		default:
			return nil, errors.New("first argument must be host object or object name")
		}

		methodName, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("second argument must be method name string")
		}

		// Call the method using the existing CallHostMethod
		return rt.CallHostMethod(obj, string(methodName), args[2:])
	})

	// Register function to call a method on a host object
	rt.Register("callMethod", func(args ...Value) (Value, error) {
		if len(args) < 2 {
			return nil, errors.New("callMethod requires at least 2 arguments: object and methodName")
		}

		var objName string
		switch obj := args[0].(type) {
		case Str:
			objName = string(obj)
		case *HostObjectValue:
			return rt.CallHostMethod(obj.Value, string(args[1].(Str)), args[2:])
		default:
			return nil, errors.New("first argument must be object name or host object")
		}

		// Get the object by name
		hostObj, exists := rt.objects[objName]
		if !exists {
			return nil, fmt.Errorf("host object '%s' not found", objName)
		}

		// Get the method name
		methodName, ok := args[1].(Str)
		if !ok {
			return nil, errors.New("second argument must be method name string")
		}

		// Call the method
		return rt.CallHostMethod(hostObj, string(methodName), args[2:])
	})

	// Register a new host object from script
	rt.Register("hostObject", func(args ...Value) (Value, error) {
		if len(args) < 1 || len(args) > 2 {
			return nil, errors.New("hostObject requires 1-2 arguments: name, [obj]")
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("first argument must be object name")
		}

		// If only one arg provided, create an empty object
		if len(args) == 1 {
			rt.objects[string(name)] = make(map[string]Value)
			return &HostObjectValue{Value: rt.objects[string(name)], Name: string(name)}, nil
		}

		// Store the provided value
		rt.objects[string(name)] = args[1]
		return &HostObjectValue{Value: args[1], Name: string(name)}, nil
	})

	rt.Register("getHostObject", func(args ...Value) (Value, error) {
		if len(args) != 1 {
			return nil, errors.New("getHostObject requires 1 argument: object name")
		}

		name, ok := args[0].(Str)
		if !ok {
			return nil, errors.New("object name must be a string")
		}

		obj, exists := rt.objects[string(name)]
		if !exists {
			return nil, fmt.Errorf("no host object registered with name: %s", name)
		}

		// Wrap the object in a HostObjectValue
		return &HostObjectValue{
			Value: obj,
			Name:  string(name),
		}, nil
	})
}

// Helper function to get a property from an object using reflection
func getObjectProperty(obj interface{}, propertyName string) (interface{}, error) {
	// Handle map[string]Value objects
	if m, ok := obj.(map[string]Value); ok {
		if val, exists := m[propertyName]; exists {
			return val, nil
		}
		return nil, fmt.Errorf("property '%s' not found", propertyName)
	}

	// Handle map[string]interface{} objects
	if m, ok := obj.(map[string]interface{}); ok {
		if val, exists := m[propertyName]; exists {
			return val, nil
		}
		return nil, fmt.Errorf("property '%s' not found", propertyName)
	}

	// Handle structs using reflection
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object is not a struct or map, cannot get property '%s'", propertyName)
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

	return field.Interface(), nil
}

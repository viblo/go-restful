package swagger

import (
	"reflect"
	"strings"
)

type modelBuilder struct {
	Models map[string]Model
}

func (b modelBuilder) addModel(st reflect.Type, nameOverride string) {
	modelName := b.keyFrom(st)
	if nameOverride != "" {
		modelName = nameOverride
	}
	// no models needed for primitive types
	if b.isPrimitiveType(modelName) {
		return
	}
	// see if we already have visited this model
	if _, ok := b.Models[modelName]; ok {
		return
	}
	sm := Model{modelName, []string{}, map[string]ModelProperty{}}
	// store before further initializing
	b.Models[modelName] = sm
	// check for structure or primitive type
	if st.Kind() == reflect.Struct {
		for i := 0; i < st.NumField(); i++ {
			sf := st.Field(i)
			jsonName := sf.Name
			sft := sf.Type
			prop := ModelProperty{}
			required := true
			// see if a tag overrides this
			if jsonTag := st.Field(i).Tag.Get("json"); jsonTag != "" {
				s := strings.Split(jsonTag, ",")
				if s[0] == "-" {
					continue
				} else if s[0] != "" {
					jsonName = s[0]
				}
				if len(s) > 1 {
					switch s[1] {
					case "string":
						prop.Description = "(" + sft.String() + " as string)"
						sft = reflect.TypeOf("")
					case "omitempty":
						required = false
					}
				}
			}
			if required {
				sm.Required = append(sm.Required, jsonName)
			}

			prop.Type = sft.String() // include pkg path

			// override type of list-likes
			if sft.Kind() == reflect.Slice || sft.Kind() == reflect.Array {
				prop.Type = "array"
				elemName := b.getElementTypeName(modelName, jsonName, sft.Elem())
				prop.Items = map[string]string{"$ref": elemName}
				// add|overwrite model for element type
				b.addModel(sft.Elem(), elemName)
			} else if sft.Kind() == reflect.Ptr { // override type of pointer to list-likes
				if sft.Elem().Kind() == reflect.Slice || sft.Elem().Kind() == reflect.Array {
					prop.Type = "array"
					elemName := b.getElementTypeName(modelName, jsonName, sft.Elem().Elem())
					prop.Items = map[string]string{"$ref": elemName}
					// add|overwrite model for element type
					b.addModel(sft.Elem().Elem(), elemName)
				} else {
					// non-array, pointer type
					prop.Type = sft.String()[1:] // no star, include pkg path
					elemName := ""
					if sft.Elem().Name() == "" {
						elemName = modelName + "." + jsonName
						prop.Type = elemName
					}
					b.addModel(sft.Elem(), elemName)
				}
			} else if sft.Name() == "" { // override type of anonymous structs
				prop.Type = modelName + "." + jsonName
				b.addModel(sft, prop.Type)
			}

			sm.Properties[jsonName] = prop
		}
	}

	// update model builder with completed model
	b.Models[modelName] = sm
}

func (b modelBuilder) getElementTypeName(modelName, jsonName string, t reflect.Type) string {
	if t.Name() == "" {
		return modelName + "." + jsonName
	}
	return b.keyFrom(t)
}

func (b modelBuilder) keyFrom(st reflect.Type) string {
	key := st.String()
	if len(st.Name()) == 0 { // unnamed type
		// Swagger UI has special meaning for [
		key = strings.Replace(key, "[]", "||", -1)
	}
	return key
}

func (b modelBuilder) isPrimitiveType(modelName string) bool {
	return strings.Contains("int int32 int64 float32 float64 bool string byte", modelName)
}

package generator

import (
	"fmt"
	"strconv"

	"github.com/cafecito-games/gdproto/internal/ast"
)

const resourceOptionKey = "(gdproto.resource)"

// resourceNativeMethods contains inherited Object, RefCounted, and Resource
// method names that can collide with gdproto's field accessor convention.
// Godot dispatches the native method instead of a same-named GDScript method,
// so an opted-in Resource message must not emit these names verbatim.
var resourceNativeMethods = map[string]struct{}{
	"add_user_signal":            {},
	"get_class":                  {},
	"get_id_for_path":            {},
	"get_incoming_connections":   {},
	"get_indexed":                {},
	"get_instance_id":            {},
	"get_local_scene":            {},
	"get_meta":                   {},
	"get_meta_list":              {},
	"get_method_argument_count":  {},
	"get_method_list":            {},
	"get_name":                   {},
	"get_path":                   {},
	"get_property_list":          {},
	"get_reference_count":        {},
	"get_rid":                    {},
	"get_scene_unique_id":        {},
	"get_script":                 {},
	"get_signal_connection_list": {},
	"get_signal_list":            {},
	"get_translation_domain":     {},
	"has_connections":            {},
	"has_meta":                   {},
	"has_method":                 {},
	"has_signal":                 {},
	"has_user_signal":            {},
	"set_block_signals":          {},
	"set_deferred":               {},
	"set_id_for_path":            {},
	"set_indexed":                {},
	"set_local_to_scene":         {},
	"set_message_translation":    {},
	"set_meta":                   {},
	"set_name":                   {},
	"set_path":                   {},
	"set_path_cache":             {},
	"set_scene_unique_id":        {},
	"set_script":                 {},
	"set_translation_domain":     {},
}

func (g *generator) isResourceMessage(m *ast.Message) bool {
	raw, ok := m.Options[resourceOptionKey]
	if !ok {
		return false
	}
	resource, ok := raw.(bool)
	if !ok {
		g.recordError(fmt.Errorf(
			"message %q: option %s must be a bool, got %T",
			m.Name, resourceOptionKey, raw,
		))
		return false
	}
	return resource
}

// accessorNames records only methods whose preferred name must be changed.
// Names absent from the map retain their existing spelling.
type accessorNames map[string]string

func (names accessorNames) get(preferred string) string {
	if allocated, ok := names[preferred]; ok {
		return allocated
	}
	return preferred
}

// allocateAccessorNames deterministically moves methods that collide with the
// native Resource API. Every preferred generated name is reserved before a
// suffix is chosen, so a mangled accessor cannot steal another field's normal
// method name.
func (g *generator) allocateAccessorNames(m *ast.Message, resource bool) accessorNames {
	if !resource {
		return nil
	}

	preferred := g.preferredAccessorNames(m)
	reserved := make(map[string]struct{}, len(preferred))
	for _, name := range preferred {
		reserved[name] = struct{}{}
	}

	allocated := map[string]struct{}{}
	result := accessorNames{}
	for _, name := range preferred {
		if _, seen := result[name]; seen {
			continue
		}
		if _, native := resourceNativeMethods[name]; !native {
			allocated[name] = struct{}{}
			continue
		}

		candidate := name + "_pb"
		for suffix := 2; ; suffix++ {
			_, preferredCollision := reserved[candidate]
			_, allocatedCollision := allocated[candidate]
			_, nativeCollision := resourceNativeMethods[candidate]
			if !preferredCollision && !allocatedCollision && !nativeCollision {
				break
			}
			candidate = name + "_pb" + strconv.Itoa(suffix)
		}
		result[name] = candidate
		allocated[candidate] = struct{}{}
	}
	return result
}

func (g *generator) preferredAccessorNames(m *ast.Message) []string {
	var names []string
	for _, field := range m.Fields {
		switch {
		case field.Repeated:
			names = append(names, "add_"+field.Name, "get_"+field.Name)
		case g.isMessageLikeType(field):
			names = append(names, "new_"+field.Name, "get_"+field.Name)
		default:
			names = append(names, "set_"+field.Name, "get_"+field.Name)
			if tracksScalarPresence(field) {
				names = append(names, "has_"+field.Name, "clear_"+field.Name)
			}
		}
	}
	for _, oneof := range m.Oneofs {
		for _, field := range oneof.Fields {
			names = append(names, "set_"+field.Name, "get_"+field.Name, "has_"+field.Name)
		}
		names = append(names, "get_"+oneof.Name+"_case")
	}
	for _, field := range m.Maps {
		names = append(names, "add_"+field.Name, "get_"+field.Name)
	}
	return names
}

func storageAnnotations(resource bool) []string {
	if !resource {
		return nil
	}
	return []string{"export_storage"}
}

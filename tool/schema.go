package tool

import "encoding/json"

// ObjectSchema resolves a tool's parameter schema to the flat JSON Schema object
// that provider APIs expect.
//
// Schemas produced by Func are reflected documents whose root is a $ref into
// $defs; providers such as Anthropic and OpenAI want the referenced object
// itself. A schema that is already a plain object is returned unchanged, and an
// unparseable one yields nil.
func ObjectSchema(raw json.RawMessage) map[string]any {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	if resolved, ok := resolveRef(doc); ok {
		return resolved
	}
	return doc
}

// SchemaRequired returns the required property names declared by a schema
// object, tolerating a missing or malformed "required" entry.
func SchemaRequired(schema map[string]any) []string {
	raw, _ := schema["required"].([]any)
	required := make([]string, 0, len(raw))
	for _, r := range raw {
		if name, ok := r.(string); ok {
			required = append(required, name)
		}
	}
	return required
}

// SchemaProperties returns the property map declared by a schema object.
func SchemaProperties(schema map[string]any) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	return properties
}

// resolveRef follows a root "$ref" of the form "#/$defs/Name" into the
// document's own $defs. It reports false when there is nothing to follow.
func resolveRef(doc map[string]any) (map[string]any, bool) {
	ref, ok := doc["$ref"].(string)
	if !ok {
		return nil, false
	}
	const prefix = "#/$defs/"
	if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
		return nil, false
	}
	defs, ok := doc["$defs"].(map[string]any)
	if !ok {
		return nil, false
	}
	target, ok := defs[ref[len(prefix):]].(map[string]any)
	return target, ok
}

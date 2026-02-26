package forge

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var templatePlaceholderPattern = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// substituteTemplate performs a best-effort placeholder replacement and leaves missing placeholders unchanged.
// It is kept for backwards-compatible behavior and unit tests; request execution should use the stricter helpers.
func substituteTemplate(template string, args map[string]interface{}) string {
	rendered, _ := renderTemplateRaw(template, args)
	return rendered
}

func renderTemplateRaw(template string, args map[string]interface{}) (string, []string) {
	return renderTemplate(template, args, func(v interface{}) string {
		return fmt.Sprintf("%v", v)
	})
}

func renderTemplatePath(template string, args map[string]interface{}) (string, []string) {
	return renderTemplate(template, args, func(v interface{}) string {
		return url.PathEscape(fmt.Sprintf("%v", v))
	})
}

func renderTemplate(template string, args map[string]interface{}, encode func(interface{}) string) (string, []string) {
	if template == "" {
		return "", nil
	}

	missingSet := map[string]struct{}{}
	rendered := templatePlaceholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		submatches := templatePlaceholderPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}

		name := submatches[1]
		val, ok := args[name]
		if !ok {
			missingSet[name] = struct{}{}
			return match
		}

		return encode(val)
	})

	if len(missingSet) == 0 {
		return rendered, nil
	}

	missing := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return rendered, missing
}

func extractTemplatePlaceholders(template string) []string {
	if template == "" {
		return nil
	}

	seen := map[string]struct{}{}
	for _, matches := range templatePlaceholderPattern.FindAllStringSubmatch(template, -1) {
		if len(matches) == 2 {
			seen[matches[1]] = struct{}{}
		}
	}

	if len(seen) == 0 {
		return nil
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func formatTemplateVarList(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}

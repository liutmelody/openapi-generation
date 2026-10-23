package changes

import (
	"fmt"
	"sort"
	"strings"
)

// maxCompactMarkdownBytes bounds the compact changelog because Github rejects
// release bodies over 125000 chars. Buffer accounts for surrounding wrapper text.
const maxCompactMarkdownBytes = 100000

// ToMarkdown converts the SDKDiff to Markdown format.
// Optional detailLevel parameter controls verbosity (defaults to DetailLevelCompact).
func ToMarkdown(d SDKDiff, detailLevel ...DetailLevel) string {
	level := DetailLevelCompact
	if len(detailLevel) > 0 {
		level = detailLevel[0]
	}
	if len(d.Changes) == 0 {
		return "No changes detected."
	}

	var sb strings.Builder

	// Extract SDK name from the AST if available
	sdkName := "sdk"
	if d.NewSubsystem != nil && d.NewSubsystem.Config != nil && d.NewSubsystem.Config.Generation.SDKClassName != "" {
		sdkName = d.NewSubsystem.Config.Generation.SDKClassName
	}

	// Sort changes: breaking changes first, then non-breaking
	changes := make([]MethodDiff, len(d.Changes))
	copy(changes, d.Changes)
	sort.Slice(changes, func(i, j int) bool {
		// Breaking changes come first
		iBreaking := changes[i].ArgumentsDiff.IsBreaking || changes[i].SuccessResponseDiff.IsBreaking || changes[i].ErrorResponseDiff.IsBreaking
		jBreaking := changes[j].ArgumentsDiff.IsBreaking || changes[j].SuccessResponseDiff.IsBreaking || changes[j].ErrorResponseDiff.IsBreaking

		if iBreaking != jBreaking {
			return iBreaking // true (breaking) comes before false (non-breaking)
		}

		// If both are breaking or both are non-breaking, maintain original order
		return false
	})

	for _, change := range changes {
		sb.WriteString("* ")
		// Format method name based on language
		methodName := strings.Join(change.MethodParts, ".")
		if d.NewSubsystem != nil && d.NewSubsystem.Target.Target != "" && len(change.MethodParts) > 0 {
			methodName = FormatMethodNameWithSDK(change.MethodParts, d.NewSubsystem.Target.Target, sdkName)
		}
		// Add "sdk." prefix to all method names
		sb.WriteString("`" + methodName + "()" + "`")
		sb.WriteString(": ")

		// Generate description based on change type and diff results
		switch change.Type {
		case MethodAdded:
			sb.WriteString("**Added**")
		case MethodDeleted:
			sb.WriteString("**Removed** (Breaking ⚠️)")
		case MethodDeprecated:
			sb.WriteString("**Deprecated**")
		case ArgumentsChanged, ResponseChanged, ArgumentsAndResponseChanged:
			// For modified methods, describe what changed
			var parts []string
			partDepthEqualToOne := true
			if !change.ArgumentsDiff.Equal {
				targetLang := ""
				if d.NewSubsystem != nil {
					targetLang = d.NewSubsystem.Target.Target
				}
				part := formatDiffCompact(change.ArgumentsDiff, "request", targetLang, level)
				if len(change.ArgumentsDiff.Path) > 1 {
					partDepthEqualToOne = false
				}
				if part != "" {
					parts = append(parts, part)
				}
			}
			if !change.SuccessResponseDiff.Equal {
				targetLang := ""
				if d.NewSubsystem != nil {
					targetLang = d.NewSubsystem.Target.Target
				}
				part := formatDiffCompact(change.SuccessResponseDiff, "response", targetLang, level)
				if len(change.ArgumentsDiff.Path) > 1 {
					partDepthEqualToOne = false
				}
				if part != "" {
					parts = append(parts, part)
				}
			}
			if !change.ErrorResponseDiff.Equal {
				targetLang := ""
				if d.NewSubsystem != nil {
					targetLang = d.NewSubsystem.Target.Target
				}
				part := formatDiffCompact(change.ErrorResponseDiff, "error", targetLang, level)
				if len(change.ArgumentsDiff.Path) > 1 {
					partDepthEqualToOne = false
				}
				if part != "" {
					parts = append(parts, part)
				}
			}
			// We show the diff as bullet points if path is more than 1 level deep or length of parts > 1
			if len(parts) == 1 && partDepthEqualToOne {
				sb.WriteString(parts[0])
			} else {
				sb.WriteString("\n")
				for _, part := range parts {
					sb.WriteString("  * ")
					sb.WriteString(part)
					sb.WriteString("\n")
				}
				continue
			}
		}
		sb.WriteString("\n")
	}

	// For the compact inline changelog, degrade to a summary of counts if body too large.
	// Full per-change detail remains in the HTML report (rendered separately via DetailLevelFull).
	if level == DetailLevelCompact && sb.Len() > maxCompactMarkdownBytes {
		sb.Reset()
		sb.WriteString(summarizeChanges(changes))
	}

	// Prepend language if sb is non-empty
	lang := ""
	if d.OldSubsystem != nil && d.OldSubsystem.Target.Target != "" {
		lang = capitalizeFirstCharacter(d.OldSubsystem.Target.Target)
	}
	if sb.Len() > 0 && lang != "" {
		old := sb.String()
		sb.Reset()
		fmt.Fprintf(&sb, "## %s SDK Changes:\n", lang)
		sb.WriteString(old)
	}

	return sb.String()
}

// summarizeChanges renders a compact count summary used when the full compact
// changelog is too large to embed inline. Full detail remains in the HTML report.
func summarizeChanges(changes []MethodDiff) string {
	var added, removed, deprecated, modified int
	for _, c := range changes {
		switch c.Type {
		case MethodAdded:
			added++
		case MethodDeleted:
			removed++
		case MethodDeprecated:
			deprecated++
		default:
			modified++
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%d method changes summary (please refer to the full report for details):\n", len(changes))
	if added > 0 {
		fmt.Fprintf(&sb, "* %d added\n", added)
	}
	if removed > 0 {
		fmt.Fprintf(&sb, "* %d removed\n", removed)
	}
	if deprecated > 0 {
		fmt.Fprintf(&sb, "* %d deprecated\n", deprecated)
	}
	if modified > 0 {
		fmt.Fprintf(&sb, "* %d changed\n", modified)
	}
	return sb.String()
}

func capitalizeFirstCharacter(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// formatDiffCompact formats a TypeDefDiffResult into a compact format like "**Added** request.foo.bar [breaking]"
func formatDiffCompact(result TypeDefDiffResult, prefix string, lang string, detailLevel DetailLevel) string {
	if result.Equal {
		return ""
	}

	// Special handling for response diffs that involve status codes and content types
	statusIdx := -1
	contentTypeIdx := -1
	for i, segment := range result.Path {
		switch segment.Type {
		case PathSegmentResponseStatus:
			statusIdx = i
		case PathSegmentResponseContentType:
			contentTypeIdx = i
		}
	}

	if statusIdx >= 0 {
		var action string
		switch result.Reason {
		case DiffReasonFieldAdded:
			action = "**Added**"
		case DiffReasonFieldRemoved:
			action = "**Removed**"
		default:
			action = "**Changed**"
		}

		// If path ends with status code
		if statusIdx == len(result.Path)-1 {
			resultStr := " " + "`" + prefix + ".status[" + result.Path[statusIdx].Name + "]" + "`"
			resultStr += " " + action
			if result.IsBreaking {
				resultStr += " (Breaking ⚠️)"
			}
			return resultStr
		}

		// If path ends with content type (immediately after status)
		pathEndsWithContentType := contentTypeIdx == len(result.Path)-1
		contentTypeFollowsStatus := contentTypeIdx == statusIdx+1
		if pathEndsWithContentType && contentTypeFollowsStatus {
			resultStr := " " + "`" + prefix + ".status[" + result.Path[statusIdx].Name + "]"
			// Only show content type if it's not application/json
			if result.Path[contentTypeIdx].Name != "" && result.Path[contentTypeIdx].Name != "application/json" {
				resultStr += ".content[" + result.Path[contentTypeIdx].Name + "`"
			} else {
				resultStr += "`"
			}
			resultStr += " " + action
			if result.IsBreaking {
				resultStr += " (Breaking ⚠️)"
			}
			return resultStr
		}

		// If path continues after status/content type, format the remaining path
		startIdx := statusIdx + 1
		if contentTypeIdx == statusIdx+1 {
			startIdx = contentTypeIdx + 1
		}

		if startIdx < len(result.Path) {
			remainingPath := result.Path[startIdx:]
			pathStr := formatDiffPath(remainingPath, lang)
			if pathStr != "" {
				resultStr := " " + "`" + prefix + "." + pathStr + "`"
				resultStr += " " + action
				if result.IsBreaking {
					resultStr += " (Breaking ⚠️)"
				}
				return resultStr
			}
		}
	}

	// Build a path string
	pathStr := formatDiffPath(result.Path, lang)

	// Add prefix if we have a path - but avoid duplicating "request" or "response"
	if pathStr != "" {
		// Don't duplicate if the path already starts with the prefix
		if !strings.HasPrefix(pathStr, prefix) {
			pathStr = prefix + "." + pathStr
		}
	} else {
		pathStr = prefix
	}

	// For changes with children, format each child separately (only in full detail mode)
	if len(result.Children) > 0 && detailLevel == DetailLevelFull {
		var parts []string
		seen := make(map[string]bool)

		// Calculate the parent path length to create relative paths
		parentPathLen := len(result.Path)

		// Helper function to recursively collect leaf field changes (avoid intermediate nodes)
		var collectLeafChanges func(diff TypeDefDiffResult)
		collectLeafChanges = func(diff TypeDefDiffResult) {
			// If this diff has children, recurse into them instead of reporting this node
			if len(diff.Children) > 0 {
				for _, child := range diff.Children {
					collectLeafChanges(child)
				}
				return
			}

			// This is a leaf node - report it if it's a meaningful change
			if diff.Reason == DiffReasonFieldAdded || diff.Reason == DiffReasonFieldRemoved ||
				diff.Reason == DiffReasonFieldChanged || diff.Reason == DiffReasonFieldDeprecated || diff.Reason == DiffReasonKind ||
				diff.Reason == DiffReasonUnionOptionAdded || diff.Reason == DiffReasonUnionOptionRemoved ||
				diff.Reason == DiffReasonUnionOptionChanged || diff.Reason == DiffReasonUnionChanged ||
				diff.Reason == DiffReasonUnionDiscriminatorAdded ||
				diff.Reason == DiffReasonEnumValueAdded || diff.Reason == DiffReasonEnumValueRemoved {

				// Use relative path from parent
				relativePath := diff.Path
				if len(diff.Path) > parentPathLen {
					relativePath = diff.Path[parentPathLen:]
				}
				leafPathStr := formatDiffPath(relativePath, lang)

				partStr := "`" + leafPathStr + "` " + formatDiffAction(diff.Reason)
				if diff.IsBreaking {
					partStr += " (Breaking ⚠️)"
				}

				if !seen[partStr] {
					seen[partStr] = true
					parts = append(parts, partStr)
				}
			}
		}

		// Collect leaf changes from all children
		for _, child := range result.Children {
			collectLeafChanges(child)
		}

		// If we have specific field changes, show parent with children underneath
		if len(parts) > 0 {
			// Sort parts for deterministic output
			sort.Strings(parts)

			// Build parent header
			var parentAction string
			if result.IsBreaking {
				parentAction = "**Changed** (Breaking ⚠️)"
			} else {
				parentAction = "**Changed**"
			}
			header := "`" + pathStr + "` " + parentAction + "\n    - " + strings.Join(parts, "\n    - ")
			return header
		}
	}

	result_str := " " + "`" + pathStr + "`"
	result_str += " " + formatDiffAction(result.Reason)
	if result.IsBreaking {
		result_str += " (Breaking ⚠️)"
	}

	return result_str
}

func formatDiffAction(reason DiffReason) string {
	switch reason {
	case DiffReasonFieldAdded, DiffReasonUnionOptionAdded, DiffReasonEnumValueAdded:
		return "**Added**"
	case DiffReasonFieldRemoved, DiffReasonUnionOptionRemoved, DiffReasonEnumValueRemoved:
		return "**Removed**"
	case DiffReasonFieldDeprecated:
		return "**Deprecated**"
	default:
		return "**Changed**"
	}
}

// formatDiffPath formats a path into a human-readable string, language-specific
func formatDiffPath(path []PathSegment, lang string) string {
	if len(path) == 0 {
		return ""
	}

	var parts []string
	for i, segment := range path {
		isLastSegment := i == len(path)-1

		switch segment.Type {
		case PathSegmentTypeField:
			parts = append(parts, segment.Name)
		case PathSegmentTypeUnionOption:
			parts = append(parts, "union("+segment.Name+")")
		case PathSegmentTypeArrayItem:
			parts = append(parts, "[]")
		case PathSegmentTypeMapItem:
			// Use Map<ValueType> format for maps
			if segment.Name != "" {
				parts = append(parts, "Map<"+segment.Name+">")
			} else {
				parts = append(parts, "Map<>")
			}
		case PathSegmentTypeEnumValue:
			parts = append(parts, "enum("+segment.Name+")")
		case PathSegmentResponseStatus:
			// Only include status in path if it's the last segment
			if isLastSegment {
				parts = append(parts, "status["+segment.Name+"]")
			}
		case PathSegmentResponseContentType:
			// Only include content type in path if it's the last segment and not application/json
			if isLastSegment && segment.Name != "application/json" {
				parts = append(parts, "content["+segment.Name+"]")
			}
		}
	}

	// Only format field path segments for language-specific output
	return FormatFieldPath(parts, lang)
}

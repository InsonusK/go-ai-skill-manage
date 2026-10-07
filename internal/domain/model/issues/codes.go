package issues

import (
	"slices"
	"strings"
)

// Code is the kind of a problem as a word, e.g. "missing-link-target". Every
// code a problem may carry is declared here, with the number it is printed
// and documented under (ID).
type Code string

// Numbers group the codes by area: 1xx configuration and command line, 2xx
// sources and skills, 3xx links, 4xx targets, 9xx the run itself. The
// letter is the severity (Severity). A number is never reused or changed,
// whatever its letter: it is what users search by.
const (
	CodeInvalidTags        Code = "invalid-tags"
	CodeUnsupportedTags    Code = "unsupported-tags"
	CodeUnsafeSubpath      Code = "unsafe-subpath"
	CodeDuplicateSource    Code = "duplicate-source"
	CodeConflictingExclude Code = "conflicting-exclude"
	CodeTargetOverlap      Code = "target-overlap"
	CodeDeprecatedSetting  Code = "deprecated-setting"
	CodeDeprecatedFlag     Code = "deprecated-flag"

	CodeSourceAcquire  Code = "source-acquire"
	CodeSourceRead     Code = "source-read"
	CodeMissingSubpath Code = "missing-subpath"
	CodeInvalidName    Code = "invalid-name"
	CodeInvalidSkill   Code = "invalid-skill"
	CodeNestedSkill    Code = "nested-skill"
	CodeSkillNotFound  Code = "skill-not-found"
	CodeDuplicateName  Code = "duplicate-name"
	CodeWhenToUseBoth  Code = "when-to-use-both"

	CodeInvalidLink       Code = "invalid-link"
	CodeMissingLinkTarget Code = "missing-link-target"
	CodePathEscape        Code = "path-escape"
	CodeMissingAnchor     Code = "missing-anchor"
	CodeUnselectedSkill   Code = "unselected-skill"
	CodeExternalFolder    Code = "external-folder"
	CodeLinkOverlap       Code = "link-overlap"
	CodeInvalidLinkSpan   Code = "invalid-link-span"
	CodeWebLink           Code = "web-link"
	CodeLinkTarget        Code = "link-target"
	CodeSharedFileLinks   Code = "shared-file-links"
	CodeSharedFileUnread  Code = "shared-file-unread"

	CodeUnmanagedTarget Code = "unmanaged-target"

	CodeCanceled            Code = "canceled"
	CodeUnsupportedPathKind Code = "unsupported-path-kind"
)

var ids = map[Code]string{
	CodeInvalidTags:        "E101",
	CodeUnsupportedTags:    "E102",
	CodeUnsafeSubpath:      "E103",
	CodeDuplicateSource:    "E104",
	CodeConflictingExclude: "E105",
	CodeTargetOverlap:      "E106",
	CodeDeprecatedSetting:  "W107",
	CodeDeprecatedFlag:     "W108",

	CodeSourceAcquire:  "E201",
	CodeSourceRead:     "E202",
	CodeMissingSubpath: "E203",
	CodeInvalidName:    "E204",
	CodeInvalidSkill:   "E205",
	CodeNestedSkill:    "E206",
	CodeSkillNotFound:  "E207",
	CodeDuplicateName:  "E208",
	CodeWhenToUseBoth:  "W209",

	CodeInvalidLink:       "E301",
	CodeMissingLinkTarget: "E302",
	CodePathEscape:        "E303",
	CodeMissingAnchor:     "E304",
	CodeUnselectedSkill:   "E305",
	CodeExternalFolder:    "E306",
	CodeLinkOverlap:       "E307",
	CodeInvalidLinkSpan:   "E308",
	CodeWebLink:           "E309",
	CodeLinkTarget:        "E310",
	CodeSharedFileLinks:   "W311",
	CodeSharedFileUnread:  "W312",

	CodeUnmanagedTarget: "E401",

	CodeCanceled:            "E901",
	CodeUnsupportedPathKind: "E902",
}

// ID is the code's number, e.g. "E302", empty for a code that isn't
// declared here.
func (c Code) ID() string { return ids[c] }

// Severity says whether a problem stops the work.
type Severity string

const (
	// SeverityError stops the work: nothing is written.
	SeverityError Severity = "error"
	// SeverityWarning is something to fix or check that doesn't stop the
	// work.
	SeverityWarning Severity = "warning"
)

// Severity is the letter of the code's number: W -- a warning, anything
// else -- an error, a code that isn't declared here included.
func (c Code) Severity() Severity {
	if strings.HasPrefix(c.ID(), "W") {
		return SeverityWarning
	}
	return SeverityError
}

// Label is how a code is printed: the number, then the word.
//
// Пример: "missing-link-target" -> "E302 missing-link-target"; необъявленный
// код "x" -> "x".
func (c Code) Label() string {
	if id := c.ID(); id != "" {
		return id + " " + string(c)
	}
	return string(c)
}

// Codes lists every declared code, ordered by number.
func Codes() []Code {
	out := make([]Code, 0, len(ids))
	for c := range ids {
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b Code) int {
		return strings.Compare(a.ID()[1:], b.ID()[1:])
	})
	return out
}

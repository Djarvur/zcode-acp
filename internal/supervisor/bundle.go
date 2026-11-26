// macOS ZCode Desktop bundle resolution (Phase 12, MDISC-01/MDISC-02).
//
// The desktop build keeps the same Electron spawn contract as the Linux
// layouts — an Electron binary plus resources/glm/zcode.cjs — but packages it
// as an app bundle:
//
//	ZCode.app/Contents/MacOS/ZCode
//	ZCode.app/Contents/Resources/glm/zcode.cjs
//	ZCode.app/Contents/Resources/config/provider/zcode-builtin.json
//	ZCode.app/Contents/Info.plist
//
// This file is intentionally platform-neutral so the resolver is fully covered
// on Linux CI; only the production scan is gated to darwin.

package supervisor

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// macExecMask is the executable-bit mask a bundle binary must carry.
	macExecMask = 0o111

	// layoutMacApp marks an Install resolved from a macOS .app bundle.
	layoutMacApp = "mac-app"

	appContentsDir   = "Contents"
	appInfoPlist     = "Info.plist"
	appMacOSDir      = "MacOS"
	appResourcesDir  = "Resources"
	appEntrypointRel = "glm/zcode.cjs"
	appProviderDir   = "config/provider"
	appExecutableKey = "CFBundleExecutable"
	appVersionKey    = "CFBundleShortVersionString"
	appBuildKey      = "CFBundleVersion"

	plistKindDict   = "dict"
	plistKindString = "string"
	plistKindArray  = "array"
)

var (
	errMacNotRegular      = errors.New("not a regular file")
	errPlistTopLevel      = errors.New("top-level plist value is not a dict")
	errPlistUnexpectedEnd = errors.New("unexpected end element while reading a value")
)

// macAppInstall validates a macOS .app bundle and returns the Install consumed
// by SpawnArgv. A non-empty reason is the actionable miss for diagnostics; the
// Install is zero in that case. The root is EvalSymlinks-resolved so spawn and
// the builtin revision both see the same path.
func macAppInstall(ctx context.Context, appPath string) (inst Install, reason string) {
	root, reason := resolveMacAppRoot(appPath)
	if reason != "" {
		return Install{}, reason
	}

	contents := filepath.Join(root, appContentsDir)
	plistPath := filepath.Join(contents, appInfoPlist)

	props, reason := readBundleProps(ctx, plistPath)
	if reason != "" {
		return Install{}, reason
	}

	executable, reason := bundleExecutable(props, plistPath)
	if reason != "" {
		return Install{}, reason
	}

	binary := filepath.Join(contents, appMacOSDir, executable)
	if reason := validateBundleBinary(binary); reason != "" {
		return Install{}, reason
	}

	entrypoint := filepath.Join(contents, appResourcesDir, filepath.FromSlash(appEntrypointRel))
	if reason := validateRegularFile(entrypoint); reason != "" {
		return Install{}, reason
	}

	builtin := macBundledBuiltinPath(root)
	if reason := validateRegularFile(builtin); reason != "" {
		return Install{}, reason
	}

	version, reason := bundleVersion(props, plistPath)
	if reason != "" {
		return Install{}, reason
	}

	return Install{
		Root:       root,
		Binary:     binary,
		Entrypoint: entrypoint,
		Layout:     layoutMacApp,
		Version:    version,
	}, ""
}

// resolveMacAppRoot validates the app directory and returns its resolved path.
func resolveMacAppRoot(appPath string) (root, reason string) {
	if strings.TrimSpace(appPath) == "" {
		return "", "empty app path"
	}

	resolved, err := filepath.EvalSymlinks(appPath)
	if err != nil {
		return "", fmt.Sprintf("%s is not a readable app bundle: %v", appPath, err)
	}

	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Sprintf("resolve %s: %v", appPath, err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Sprintf("stat %s: %v", resolved, err)
	}
	if !info.IsDir() {
		return "", resolved + " is not an app bundle directory"
	}
	if !strings.HasSuffix(filepath.Base(resolved), ".app") {
		return "", resolved + " is not a .app bundle"
	}

	return resolved, ""
}

// readBundleProps reads the Info.plist keys and renders a miss reason.
func readBundleProps(ctx context.Context, plistPath string) (props map[string]string, reason string) {
	props, err := readBundleInfo(ctx, plistPath)
	if err != nil {
		return nil, fmt.Sprintf("read %s: %v", plistPath, err)
	}

	return props, ""
}

// bundleExecutable extracts CFBundleExecutable and rejects path escapes.
func bundleExecutable(props map[string]string, plistPath string) (executable, reason string) {
	executable = strings.TrimSpace(props[appExecutableKey])
	if executable == "" {
		return "", plistPath + " has no " + appExecutableKey
	}
	if executable != filepath.Base(executable) || executable == "." || executable == ".." ||
		strings.ContainsAny(executable, `/\`) {
		return "", fmt.Sprintf("%s value %q escapes %s/%s", appExecutableKey, executable,
			appContentsDir, appMacOSDir)
	}

	return executable, ""
}

// bundleVersion reads the short version, falling back to CFBundleVersion.
func bundleVersion(props map[string]string, plistPath string) (version, reason string) {
	version = strings.TrimSpace(props[appVersionKey])
	if version == "" {
		version = strings.TrimSpace(props[appBuildKey])
	}
	if version == "" {
		return "", plistPath + " has no " + appVersionKey + " or " + appBuildKey
	}

	return version, ""
}

// validateBundleBinary requires a regular executable file.
func validateBundleBinary(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("missing bundle executable %s: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Sprintf("bundle executable %s: %v", path, errMacNotRegular)
	}
	if info.Mode().Perm()&macExecMask == 0 {
		return path + " is not executable"
	}

	return ""
}

// validateRegularFile refuses a missing, directory, or non-regular path.
func validateRegularFile(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("missing %s: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Sprintf("%s: %v", path, errMacNotRegular)
	}

	return ""
}

// macBundledBuiltinPath is the provider table shipped inside the app bundle.
func macBundledBuiltinPath(appRoot string) string {
	return filepath.Join(appRoot, appContentsDir, appResourcesDir,
		filepath.FromSlash(appProviderDir), builtinFileName)
}

// macAppOverridePath recognizes the three supported macOS override spellings:
// the .app itself, its Contents directory, and its Contents/MacOS/<exe> file.
// It maps each to the app root; a path outside those shapes returns false so
// the Linux override branches stay in charge.
func macAppOverridePath(path string) (string, bool) {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if strings.HasSuffix(base, ".app") {
		return clean, true
	}

	if base == appContentsDir {
		app := filepath.Dir(clean)
		if strings.HasSuffix(filepath.Base(app), ".app") {
			return app, true
		}

		return "", false
	}

	if filepath.Base(filepath.Dir(clean)) == appMacOSDir {
		contents := filepath.Dir(filepath.Dir(clean))
		app := filepath.Dir(contents)
		if filepath.Base(contents) == appContentsDir && strings.HasSuffix(filepath.Base(app), ".app") {
			return app, true
		}
	}

	return "", false
}

// readBundleInfo reads the string-valued keys of an Info.plist. XML plists are
// parsed in-process; a binary plist is converted with the macOS plutil tool
// when available. Tests use XML fixtures and never invoke plutil.
func readBundleInfo(ctx context.Context, path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	props, parseErr := parsePlistDict(data)
	if parseErr == nil {
		return props, nil
	}

	if runtime.GOOS == goosDarwin {
		args := []string{"-convert", "xml1", "-o", "-", path}
		out, convErr := exec.CommandContext(ctx, "plutil", args...).Output()
		if convErr == nil {
			if converted, err := parsePlistDict(out); err == nil {
				return converted, nil
			}
		}
	}

	return nil, fmt.Errorf("parse Info.plist: %w", parseErr)
}

// plistValue is the subset of plist values bundle metadata needs.
type plistValue struct {
	kind string
	text string
	dict map[string]plistValue
}

// parsePlistDict extracts the top-level dict's string values from an XML plist.
func parsePlistDict(data []byte) (map[string]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	for {
		tok, err := nextPlistToken(dec)
		if err != nil {
			return nil, err
		}

		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "plist" {
			if err := skipPlistElement(dec, start); err != nil {
				return nil, err
			}

			continue
		}

		value, err := parsePlistValue(dec)
		if err != nil {
			return nil, err
		}
		if value.kind != plistKindDict {
			return nil, errPlistTopLevel
		}

		props := make(map[string]string, len(value.dict))
		for key, item := range value.dict {
			if item.kind == plistKindString {
				props[key] = item.text
			}
		}

		return props, nil
	}
}

// parsePlistValue reads the next value element.
func parsePlistValue(dec *xml.Decoder) (plistValue, error) {
	for {
		tok, err := nextPlistToken(dec)
		if err != nil {
			return plistValue{}, err
		}

		switch typed := tok.(type) {
		case xml.StartElement:
			return parsePlistElement(dec, typed)
		case xml.EndElement:
			return plistValue{}, fmt.Errorf("%w: </%s>", errPlistUnexpectedEnd, typed.Name.Local)
		}
	}
}

// parsePlistElement parses one already-opened plist element.
func parsePlistElement(dec *xml.Decoder, start xml.StartElement) (plistValue, error) {
	switch start.Name.Local {
	case plistKindDict:
		return parsePlistDictElement(dec)
	case plistKindArray:
		return parsePlistArray(dec)
	case "string", "integer", "real", "date", "data", "key":
		text, err := decodePlistText(dec, start)
		if err != nil {
			return plistValue{}, err
		}

		return plistValue{kind: plistKindString, text: text}, nil
	case "true", "false":
		if err := skipPlistElement(dec, start); err != nil {
			return plistValue{}, err
		}

		return plistValue{kind: start.Name.Local}, nil
	default:
		if err := skipPlistElement(dec, start); err != nil {
			return plistValue{}, err
		}

		return plistValue{kind: "other"}, nil
	}
}

// parsePlistDictElement parses a dict's key/value pairs.
func parsePlistDictElement(dec *xml.Decoder) (plistValue, error) {
	dict := make(map[string]plistValue)
	var key string

	for {
		tok, err := nextPlistToken(dec)
		if err != nil {
			return plistValue{}, err
		}

		switch typed := tok.(type) {
		case xml.StartElement:
			if typed.Name.Local == "key" {
				key, err = decodePlistText(dec, typed)
				if err != nil {
					return plistValue{}, err
				}

				continue
			}
			if key == "" {
				if err := skipPlistElement(dec, typed); err != nil {
					return plistValue{}, err
				}

				continue
			}

			value, err := parsePlistElement(dec, typed)
			if err != nil {
				return plistValue{}, err
			}
			dict[key] = value
			key = ""
		case xml.EndElement:
			if typed.Name.Local == plistKindDict {
				return plistValue{kind: plistKindDict, dict: dict}, nil
			}
		}
	}
}

// parsePlistArray parses array elements (values are accepted and discarded).
func parsePlistArray(dec *xml.Decoder) (plistValue, error) {
	for {
		tok, err := nextPlistToken(dec)
		if err != nil {
			return plistValue{}, err
		}

		switch typed := tok.(type) {
		case xml.StartElement:
			if _, err := parsePlistElement(dec, typed); err != nil {
				return plistValue{}, err
			}
		case xml.EndElement:
			if typed.Name.Local == plistKindArray {
				return plistValue{kind: plistKindArray}, nil
			}
		}
	}
}

// nextPlistToken reads one decoder token and wraps its error.
func nextPlistToken(dec *xml.Decoder) (xml.Token, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("read plist token: %w", err)
	}

	return tok, nil
}

// skipPlistElement skips the current element subtree.
func skipPlistElement(dec *xml.Decoder, start xml.StartElement) error {
	if err := dec.Skip(); err != nil {
		return fmt.Errorf("skip plist element %s: %w", start.Name.Local, err)
	}

	return nil
}

// decodePlistText decodes a text-only element.
func decodePlistText(dec *xml.Decoder, start xml.StartElement) (string, error) {
	var text string
	if err := dec.DecodeElement(&text, &start); err != nil {
		return "", fmt.Errorf("decode plist element %s: %w", start.Name.Local, err)
	}

	return text, nil
}

// defaultMacApps is the production standard-location scan. It returns nil on
// non-Darwin hosts so Linux discovery cannot adopt an unrelated directory.
func defaultMacApps(home string) func() []string {
	if runtime.GOOS != goosDarwin {
		return nil
	}

	return func() []string {
		dirs := []string{"/Applications"}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "Applications"))
		}

		return scanMacApps(dirs)
	}
}

// scanMacApps finds ZCode*.app bundles in the given application dirs.
func scanMacApps(dirs []string) []string {
	seen := make(map[string]struct{})
	apps := make([]string, 0, len(dirs))

	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		apps = append(apps, path)
	}

	for _, dir := range dirs {
		matches, _ := filepath.Glob(filepath.Join(dir, "ZCode*.app"))
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.IsDir() {
				add(match)
			}
		}
	}

	return apps
}

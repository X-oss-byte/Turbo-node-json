package lockfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/vercel/turborepo/cli/internal/turbopath"
	"gopkg.in/yaml.v3"
)

// YarnLockfileEntry package information from yarn lockfile
type YarnLockfileEntry struct {
	// resolved version for the particular entry based on the provided semver revision
	Version   string `yaml:"version"`
	Resolved  string `yaml:"resolved"`
	Integrity string `yaml:"integrity"`
	// the list of unresolved modules and revisions (e.g. type-detect : ^4.0.0)
	Dependencies map[string]string `yaml:"dependencies,omitempty"`
	// the list of unresolved modules and revisions (e.g. type-detect : ^4.0.0)
	OptionalDependencies map[string]string `yaml:"optionalDependencies,omitempty"`
}

// BerryLockfile representation of berry lockfile
type BerryLockfile map[string]*YarnLockfileEntry

var _ Lockfile = (*BerryLockfile)(nil)

// ResolvePackage Given a package and version returns the key, resolved version, and if it was found
func (l *BerryLockfile) ResolvePackage(name string, version string) (string, string, bool) {
	for _, key := range yarnPossibleKeys(name, version) {
		if entry, ok := (*l)[key]; ok {
			return key, entry.Version, true
		}
	}

	return "", "", false
}

// AllDependencies Given a lockfile key return all (dev/optional/peer) dependencies of that package
func (l *BerryLockfile) AllDependencies(key string) (map[string]string, bool) {
	deps := map[string]string{}
	entry, ok := (*l)[key]
	if !ok {
		return deps, false
	}

	for name, version := range entry.Dependencies {
		deps[name] = version
	}
	for name, version := range entry.OptionalDependencies {
		deps[name] = version
	}

	return deps, true
}

// Subgraph Given a list of lockfile keys returns a Lockfile based off the original one that only contains the packages given
func (l *BerryLockfile) Subgraph(_ []turbopath.AnchoredSystemPath, packages []string) (Lockfile, error) {
	lockfile := make(BerryLockfile, len(packages))
	for _, key := range packages {
		entry, ok := (*l)[key]
		if ok {
			lockfile[key] = entry
		}
	}

	return &lockfile, nil
}

// Encode encode the lockfile representation and write it to the given writer
func (l *BerryLockfile) Encode(w io.Writer) error {
	var tmp bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&tmp)
	yamlEncoder.SetIndent(2)
	if err := yamlEncoder.Encode(l); err != nil {
		return errors.Wrap(err, "failed to materialize sub-lockfile. This can happen if your lockfile contains merge conflicts or is somehow corrupted. Please report this if it occurs")
	}

	if _, err := io.WriteString(w, "# This file is generated by running \"yarn install\" inside your project.\n# Manual changes might be lost - proceed with caution!\n\n__metadata:\n  version: 5\n  cacheKey: 8\n\n"); err != nil {
		return errors.Wrap(err, "failed to write to buffer")
	}

	// because of yarn being yarn, we need to inject lines in between each block of YAML to make it "valid" SYML
	scan := bufio.NewScanner(&tmp)
	buf := make([]byte, 0, 1024*1024)
	scan.Buffer(buf, 10*1024*1024)
	for scan.Scan() {
		line := scan.Text() //Writing to Stdout
		var stringToWrite string
		if !strings.HasPrefix(line, " ") {
			stringToWrite = fmt.Sprintf("\n%v\n", strings.ReplaceAll(line, "'", "\""))
		} else {
			stringToWrite = fmt.Sprintf("%v\n", strings.ReplaceAll(line, "'", "\""))
		}

		if _, err := io.WriteString(w, stringToWrite); err != nil {
			return errors.Wrap(err, "failed to write to buffer")
		}
	}

	return nil
}

// Patches return a list of patches used in the lockfile
func (l *BerryLockfile) Patches() []turbopath.AnchoredUnixPath {
	return nil
}

// DecodeBerryLockfile Takes the contents of a berry lockfile and returns a struct representation
func DecodeBerryLockfile(contents []byte) (*BerryLockfile, error) {
	var lockfile map[string]*YarnLockfileEntry

	err := yaml.Unmarshal(contents, &lockfile)
	if err != nil {
		return &BerryLockfile{}, fmt.Errorf("could not unmarshal lockfile: %w", err)
	}

	prettyLockFile := BerryLockfile(yarnSplitOutEntries(lockfile))
	return &prettyLockFile, nil
}

func yarnSplitOutEntries(lockfile map[string]*YarnLockfileEntry) map[string]*YarnLockfileEntry {
	prettyLockfile := map[string]*YarnLockfileEntry{}
	// This final step is important, it splits any deps with multiple-resolutions
	// (e.g. "@babel/generator@^7.13.0, @babel/generator@^7.13.9":) into separate
	// entries in our map
	for key, val := range lockfile {
		for _, v := range strings.Split(key, ", ") {
			prettyLockfile[strings.TrimSpace(v)] = val
		}
	}
	return prettyLockfile
}

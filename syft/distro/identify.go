package distro

import (
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/anchore/syft/internal"

	"github.com/anchore/syft/internal/log"
	"github.com/anchore/syft/syft/source"
)

// returns a distro or nil
type parseFunc func(string) *Distro

type parseEntry struct {
	path string
	fn   parseFunc
}

var identityFiles = []parseEntry{
	{
		// most distros provide a link at this location
		path: "/etc/os-release",
		fn:   parseOsRelease,
	},
	{
		// standard location for rhel & debian distros
		path: "/usr/lib/os-release",
		fn:   parseOsRelease,
	},
	{
		// check for busybox (important to check this last since other distros contain the busybox binary)
		path: "/bin/busybox",
		fn:   parseBusyBox,
	},
}

// Identify parses distro-specific files to determine distro metadata like version and release.
func Identify(resolver source.FileResolver) *Distro {
	var distro *Distro

identifyLoop:
	for _, entry := range identityFiles {
		locations, err := resolver.FilesByPath(entry.path)
		if err != nil {
			log.Errorf("unable to get path locations from %s: %s", entry.path, err)
			break
		}

		if len(locations) == 0 {
			log.Debugf("No Refs found from path: %s", entry.path)
			continue
		}

		for _, location := range locations {
			contentReader, err := resolver.FileContentsByLocation(location)
			if err != nil {
				log.Debugf("unable to get contents from %s: %s", entry.path, err)
				continue
			}

			content, err := ioutil.ReadAll(contentReader)
			internal.CloseAndLogError(contentReader, location.VirtualPath)
			if err != nil {
				log.Errorf("unable to read %q: %+v", location.RealPath, err)
				break
			}

			if len(content) == 0 {
				log.Debugf("no contents in file, skipping: %s", entry.path)
				continue
			}

			if candidateDistro := entry.fn(string(content)); candidateDistro != nil {
				distro = candidateDistro
				break identifyLoop
			}
		}
	}

	if distro != nil && distro.Type == UnknownDistroType {
		return nil
	}

	return distro
}

func assemble(name, version, like string) *Distro {
	distroType, ok := IDMapping[name]

	// Both distro and version must be present
	if len(name) == 0 && len(version) == 0 {
		return nil
	}

	// If it's an unknown distro, try mapping the ID_LIKE
	if !ok && len(like) != 0 {
		distroType, ok = IDMapping[like]
	}

	if ok {
		distro, err := NewDistro(distroType, version, like)
		if err != nil {
			return nil
		}
		return &distro
	}

	return nil
}

func parseOsRelease(contents string) *Distro {
	id, vers, like := "", "", ""
	for _, line := range strings.Split(contents, "\n") {
		parts := strings.Split(line, "=")
		prefix := parts[0]
		value := strings.ReplaceAll(parts[len(parts)-1], `"`, "")

		switch prefix {
		case "ID":
			id = strings.TrimSpace(value)
		case "VERSION_ID":
			vers = strings.TrimSpace(value)
		case "ID_LIKE":
			like = strings.TrimSpace(value)
		}
	}

	return assemble(id, vers, like)
}

var busyboxVersionMatcher = regexp.MustCompile(`BusyBox v[\d.]+`)

func parseBusyBox(contents string) *Distro {
	matches := busyboxVersionMatcher.FindAllString(contents, -1)
	for _, match := range matches {
		parts := strings.Split(match, " ")
		version := strings.ReplaceAll(parts[1], "v", "")
		distro := assemble("busybox", version, "")
		if distro != nil {
			return distro
		}
	}
	return nil
}

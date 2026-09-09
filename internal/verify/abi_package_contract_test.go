package verify

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func abiPackageDependencySurface(relative string, source []byte) (string, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(source, &document); err != nil {
		return "", err
	}
	if relative == "abi/package.json" {
		if err := filterABIManifest(document); err != nil {
			return "", err
		}
	} else {
		var version int
		if err := json.Unmarshal(document["lockfileVersion"], &version); err != nil || version != 3 {
			return "", fmt.Errorf("ABI lockfile requires schema 3")
		}
		var packages map[string]map[string]json.RawMessage
		if err := json.Unmarshal(document["packages"], &packages); err != nil {
			return "", err
		}
		if packages[""] == nil {
			return "", fmt.Errorf("ABI lockfile root package is absent")
		}
		if err := filterABIManifest(packages[""]); err != nil {
			return "", err
		}
		for _, name := range []string{"source-core", "target-api", "tsts"} {
			shared := "@" + "tso" + "nic/" + name
			path := "../../tsonic/packages/" + name
			if record := packages[path]; record != nil {
				if string(record["name"]) != fmt.Sprintf("%q", shared) || string(record["dev"]) != "true" {
					return "", fmt.Errorf("ABI shared lock entry %s is not a test dependency", name)
				}
				delete(record, "name")
				if err := filterABISharedDependencies(record, "dependencies", false); err != nil {
					return "", err
				}
			}
			key := "node_modules/" + shared
			if record := packages[key]; record != nil {
				if len(record) != 2 || string(record["link"]) != "true" || string(record["resolved"]) != fmt.Sprintf("%q", path) || packages[path] == nil {
					return "", fmt.Errorf("ABI shared lock link %s does not select its test dependency", name)
				}
				delete(packages, key)
			}
		}
		encoded, err := json.Marshal(packages)
		if err != nil {
			return "", err
		}
		document["packages"] = encoded
	}
	encoded, err := json.Marshal(document)
	return string(encoded), err
}

func filterABIManifest(document map[string]json.RawMessage) error {
	if err := filterABISharedDependencies(document, "peerDependencies", false); err != nil {
		return err
	}
	return filterABISharedDependencies(document, "devDependencies", true)
}

func filterABISharedDependencies(document map[string]json.RawMessage, field string, development bool) error {
	if document[field] == nil {
		return nil
	}
	var dependencies map[string]string
	if err := json.Unmarshal(document[field], &dependencies); err != nil {
		return err
	}
	shared := "@" + "tso" + "nic/"
	for name, selection := range dependencies {
		if !strings.HasPrefix(name, shared) {
			if field == "peerDependencies" {
				return fmt.Errorf("ABI adapter has an unapproved peer %s", name)
			}
			continue
		}
		owner := strings.TrimPrefix(name, shared)
		if owner != "source-core" && owner != "target-api" && owner != "tsts" {
			return fmt.Errorf("ABI adapter has an unapproved shared dependency %s", name)
		}
		if development {
			if selection != "file:../../tsonic/packages/"+owner {
				return fmt.Errorf("ABI adapter test dependency %s has a different owner", name)
			}
		} else if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(selection) {
			return fmt.Errorf("ABI adapter peer %s is not exact", name)
		}
		delete(dependencies, name)
	}
	encoded, err := json.Marshal(dependencies)
	if err != nil {
		return err
	}
	document[field] = encoded
	return nil
}

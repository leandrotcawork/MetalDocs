package mddm

import "fmt"

// EnvelopeMigration transforms an envelope from version N to version N+1.
type EnvelopeMigration func(envelope map[string]any) (map[string]any, error)

// migrations maps source version → migration function.
// Add new entries when bumping mddm_version.
var migrations = map[int]EnvelopeMigration{
	// 1: migrateV1toV2,  // example for the future
}

// MigrateEnvelopeForward applies all migrations to bring envelope to targetVersion.
func MigrateEnvelopeForward(envelope map[string]any, targetVersion int) (map[string]any, error) {
	versionRaw, ok := envelope["mddm_version"]
	if !ok {
		return nil, fmt.Errorf("envelope missing mddm_version")
	}
	currentFloat, ok := versionRaw.(float64) // JSON-parsed numbers
	if !ok {
		if intVer, intOk := versionRaw.(int); intOk {
			currentFloat = float64(intVer)
		} else {
			return nil, fmt.Errorf("mddm_version is not numeric: %T", versionRaw)
		}
	}
	current := int(currentFloat)

	if current > targetVersion {
		return nil, fmt.Errorf("envelope mddm_version %d is newer than supported %d", current, targetVersion)
	}
	if current < 1 {
		return nil, fmt.Errorf("invalid mddm_version: %d", current)
	}

	for v := current; v < targetVersion; v++ {
		migration, exists := migrations[v]
		if !exists {
			return nil, fmt.Errorf("missing migration from v%d to v%d", v, v+1)
		}
		next, err := migration(envelope)
		if err != nil {
			return nil, fmt.Errorf("migration v%d→v%d failed: %w", v, v+1, err)
		}
		envelope = next
		envelope["mddm_version"] = v + 1
	}

	return envelope, nil
}

const CurrentMDDMVersion = 1

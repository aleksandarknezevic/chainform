package abi

import ethabi "github.com/ethereum/go-ethereum/accounts/abi"

// TogglePair is a zero-argument on/off method pair for a bool getter, e.g.
// pause()/unpause() for the paused() getter (OpenZeppelin Pausable).
type TogglePair struct {
	On  string // call when desired true, actual false — e.g. "pause"
	Off string // call when desired false, actual true — e.g. "unpause"
}

// knownBoolToggles maps bool getter names to their conventional toggle pairs.
var knownBoolToggles = map[string]TogglePair{
	"paused": {On: "pause", Off: "unpause"},
}

// BoolTogglePairs returns toggle pairs present in the ABI for known bool getters.
// When a pair is found, planning should prefer it over a setX(bool) setter.
func BoolTogglePairs(parsed *ethabi.ABI) map[string]TogglePair {
	out := make(map[string]TogglePair)
	for getter, pair := range knownBoolToggles {
		if hasZeroArgMutator(parsed, pair.On) && hasZeroArgMutator(parsed, pair.Off) {
			out[getter] = pair
		}
	}
	return out
}

func hasZeroArgMutator(parsed *ethabi.ABI, name string) bool {
	m, ok := parsed.Methods[name]
	if !ok {
		return false
	}
	return !m.IsConstant() && len(m.Inputs) == 0
}

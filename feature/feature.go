package feature

import (
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

const (
	// MachinePool is used to enable instance pool support
	// alpha: v0.1
	MachinePool featuregate.Feature = "MachinePool"
)

func init() {
	runtime.Must(MutableGates.Add(defaultCAPAFeatureGates))
}

// defaultCAPAFeatureGates consists of all known capa-specific feature keys.
// To add a new feature, define a key for it above and add it here.
var defaultCAPAFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	// Every feature should be initiated here:
	MachinePool: {Default: false, PreRelease: featuregate.Alpha},
}

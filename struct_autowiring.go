package di

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

// EnvEnableStructAutoWiring enables implicit struct auto-wiring for newly created containers.
// Existing containers keep their current setting.
const EnvEnableStructAutoWiring = "DI_ENABLE_STRUCT_AUTOWIRING"

func defaultStructAutoWiringEnabled() bool {
	raw, ok := os.LookupEnv(EnvEnableStructAutoWiring)
	if !ok {
		return false
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return enabled
}

func shouldAutoWireStruct(c *Container, dep Dependency) bool {
	return c != nil &&
		c.StructAutoWiringEnabled() &&
		dep.Name == "" &&
		dep.Type != nil &&
		dep.Type.Kind() == reflect.Struct
}

// SetStructAutoWiring enables or disables implicit struct auto-wiring for this container.
func (c *Container) SetStructAutoWiring(enabled bool) {
	if c == nil {
		return
	}
	c.runtimeState.StructAutoWire.Store(enabled)
}

// StructAutoWiringEnabled reports whether implicit struct auto-wiring is enabled for this container.
func (c *Container) StructAutoWiringEnabled() bool {
	if c == nil {
		return false
	}
	return c.runtimeState.StructAutoWire.Load()
}

// SetStructAutoWiring enables or disables implicit struct auto-wiring for this scope.
func (s *Scope) SetStructAutoWiring(enabled bool) {
	if s == nil || s.container == nil {
		return
	}
	s.container.SetStructAutoWiring(enabled)
}

// StructAutoWiringEnabled reports whether implicit struct auto-wiring is enabled for this scope.
func (s *Scope) StructAutoWiringEnabled() bool {
	if s == nil || s.container == nil {
		return false
	}
	return s.container.StructAutoWiringEnabled()
}

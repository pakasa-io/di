package di

import (
	"reflect"
	"sync/atomic"
	"time"
)

// Instrumentation contains optional callbacks invoked during dependency resolution.
// Callbacks run inline after the observed operation completes.
type Instrumentation struct {
	OnResolve         func(ResolveEvent)
	OnInstanceCreated func(InstanceEvent)
}

// ResolveEvent describes a dependency resolution attempt.
type ResolveEvent struct {
	Key       string
	Type      reflect.Type
	Name      string
	Duration  time.Duration
	Err       error
	Override  bool
	Container *Container
	Owner     *Container
}

// InstanceEvent describes an instance creation attempt.
type InstanceEvent struct {
	Key       string
	Type      reflect.Type
	Name      string
	Lifetime  Lifetime
	Duration  time.Duration
	Err       error
	Container *Container
}

// ContainerMetrics is a cumulative snapshot of container activity.
type ContainerMetrics struct {
	Resolutions            uint64
	ResolutionErrors       uint64
	ResolutionDuration     time.Duration
	InstancesCreated       uint64
	InstanceCreationErrors uint64
	InstanceCreationTime   time.Duration
	OverrideCalls          uint64
}

type containerMetrics struct {
	resolutions      atomic.Uint64
	resolutionErrors atomic.Uint64
	resolutionNanos  atomic.Uint64
	instancesCreated atomic.Uint64
	instanceErrors   atomic.Uint64
	instanceNanos    atomic.Uint64
	overrideCalls    atomic.Uint64
}

// SetInstrumentation installs instrumentation callbacks on this container.
func (c *Container) SetInstrumentation(instrumentation Instrumentation) {
	c.runtimeState.Instrumentation.Store(instrumentation)
}

// Metrics returns a cumulative metrics snapshot for this container.
func (c *Container) Metrics() ContainerMetrics {
	return ContainerMetrics{
		Resolutions:            c.runtimeState.Metrics.resolutions.Load(),
		ResolutionErrors:       c.runtimeState.Metrics.resolutionErrors.Load(),
		ResolutionDuration:     time.Duration(c.runtimeState.Metrics.resolutionNanos.Load()),
		InstancesCreated:       c.runtimeState.Metrics.instancesCreated.Load(),
		InstanceCreationErrors: c.runtimeState.Metrics.instanceErrors.Load(),
		InstanceCreationTime:   time.Duration(c.runtimeState.Metrics.instanceNanos.Load()),
		OverrideCalls:          c.runtimeState.Metrics.overrideCalls.Load(),
	}
}

// ResetMetrics resets all cumulative metrics for this container.
func (c *Container) ResetMetrics() {
	c.runtimeState.Metrics.resolutions.Store(0)
	c.runtimeState.Metrics.resolutionErrors.Store(0)
	c.runtimeState.Metrics.resolutionNanos.Store(0)
	c.runtimeState.Metrics.instancesCreated.Store(0)
	c.runtimeState.Metrics.instanceErrors.Store(0)
	c.runtimeState.Metrics.instanceNanos.Store(0)
	c.runtimeState.Metrics.overrideCalls.Store(0)
}

func (c *Container) currentInstrumentation() Instrumentation {
	if c == nil {
		return Instrumentation{}
	}
	value := c.runtimeState.Instrumentation.Load()
	if value == nil {
		return Instrumentation{}
	}
	return value.(Instrumentation)
}

func (c *Container) recordResolve(event ResolveEvent) {
	if c == nil {
		return
	}

	c.runtimeState.Metrics.resolutions.Add(1)
	c.runtimeState.Metrics.resolutionNanos.Add(uint64(event.Duration))
	if event.Err != nil {
		c.runtimeState.Metrics.resolutionErrors.Add(1)
	}
	if event.Override {
		c.runtimeState.Metrics.overrideCalls.Add(1)
	}

	if callback := c.currentInstrumentation().OnResolve; callback != nil {
		callback(event)
	}
}

func (c *Container) recordInstance(event InstanceEvent) {
	if c == nil {
		return
	}

	c.runtimeState.Metrics.instanceNanos.Add(uint64(event.Duration))
	if event.Err != nil {
		c.runtimeState.Metrics.instanceErrors.Add(1)
	} else {
		c.runtimeState.Metrics.instancesCreated.Add(1)
	}

	if callback := c.currentInstrumentation().OnInstanceCreated; callback != nil {
		callback(event)
	}
}

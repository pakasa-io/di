package di

import diModel "github.com/pakasa-io/di/internal/model"

// PreConstruct is implemented by values that want a pre-construction hook.
type PreConstruct = diModel.PreConstruct

// PostConstruct is implemented by values that want a post-construction hook.
type PostConstruct = diModel.PostConstruct

// AutoCloseable is implemented by values that should be closed automatically.
type AutoCloseable = diModel.AutoCloseable

// CloseFunc is a close hook registered on a binding.
type CloseFunc = diModel.CloseFunc

// LifecycleHooks holds binding lifecycle hooks.
type LifecycleHooks = diModel.LifecycleHooks

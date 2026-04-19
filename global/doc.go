// Package global provides process-wide default and named container helpers on
// top of the root [github.com/pakasa-io/di] package.
//
// Use this package when an application naturally has one default container, or
// a small set of process-wide named containers, and explicit container
// ownership would add more ceremony than clarity.
//
// For libraries and most tests, prefer the explicit [github.com/pakasa-io/di]
// container APIs.
package global

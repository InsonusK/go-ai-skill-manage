// Package sourcing caches acquired repositories by identity and owns their
// cleanup lifecycle, depending only on the domain-declared SourceProvider
// port -- never on os/exec, net/http, or any concrete infrastructure type.
package sourcing

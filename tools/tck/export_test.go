package tck

// The external self-tests drive the same in-memory backend as
// TestInMemoryProvider, so they share its control and its capability list
// rather than restating them.
type PlainMemoryControl = plainMemoryControl

var InMemoryCapabilities = inMemoryCapabilities

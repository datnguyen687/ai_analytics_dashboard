package service

// Chart series colors are emitted as CSS custom-property references so the
// frontend renders them with its own light/dark theme tokens.
const (
	colorPrimary  = "var(--series-1)"
	colorAccent   = "var(--series-2)"
	colorSuccess  = "var(--status-good)"
	colorDanger   = "var(--status-critical)"
)

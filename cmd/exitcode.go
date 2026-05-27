package cmd

// exitCode signals a non-zero process exit without printing an error message.
type exitCode int

func (e exitCode) Error() string { return "" }

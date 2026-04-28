package generator

// WrapperClassNameForTest exposes wrapperClassName to the external test
// package without widening the public API.
func WrapperClassNameForTest(protoFile string) string {
	return wrapperClassName(protoFile)
}

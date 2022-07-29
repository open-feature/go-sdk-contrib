package tests

// error comparison to reduce repeated code (errors.Is will fail)
func errorCompare(err1 error, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 != nil && err2 == nil {
		return false
	}
	if err1 == nil && err2 != nil {
		return false
	}
	if err1.Error() != err2.Error() {
		return false
	}
	return true
}

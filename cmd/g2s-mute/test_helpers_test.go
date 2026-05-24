package main

func containsString(values []string, target string) bool {
<<<<<<< HEAD
	for _, value := range values {
		if value == target {
=======
	for _, item := range values {
		if item == target {
>>>>>>> 9054deb (Remove blocker-policy enforcement and API surface)
			return true
		}
	}
	return false
}

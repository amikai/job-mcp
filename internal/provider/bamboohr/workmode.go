package bamboohr

// WorkModeLabel translates the API's numeric-string locationType ("0"
// on-site, "1" remote, "2" hybrid — see openapi.yaml) into a display label.
// Unknown or null codes return "".
func WorkModeLabel(locationType string) string {
	switch locationType {
	case "0":
		return "On-site"
	case "1":
		return "Remote"
	case "2":
		return "Hybrid"
	}
	return ""
}

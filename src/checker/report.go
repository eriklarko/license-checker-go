package checker

type Report struct {
	Allowed    map[string][]string
	Disallowed map[string][]string

	Unknown map[string][]string
}

func (r *Report) RecordDecision(license string, dependency string, allowed bool) {
	if allowed {
		r.RecordAllowed(license, dependency)
	} else {
		r.RecordDisallowed(license, dependency)
	}
}

// RecordAllowed records that a license is allowed
func (r *Report) RecordAllowed(license string, dependency string) {
	if r.Allowed == nil {
		r.Allowed = make(map[string][]string)
	}
	r.Allowed[license] = append(r.Allowed[license], dependency)
}

// RecordDisallowed records that a license is disallowed
func (r *Report) RecordDisallowed(license string, dependency string) {
	if r.Disallowed == nil {
		r.Disallowed = make(map[string][]string)
	}
	r.Disallowed[license] = append(r.Disallowed[license], dependency)
}

func (r *Report) HasDisallowedLicenses() bool {
	return len(r.Disallowed) > 0
}

func (r *Report) RecordUnknownLicense(license string, dependency string) {
	if r.Unknown == nil {
		r.Unknown = make(map[string][]string)
	}
	r.Unknown[license] = append(r.Unknown[license], dependency)
}

func (r *Report) HasUnknownLicenses() bool {
	return len(r.Unknown) > 0
}

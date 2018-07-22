package di

// App is the name of the application scope.
const App = "app"

// Request is the name of the request scope.
const Request = "request"

// SubRequest is the name of the subrequest scope.
const SubRequest = "subrequest"

// ScopeList is a slice of scope.
type ScopeList []string

// Copy returns a copy of the ScopeList.
func (l ScopeList) Copy() ScopeList {
	scopes := make(ScopeList, len(l))
	copy(scopes, l)
	return scopes
}

// ParentScopes returns the scopes before the one given as parameter.
func (l ScopeList) ParentScopes(scope string) ScopeList {
	scopes := l.Copy()

	for i, s := range scopes {
		if s == scope {
			return scopes[:i]
		}
	}

	return ScopeList{}
}

// SubScopes returns the scopes after the one given as parameter.
func (l ScopeList) SubScopes(scope string) ScopeList {
	scopes := l.Copy()

	for i, s := range scopes {
		if s == scope {
			return scopes[i+1:]
		}
	}

	return ScopeList{}
}

// Contains returns true if the ScopeList contains the given scope.
func (l ScopeList) Contains(scope string) bool {
	for _, s := range l {
		if scope == s {
			return true
		}
	}

	return false
}

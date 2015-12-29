package di

import "testing"

func TestNewContextManagerErrors(t *testing.T) {
	if _, err := NewContextManager(); err == nil {
		t.Error("should not be able to create a ContextManager without any scope")
	}

	if _, err := NewContextManager("app", ""); err == nil {
		t.Error("should not be able to create a ContextManager with an empty scope")
	}

	if _, err := NewContextManager("app", "request", "app", "subrequest"); err == nil {
		t.Error("should not be able to create a ContextManager with two identical scopes")
	}
}

func TestResolveName(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Instance(Instance{
		Name:    "instance",
		Aliases: []string{"instance1", "i"},
	})

	cm.Maker(Maker{
		Name:    "maker",
		Aliases: []string{"maker1", "m"},
		Scope:   "request",
	})

	tests := map[string]string{
		"instance":  "instance",
		"instance1": "instance",
		"i":         "instance",
		"maker":     "maker",
		"maker1":    "maker",
		"m":         "maker",
		"MAKER":     "",
	}

	for input, expected := range tests {
		if output, _ := cm.ResolveName(input); output != expected {
			t.Errorf("could not resolve name, expected `%s`, returned `%s`", expected, output)
		}
	}
}

func TestNameIsUsed(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	cm.Instance(Instance{
		Name:    "instance",
		Aliases: []string{"instance1", "i"},
	})

	cm.Maker(Maker{
		Name:    "maker",
		Aliases: []string{"maker1", "m"},
		Scope:   "request",
	})

	tests := map[string]bool{
		"instance":  true,
		"instance1": true,
		"i":         true,
		"maker":     true,
		"maker1":    true,
		"m":         true,
		"MAKER":     false,
	}

	for input, expected := range tests {
		if output := cm.NameIsUsed(input); output != expected {
			t.Errorf("NameIsUsed failed for `%s`, expected `%t`, returned `%t`", input, expected, output)
		}
	}
}

func TestMakerRegistrationErrors(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	if err := cm.Maker(Maker{Name: "maker", Aliases: []string{"m"}, Scope: "request"}); err != nil {
		t.Errorf("unexpected error during the first Maker registration `%s`", err)
	}

	if cm.Maker(Maker{Name: "maker2", Scope: "undefined"}) == nil {
		t.Error("should not be able to register a Maker in an undefined scope")
	}

	if cm.Maker(Maker{Name: "maker", Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if the name is already used")
	}

	if cm.Maker(Maker{Name: "", Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if the name is empty")
	}

	if cm.Maker(Maker{Name: "maker2", Aliases: []string{""}, Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if an alias is empty")
	}

	if cm.Maker(Maker{Name: "maker2", Aliases: []string{"maker2"}, Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if an alias is equal to the name")
	}

	if cm.Maker(Maker{Name: "maker2", Aliases: []string{"a", "a"}, Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if two aliases are identical")
	}

	if cm.Maker(Maker{Name: "maker2", Aliases: []string{"maker"}, Scope: "subrequest"}) == nil {
		t.Error("should not be able to register a Maker if an alias is already used")
	}

	if err := cm.Maker(Maker{Name: "maker2", Aliases: []string{"m2"}, Scope: "request"}); err != nil {
		t.Errorf("unexpected error during the second Maker registration `%s`", err)
	}
}

func TestInstanceRegistrationErrors(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	if err := cm.Instance(Instance{Name: "instance", Aliases: []string{"i"}}); err != nil {
		t.Errorf("unexpected error during the first Instance registration `%s`", err)
	}

	if cm.Instance(Instance{Name: "instance"}) == nil {
		t.Error("should not be able to register a Instance if the name is already used")
	}

	if cm.Instance(Instance{Name: ""}) == nil {
		t.Error("should not be able to register a Instance if the name is empty")
	}

	if cm.Instance(Instance{Name: "instance2", Aliases: []string{"instance"}}) == nil {
		t.Error("should not be able to register a Instance if an alias is already used")
	}

	if err := cm.Instance(Instance{Name: "instance2", Aliases: []string{"i2"}}); err != nil {
		t.Errorf("unexpected error during the second Instance registration `%s`", err)
	}
}

func TestContextCeation(t *testing.T) {
	cm, _ := NewContextManager("app", "request", "subrequest")

	if _, err := cm.Context("undefined"); err == nil {
		t.Errorf("should not be able to create a context in an undefined scope")
	}

	app, err := cm.Context("app")
	if err != nil {
		t.Errorf("should be able to create an app Context, error = `%s`", err)
	}
	if app.ContextManager() != cm || app.Scope() != "app" {
		t.Errorf("app Context is not well defined %+v", app)
	}

	subrequest, err := cm.Context("subrequest")
	if err != nil {
		t.Errorf("should be able to create a subrequest Context, error = `%s`", err)
	}
	if subrequest.ContextManager() != cm || subrequest.Scope() != "subrequest" {
		t.Errorf("subrequest Context is not well defined %+v", app)
	}
	if subrequest.Parent().Scope() != "request" || subrequest.Parent().Parent().Scope() != "app" {
		t.Error("subrequest should be a child of a request Context and a grandchild of an app Context")
	}
	if subrequest.Parent().Parent() == app {
		t.Error("subrequest should not use the same app Context")
	}
}

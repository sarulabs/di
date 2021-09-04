package di

import (
	"fmt"
)

// buildingChan is used internally as the value of an object while it is being built.
type buildingChan chan struct{}

// buildObject wraps the Build function to recover from a panic.
func buildObject(
	buildFunc func(ctn Container) (interface{}, error),
	ctn Container,
	index int,
	defName string,
) (obj interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not build `%s` because the build function panicked: %+v", defName, r)
		}
	}()

	ctn.builtList = append(ctn.builtList, index)

	return buildFunc(ctn)
}

// formatBuiltOnClosedContainerError formats the error that happens when you try to build an object with a closed container.
func formatBuiltOnClosedContainerError(def Def, closeObjectErr error) error {
	formattedCloseObjectErr := ""
	if closeObjectErr != nil {
		formattedCloseObjectErr = fmt.Sprintf(" (with an error: %+v)", closeObjectErr)
	}

	return fmt.Errorf(
		"could not get `%s` because the container has been deleted, the object has been created and closed%s",
		def.Name,
		formattedCloseObjectErr,
	)
}

// formatCycleError formats the error that happens when a cycle is detected.
func formatCycleError(ctn Container, def Def) error {
	cycle := []string{}

	for _, i := range ctn.builtList {
		cycle = append(cycle, ctn.core.definitions[i].Name)
	}

	cycle = append(cycle, def.Name)

	return fmt.Errorf(
		"could not get `%s` because there is a cycle in the object definitions (%v)",
		def.Name,
		cycle,
	)
}

// Fill is similar to SafeGet but it does not return the object.
// Instead it fills the provided object with the value returned by SafeGet.
// The provided object must be a pointer to the value returned by SafeGet.
// It uses reflection so it is slower than Get and SafeGet.
// But it can be convenient in some cases where performance is not a critical factor.
func (ctn Container) Fill(in interface{}, dst interface{}) error {
	obj, err := ctn.SafeGet(in)
	if err != nil {
		return err
	}
	return fill(obj, dst)
}

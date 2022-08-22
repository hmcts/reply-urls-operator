package azureGraph

import "fmt"

type FieldNotFoundError struct {
	Field    string
	Resource string
}

func (err FieldNotFoundError) Error() string {
	return fmt.Sprintf("Field '%s' is missing please add to your %s resource", err.Field, err.Resource)
}

func (err *FieldNotFoundError) SetField(field string) {
	err.Field = field
}

func (err *FieldNotFoundError) SetResource(resource string) {
	err.Resource = resource
}

package v1alpha1

// Application is a minimal API type used to bootstrap protobuf generation.
type Application struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}

package localtypes

type BundleConfigMapSpec struct {
	// Name is the name of the ConfigMap containing the CA bundle
	Name string `json:"name"`
	// Key is the key in the ConfigMap containing the CA bundle
	Key string `json:"key"`
}

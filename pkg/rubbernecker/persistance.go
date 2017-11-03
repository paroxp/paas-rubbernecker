package rubbernecker

// PersistanceEngine interface should ensure any backing service will follow the
// same set of rules.
type PersistanceEngine interface {
	Get(name string) (interface{}, error)
	Put(name string, object interface{}) error
}

package api

type Placement interface {
	TypeImport(modulePath string, exportedName string) (localName string, err error)
}

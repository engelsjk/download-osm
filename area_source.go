package downloadosm

type AreaSource interface {
	Search(string) error
	GetCatalog() Catalog
	FetchCatalog() error
	ParseCatalog(string) Catalog
}

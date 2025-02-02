package homepage

type (
	Config   map[string]Category
	Category []*Item

	Item struct {
		Show         bool           `json:"show" yaml:"show"`
		Name         string         `json:"name" yaml:"name"`
		Icon         string         `json:"icon" yaml:"icon"`
		URL          string         `json:"url" yaml:"url"` // alias + domain
		Category     string         `json:"category" yaml:"category"`
		Description  string         `json:"description" yaml:"description"`
		WidgetConfig map[string]any `json:"widget_config" yaml:",flow"`

		SourceType string `json:"source_type" yaml:"-"`
		AltURL     string `json:"alt_url" yaml:"-"` // original proxy target
	}
)

func (item *Item) IsEmpty() bool {
	return item == nil || (item.Name == "" &&
		item.Icon == "" &&
		item.URL == "" &&
		item.Category == "" &&
		item.Description == "" &&
		len(item.WidgetConfig) == 0)
}

func NewHomePageConfig() Config {
	return Config(make(map[string]Category))
}

func (c *Config) Clear() {
	*c = make(Config)
}

func (c Config) Add(item *Item) {
	if c[item.Category] == nil {
		c[item.Category] = make(Category, 0)
	}
	c[item.Category] = append(c[item.Category], item)
}

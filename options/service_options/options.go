package service_options

// ====== register options ===================

// app内唯一的服务
func WithAsAppServiceOption() AsAppServiceOption {
	return AsAppServiceOption{}
}

// ====== where options ===================
// 最大数量
func WithWhereMaxCountOption(count int) WhereMaxCountOption {
	return WhereMaxCountOption{
		count: count,
	}
}

// 设置正则表达式查找
func WithWhereRegexOption() WhereRegexOption {
	return WhereRegexOption{}
}

// 前缀查找
func WithWherePrefixOption() WherePrefixOption {
	return WherePrefixOption{}
}

// 按目录查找
func WithWhereCatalogOption() WhereCatalogOption {
	return WhereCatalogOption{}
}

// ====== where options ===================
type WhereOptions struct {
	MaxCount int
	Regex    bool
	Prefix   bool
	Catalog  bool
}

type WhereOption interface {
	ConfigWhereOption(opts *WhereOptions)
}

type WhereMaxCountOption struct {
	count int
}

func (opt WhereMaxCountOption) ConfigWhereOption(opts *WhereOptions) {
	opts.MaxCount = opt.count
}

type WhereRegexOption struct {
}

func (opt WhereRegexOption) ConfigWhereOption(opts *WhereOptions) {
	opts.Regex = true
}

type WherePrefixOption struct {
}

func (opt WherePrefixOption) ConfigWhereOption(opts *WhereOptions) {
	opts.Prefix = true
}

type WhereCatalogOption struct {
}

func (opt WhereCatalogOption) ConfigWhereOption(opts *WhereOptions) {
	opts.Catalog = true
}

// ====== register options ===================

type RegisterOptions struct {
	AsAppService bool
}

type RegisterOption interface {
	ConfigRegisterOption(opts *RegisterOptions)
}

type AsAppServiceOption struct {
	enabled bool
}

func (opt AsAppServiceOption) ConfigRegisterOption(opts *RegisterOptions) {
	opts.AsAppService = true
}

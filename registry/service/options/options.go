package options

// where option
type WhereOptions struct {
	MaxCount int
	Regex    string
}

type WhereOption interface {
	ConfigWhereOption(opts *WhereOptions)
}

type WhereMaxCountOption struct {
	count int
}

func WithWhereMaxCountOption(count int) WhereMaxCountOption {
	return WhereMaxCountOption{
		count: count,
	}
}

func (opt WhereMaxCountOption) ConfigWhereOption(opts *WhereOptions) {
	opts.MaxCount = opt.count
}

type WhereRegexOption struct {
	regex string
}

func WithWhereRegexOption(regex string) WhereRegexOption {
	return WhereRegexOption{
		regex: regex,
	}
}

func (opt WhereRegexOption) ConfigWhereOption(opts *WhereOptions) {
	opts.Regex = opt.regex
}

type RegisterOptions struct {
	AsGroup  bool
	CatAppId bool
}

type RegisterOption interface {
	ConfigRegisterOption(opts *RegisterOptions)
}

type RegisterAsGroupOption struct {
	asGroup bool
}

func WithRegisterAsGroupOption(asGroup bool) RegisterAsGroupOption {
	return RegisterAsGroupOption{
		asGroup: asGroup,
	}
}

func (opt RegisterAsGroupOption) ConfigRegisterOption(opts *RegisterOptions) {
	opts.AsGroup = opt.asGroup
}

type RegisterCatAppidOption struct {
	catAppId bool
}

func WithRegisterCatAppIdOption(catAppId bool) RegisterCatAppidOption {
	return RegisterCatAppidOption{
		catAppId: catAppId,
	}
}

func (opt RegisterCatAppidOption) ConfigRegisterOption(opts *RegisterOptions) {
	opts.CatAppId = opt.catAppId
}

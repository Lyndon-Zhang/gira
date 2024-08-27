package resource_options

func WithPushDropOption() PushDropOption {
	return PushDropOption{}
}

func WithPushForceOption() PushForceOption {
	return PushForceOption{}
}

type PushOption interface {
	ConfigPushOption(opts *PushOptions)
}

type PushOptions struct {
	Drop  bool
	Force bool
}

type PushDropOption struct {
}

func (opt PushDropOption) ConfigPushOption(opts *PushOptions) {
	opts.Drop = true
}

type PushForceOption struct {
}

func (opt PushForceOption) ConfigPushOption(opts *PushOptions) {
	opts.Force = true
}

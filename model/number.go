package model

type Number struct {
	NumberId string `option:"mandatory"`
	IsCnSub  bool

	Episode      string
	IsUncensored bool
	Is4k         bool
	IsCracked    bool
	IsLeaked     bool
	Cat          Category
}

func (number *Number) WithNumberId(numberId string) *Number {
	number.NumberId = numberId
	return number
}

func (number *Number) WithIsCnSub(isCnSub bool) *Number {
	number.IsCnSub = isCnSub
	return number
}

func (number *Number) WithEpisode(_episode string) *Number {
	number.Episode = _episode
	return number
}

func (number *Number) WithIsUncensored(isUncensored bool) *Number {
	number.IsUncensored = isUncensored
	return number
}

func (number *Number) WithIs4k(is4k bool) *Number {
	number.Is4k = is4k
	return number
}

func (number *Number) WithIsCracked(isCracked bool) *Number {
	number.IsCracked = isCracked
	return number
}

func (number *Number) WithIsLeaked(isLeaked bool) *Number {
	number.IsLeaked = isLeaked
	return number
}

func (number *Number) WithCat(cat Category) *Number {
	number.Cat = cat
	return number
}

func (number *Number) Build() Number {
	return *number
}

func DefaultNumber() *Number {
	return &Number{}
}

// ----
func (n *Number) GetCategory() Category {
	return n.Cat
}

func (n *Number) GetNumberID() string {
	return n.NumberId
}

func (n *Number) GetIsChineseSubtitle() bool {
	return n.IsCnSub
}
func (n *Number) SetIsChineseSubtitle(is bool) {
	n.IsCnSub = is
}

func (n *Number) GetIsUncensorMovie() bool {
	return n.IsUncensored
}

func (n *Number) GetIs4K() bool {
	return n.Is4k
}

func (n *Number) GetIsLeak() bool {
	return n.IsLeaked
}

func (n *Number) GenerateSuffix(base string) string {
	if n.GetIs4K() {
		base += "-" + DefaultSuffix4K
	}
	if n.GetIsChineseSubtitle() {
		base += "-" + DefaultSuffixChineseSubtitle
	}
	if n.GetIsLeak() {
		base += "-" + DefaultSuffixLeak
	}
	// if n.GetIsMultiCD() {
	// 	base += "-" + DefaultSuffixMultiCD + n.GetEpisode()
	// }
	return base
}

func (n *Number) GenerateTags() []string {
	rs := make([]string, 0, 5)
	if n.GetIsUncensorMovie() {
		rs = append(rs, DefaultTagUncensored)
	}
	if n.GetIsChineseSubtitle() {
		rs = append(rs, DefaultTagChineseSubtitle)
	}
	if n.GetIs4K() {
		rs = append(rs, DefaultTag4K)
	}
	if n.GetIsLeak() {
		rs = append(rs, DefaultTagLeak)
	}
	return rs
}

func (n *Number) GenerateFileName() string {
	return n.GenerateSuffix(n.GetNumberID())
}

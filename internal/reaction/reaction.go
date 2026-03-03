package reaction

const (
	TypeHeart  = "heart"
	TypeHate   = "hate"
	TypeIgnore = "ignore"
)

func ValidType(value string) bool {
	switch value {
	case TypeHeart, TypeHate, TypeIgnore:
		return true
	default:
		return false
	}
}

package entities

type ShURL struct {
	Token     string
	LongURL   string
	CreatedBy string
}

const (
	ShURLTokenFieldName     = "Token"
	ShURLLongURLFieldName   = "LongURL"
	ShURLCreatedByFieldName = "CreatedBy"
)

func (su ShURL) GetID() string {
	return su.Token
}

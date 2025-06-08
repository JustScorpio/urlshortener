package entities

type ShURL struct {
	Token     string
	LongURL   string
	CreatedBy string
}

func (su ShURL) GetID() string {
	return su.Token
}

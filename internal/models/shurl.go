package models

type ShURL struct {
	Token   string
	LongURL string
}

func (su ShURL) GetID() string {
	return su.Token
}

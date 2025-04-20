package models

type ShURL struct {
	Token   string
	LongURL string
}

func (su ShURL) GetId() string {
	return su.Token
}

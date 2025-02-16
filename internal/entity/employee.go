package entity

type InfoResponse struct {
	Coins       int         `json:"coins"`
	Inventory   []Inventory `json:"inventory"`
	CoinHistory CoinHistory `json:"coinHistory"`
}

type Inventory struct {
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type CoinHistory struct {
	Received []Transaction `json:"received"`
	Sent     []Transaction `json:"sent"`
}

type Transaction struct {
	UserID int `json:"user"`
	Amount int `json:"amount"`
}

type Employee struct {
	ID       int
	Name     string
	Coins    int
	Password string
}

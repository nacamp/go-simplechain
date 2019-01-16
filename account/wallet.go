package account

type wallet interface {
	Accounts()
}
type Wallet struct {
	wallet wallet
	keyStore
}

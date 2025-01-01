package payment

type PaymentGateway interface {
	CreateSubscription(itemName string, price int, currency string) (*Subscription, error)
	CancelSubscription(subscriptionId string) error
	RetrieveSubscription(subscriptionId string) (*Subscription, error)
}

type Subscription struct {
	ID        string `json:"id"`
	Link      string `json:"link"`
	Completed bool   `json:"completed"`
	Canceled  bool   `json:"canceled"`
}

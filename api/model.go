package api

type CreateSpaceRequest struct {
	Name     string  `json:"name"`
	ParentId *string `json:"parentId"`
}

type CreateConsumerRequest struct {
	PhoneRegion string `json:"phoneRegion"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

type CreateSubscriptionPlanRequest struct {
	PaymentGateway string `json:"paymentGateway"`
	PlanName       string `json:"planName"`
	Currency       string `json:"currency"`
	Price          int    `json:"price"`
}

type CancelSubscriptionRequest struct {
	Secret string `json:"secret"`
}

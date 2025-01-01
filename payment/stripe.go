package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	stripePrice "github.com/stripe/stripe-go/v81/price"
	"github.com/stripe/stripe-go/v81/subscription"
)

type Stripe struct {
	Key string `json:"key"`
}

func (s *Stripe) RetrieveSubscription(subscriptionId string) (*Subscription, error) {
	sess, err := session.Get(subscriptionId, &stripe.CheckoutSessionParams{})
	if err != nil {
		return nil, err
	}
	sub := Subscription{
		ID:        sess.ID,
		Link:      sess.URL,
		Completed: sess.PaymentStatus == stripe.CheckoutSessionPaymentStatusPaid,
	}
	if !sub.Completed {
		return &sub, nil
	}
	subResult, err := subscription.Get(sess.Subscription.ID, &stripe.SubscriptionParams{})
	if err != nil {
		return &sub, nil
	}
	sub.Canceled = subResult.CanceledAt != 0
	return &sub, nil
}

func (s *Stripe) GetStripeSession(subscriptionId string) (*stripe.CheckoutSession, error) {
	sess, err := session.Get(subscriptionId, &stripe.CheckoutSessionParams{})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Stripe) GetStripeSubscription(subscriptionId string) (*stripe.Subscription, error) {
	sess, err := s.GetStripeSession(subscriptionId)
	if err != nil {
		return nil, err
	}
	if sess.Subscription == nil {
		return nil, fmt.Errorf("subscription not found")
	}
	subResult, err := subscription.Get(sess.Subscription.ID, &stripe.SubscriptionParams{})
	if err != nil {
		return nil, err
	}
	return subResult, nil
}

func (s *Stripe) CreateSubscription(itemName string, price int, currency string) (*Subscription, error) {
	priceParams := &stripe.PriceParams{
		Currency:   stripe.String(currency),
		UnitAmount: stripe.Int64(int64(price)),
		Recurring: &stripe.PriceRecurringParams{
			Interval: stripe.String(string(stripe.PriceRecurringIntervalMonth)),
		},
		ProductData: &stripe.PriceProductDataParams{Name: stripe.String(itemName)},
	}
	priceEntity, err := stripePrice.New(priceParams)
	if err != nil {
		return nil, err
	}
	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String("https://www.google.com"),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceEntity.ID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
	}
	result, err := session.New(params)
	if err != nil {
		return nil, err
	}
	subscription := Subscription{
		ID:   result.ID,
		Link: result.URL,
	}
	return &subscription, nil
}

func (s *Stripe) CancelSubscription(subscriptionId string) error {
	sub, err := s.GetStripeSubscription(subscriptionId)
	if err != nil {
		return err
	}
	_, err = subscription.Cancel(sub.ID, &stripe.SubscriptionCancelParams{})
	return err
}

func (s *Stripe) CancelPayment(subscriptionId string) error {
	sess, _ := s.GetStripeSession(subscriptionId)
	_, err := session.Expire(
		sess.ID,
		&stripe.CheckoutSessionExpireParams{},
	)
	if err != nil {
		return err
	}
	return nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/plutov/paypal/v4"
	ipn "github.com/webhookrelay/paypal-ipn"
)

func payHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "cookie-name")
	if err != nil {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
	}
	var val interface{}
	var ok bool
	if val, ok = session.Values["user"]; !ok {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
		return
	}

	user := val.(cookieUser)
	u := getUserByEmail(user.Email)
	if u.Paid {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
		return
	}

	sub_URL, err := newSubscriptionURL(user.Email)
	if err != nil {
		http.Redirect(w, r, "https://aipokedex.com/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, sub_URL, http.StatusSeeOther)
}

func newSubscriptionURL(email string) (url string, err error) {
	now := paypal.JSONTime(time.Now().Add(30 * time.Second))
	LivePLAN := "P-1YM44076XS7765410MCPINYI"
	//SandboxPLAN := "P-9VH32563PC536525BMCPHUJI"
	sub, err := paypalClient.CreateSubscription(context.Background(), paypal.SubscriptionBase{
		PlanID:    LivePLAN,
		StartTime: &now,
		Quantity:  "1",
		ApplicationContext: &paypal.ApplicationContext{
			BrandName: "AI Pokedex",
			ReturnURL: "https://aipokedex.com",
			CancelURL: "https://aipokedex.com",
		},
	})
	if err != nil {
		return "", errors.New("error creating subscriptions")
	}
	database.Update("accounts").
		Set("sub_id", sub.ID).
		Where("email=$1", email).
		Exec()
	return sub.Links[0].Href, nil
}

func paypalWebhook(err error, n *ipn.PaypalNotification) {
	if err != nil {
		return
	}
	if n.EventType == "BILLING.SUBSCRIPTION.ACTIVATED" {
		fmt.Println("activate: " + n.Resource.ID)
		database.Update("accounts").
			Set("paid", true).
			Where("sub_id=$1", n.Resource.ID).
			Exec()
	}
	if n.EventType == "BILLING.SUBSCRIPTION.CANCELLED" {
		fmt.Println("cancel: " + n.Resource.ID)
		database.Update("accounts").
			Set("paid", false).
			Where("sub_id=$1", n.Resource.ID).
			Exec()
	}
}

/*
	product, err := paypalClient.CreateProduct(context.Background(), paypal.Product{
		Name:        "AI Pokedex Premium Status",
		Description: "Activate premium status.",
		Type:        paypal.ProductTypeDigital,
		Category:    paypal.ProductCategorySoftwareOnlineServices,
		ImageUrl:    "https://i.imgur.com/o2mZ2ys.png",
		HomeUrl:     "https://aipokdex.com/",
	})
	if err != nil {
		fmt.Println("Error making product")
	}

	plan, err := paypalClient.CreateSubscriptionPlan(context.Background(), paypal.SubscriptionPlan{
		ProductId:   product.ID,
		Name:        "AI Pokedex Subscription",
		Description: "Activate premium status.",
		Status:      paypal.SubscriptionPlanStatusActive,
		BillingCycles: []paypal.BillingCycle{
			{
				Sequence:   1,
				TenureType: paypal.TenureTypeRegular,
				Frequency: paypal.Frequency{
					IntervalUnit:  paypal.IntervalUnitMonth,
					IntervalCount: 1,
				},
				TotalCycles: 0,
				PricingScheme: paypal.PricingScheme{
					FixedPrice: paypal.Money{
						Currency: "EUR",
						Value:    "2",
					},
				},
			},
		},
		PaymentPreferences: &paypal.PaymentPreferences{
			PaymentFailureThreshold: 1,
		},
	})
	if err != nil {
		fmt.Println("Error making plan")
	}
	fmt.Println(plan.ID)

	now := paypal.JSONTime(time.Now().Add(30 * time.Second))
	sub, err := paypalClient.CreateSubscription(context.Background(), paypal.SubscriptionBase{
		PlanID:    plan.ID,
		StartTime: &now,
		Quantity:  "1",
		ApplicationContext: &paypal.ApplicationContext{
			ReturnURL: "https://aipokedex.com",
			CancelURL: "https://aipokedex.com",
		},
	})
	if err != nil {
		fmt.Println("Error making sub")
	}

	fmt.Println(sub.Links[0].Href)
*/

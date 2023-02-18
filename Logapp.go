package main

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/goombaio/namegenerator"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
	"time"
)

type Item struct {
	Name     string
	Uuid     string
	Quantity int
}

var Stocks = []Item{}

type User struct {
	Name         string
	Uuid         string
	Step         int
	isBusy       bool
	Orders       *[]string
	CurrentOrder string
	Cart         []int
}

var users []User

const MaxUsers = 100
const SimultaneousLogs = 20
const NbProducts = 100
const MaxQuantities = 20

var inUse = 0

func main() {
	rand.Seed(time.Now().UnixNano())
	log.SetFormatter(&log.JSONFormatter{})
	nameGenerator := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano())

	productNames := productNameGenerator()
	for i := 0; i < NbProducts; i++ {
		f := uuid.New().String()
		Stocks = append(Stocks, Item{
			Name:     productNames[i],
			Quantity: rand.Intn(MaxQuantities),
			Uuid:     f,
		})
	}
	users = make([]User, MaxUsers)
	for i := 0; i < MaxUsers; i++ {
		uuid := uuid.New()
		users[i] = User{
			Name:   nameGenerator.Generate(),
			Uuid:   uuid.String(),
			Step:   0,
			isBusy: false,
			Orders: nil,
			Cart:   []int{},
		}
	}

	mux := sync.Mutex{}
	for {
		for inUse < SimultaneousLogs {
			mux.Lock()
			inUse++
			mux.Unlock()
			go func() {
				var u *User
				for {
					e := rand.Intn(MaxUsers)
					u = &users[e]
					if !u.isBusy {
						u.isBusy = true
						break
					}
				}
				u.MakeALog()
				mux.Lock()
				inUse--
				u.isBusy = false
				mux.Unlock()
			}()
		}
	}

}

func (u *User) MakeALog() {
	u.Step = u.DecisionTree()
}

var sleepDuration = time.Second

func (u *User) DecisionTree() int {
	l2 := log.WithField("uuid", u.Uuid)

	switch u.Step {
	case -2:
		l2.WithField("evt", "login.reset").Info("Password reset successful")
		return 0
	case -1:
		l2.WithField("evt", "login.forget_password").Info("user has requested new password")
		time.Sleep(10 * sleepDuration)
		return -2
	case 0:
		f := rand.Intn(4) % 4
		d := device()
		if f == 0 {
			l2.
				WithField("evt", "login.failure").
				WithField("device", d).
				Warn(u.Name + " failed to log")
			time.Sleep(time.Duration(5) * sleepDuration)

			return []int{0, -1}[rand.Intn(3)%2]
		}
		l2.
			WithField("evt", "login.success").
			WithField("device", d).
			Info(u.Name + " successfully logged")
		return []int{1, 10}[rand.Intn(2)]
	case 1:
		l2.WithField("evt", "shop.browse").Info("visiting shop")
		time.Sleep(time.Duration(rand.Intn(5)) * sleepDuration)
		return []int{1, 1, 1, 1, 2, 2, 2, 2, 3, 10}[rand.Intn(10)]
	case 2:
		e := rand.Intn(len(Stocks))
		product := &Stocks[e]
		l2.
			WithField("evt", "product.add").
			WithField("product", product.Uuid).
			Info("adding product in cart")
		if product.Quantity < 1 {
			l2.
				WithField("evt", "product.add").
				WithField("product", product.Uuid).
				WithField("stocks", 0).
				Error("Product Out of Stock")
			return 1
		}
		product.Quantity--
		if product.Quantity < 10 {
			l2.
				WithField("evt", "product.add").
				WithField("product", product.Uuid).
				WithField("stocks", product.Quantity).
				Warn("low product stocks")
		}
		u.Cart = append(u.Cart, e)
		return []int{1, 1, 1, 2, 2, 3, 4, 4, 4}[rand.Intn(9)]
	case 3: //remove product from cart
		if len(u.Cart) == 0 {
			return 1
		}
		r := rand.Intn(len(u.Cart))

		e := u.Cart[r]
		product := &Stocks[e]
		product.Quantity++
		l2.
			WithField("evt", "product.remove").
			WithField("product", product.Uuid).
			WithField("stocks", product.Quantity).
			Info("Removing from cart")
		return 1
	case 4: //checkout
		if len(u.Cart) == 0 {
			l2.
				WithField("evt", "cart.empty").
				Warn("Cannot checkout on empty cart")
			return []int{1, 10}[rand.Intn(2)]
		}
		t := []string{}
		if u.Orders != nil {
			t = *u.Orders
		}
		t = append(t, uuid.New().String()[0:8])
		u.Orders = &t
		l2.
			WithField("evt", "cart.checkout").
			Info("Checking out")
		return []int{5, 5, 5}[rand.Intn(3)]
	case 5:
		if len(*u.Orders) < 1 {
			l2.WithField("evt", "pay.error").Error("Cannot Pay unexisting order")
			return 1
		}
		l2.
			WithField("evt", "pay.secure").
			WithField("order", (*u.Orders)[len(*u.Orders)-1]).
			Info("Starting Secure Payment")
		time.Sleep(5 * sleepDuration)
		return []int{6, 6, 6, 6, 6, 6, 6, 6, 6, 7}[rand.Intn(10)]
	case 6: //pay success
		l2.
			WithField("evt", "pay.success").
			Info("Payment Success")
		u.Cart = []int{}
		u.CurrentOrder = ""
		return []int{1, 10}[rand.Intn(2)]

	case 7: //pay failure
		t := (*u.Orders)[:len(*u.Orders)-1]
		u.Orders = &t
		l2.
			WithField("evt", "pay.failure").
			Error("Payment Failure")
		return []int{4, 4, 4, 4, 8}[rand.Intn(5)]
	case 8:
		l2.
			WithField("evt", "order.canceled").
			Error("Cancelling Payment")
		return []int{1, 10}[rand.Intn(2)]

	case 10:
		l2.WithField("evt", "orders.browse").Info("browsing old order history")

		time.Sleep(time.Duration(rand.Intn(5)) * sleepDuration)
		return []int{11, 12}[rand.Intn(2)]

	case 11:
		o := u.gettingOneOrder()
		if o == nil {
			return 1
		}
		l2.
			WithField("evt", "orders.view").
			WithField("order", *o).
			Error("checking old order")
	case 12:
		l2.
			WithField("evt", "user.logout").
			Info("user Logout")
		return 0
	}

	return 0
}

func device() string {
	switch rand.Intn(4) {
	case 0:
	case 1:
	case 2:
	case 3:
		return fmt.Sprintf("safari-%v.%v", 14+rand.Intn(3), rand.Intn(5))
	}
	return fmt.Sprintf("chrome-%v", (rand.Intn(20) + 89))

}

func (u *User) gettingOneOrder() *string {
	if u.Orders == nil {
		history := rand.Intn(10)
		t := []string{}
		for i := 0; i < history; i++ {
			t = append(t, uuid.New().String()[0:8])
		}
		u.Orders = &t
	}

	if len(*u.Orders) == 0 {
		return nil
	}

	e := rand.Intn(len(*u.Orders))
	u.CurrentOrder = (*u.Orders)[e]
	return &u.CurrentOrder
}

func productNameGenerator() []string {
	prefixes := []string{"pantalon", "tee-shirt", "manteau", "chemise", "écharpe", "chaussures", "chausettes", "bottes", "bonnet", "gants", "lunettes"}
	gender := []string{"homme", "femme", "fille", "garçon"}
	colors := []string{"bleu", "blanc", "rouge", "vert", "noir", "gris", "jaune", "violet", "rose"}
	e := []string{}
	for _, p := range prefixes {
		for _, g := range gender {
			for _, c := range colors {
				e = append(e, fmt.Sprintf("%v %v %v", p, c, g))
			}
		}
	}
	return e
}

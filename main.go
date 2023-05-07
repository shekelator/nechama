package main

import (
	"fmt"
	"time"

	// "github.com/hebcal/hebcal-go"
	"github.com/hebcal/hdate"
	// "github.com/hebcal/hebcal-go/event"
)

func main() {

	d := hdate.FromGregorian(2012, time.April, 21)
	barMitzvah, err := hdate.GetBirthdayOrAnniversary(5785, d)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s - %s\n", barMitzvah, barMitzvah.Gregorian())
}

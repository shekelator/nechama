package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hebcal/hdate"
)

type nestedArray [][]string

type Response struct {
	Hebrew  nestedArray `json:"he"`
	English nestedArray `json:"text"`
}

type Text struct {
	Hebrew    string
	English   string
	Reference string
}

func main() {
	bmDate := getAriBarMitzvahDate()

	fmt.Println(bmDate)

	text, err := getSefariaData("Exodus 6:29-7:7")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(text)
}

func getAriBarMitzvahDate() string {
	d := hdate.FromGregorian(2012, time.April, 21)
	barMitzvah, err := hdate.GetBirthdayOrAnniversary(5785, d)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return fmt.Sprintf("%s - %s\n", barMitzvah, barMitzvah.Gregorian())
}

func getSefariaData(ref string) (string, error) {
	resp, err := http.Get("https://www.sefaria.org/api/texts/Exodus 6:29-7:7/en/Tanakh: The Holy Scriptures, published by JPS")
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	var responseData Response
	response := json.Unmarshal(body, &responseData)
	if response != nil {
		return "", response
	}
	// fmt.Println(responseData)
	result := flatten(responseData.English)

	return result, nil
}

func flatten(incoming nestedArray) string {
	var sb strings.Builder

	for _, arr := range incoming {
		for _, str := range arr {
			sb.WriteString(str)
		}
	}
	return sb.String()
}

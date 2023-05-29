package sefariawrap

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type nestedArray [][]string

type response struct {
	Hebrew  nestedArray `json:"he"`
	English nestedArray `json:"text"`
}

type Text struct {
	Hebrew    string
	English   string
	Reference string
}

func GetSefariaData(ref string) (*Text, error) {
	resp, err := http.Get("https://www.sefaria.org/api/texts/Exodus 6:29-7:7/en/Tanakh: The Holy Scriptures, published by JPS")
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var responseData response
	response := json.Unmarshal(body, &responseData)
	if response != nil {
		return nil, response
	}
	// fmt.Println(responseData)
	// result := flatten(responseData.English)
	result := Text{
		Hebrew:    flatten(responseData.Hebrew),
		English:   flatten(responseData.English),
		Reference: ref,
	}

	return &result, nil
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

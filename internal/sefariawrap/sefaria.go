package sefariawrap

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode"
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
	body, err := io.ReadAll(resp.Body)
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
		Hebrew:    processHebrewResponse(responseData.Hebrew),
		English:   flatten(responseData.English),
		Reference: ref,
	}

	return &result, nil
}

func isCharToKeep(r rune) bool {
	return unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Space, r)
}

func processHebrewResponse(nestedArray nestedArray) string {
	flattened := flattenRTL(nestedArray)
	var sb strings.Builder
	for _, r := range flattened {
		if isCharToKeep(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func flattenRTL(incoming nestedArray) string {
	var sb strings.Builder

	replacer := strings.NewReplacer("<span class=\"mam-spi-samekh\">{ס}</span>", "{ס}\t",
		"<span class=\"mam-spi-pe\">{פ}</span>", "{פ}", "<br>", "\n", "<b>", "", "</b>", "")

	for _, arr := range incoming {
		for i := len(arr) - 1; i >= 0; i-- {
			str := arr[i]

			sb.WriteString(replacer.Replace(str))
		}
	}

	return sb.String()
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

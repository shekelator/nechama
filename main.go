package main

import (
	"fmt"
	"time"

	"github.com/shekelator/nechama/internal/sefariawrap"

	"github.com/hebcal/hdate"
	"github.com/spf13/cobra"
)

func main() {
	// root.Execute()

	bmDate := getAriBarMitzvahDate()

	fmt.Println(bmDate)

	text, err := sefariawrap.GetSefariaData("Deuteronomy 3")
	if err != nil {
		fmt.Println(err)
		return
	}

	hebText := text.Hebrew
	// for i, c := range hebText {
	// 	if i > 2500 {
	// 		break
	// 	}
	// 	s := string(c)
	// 	r, size := utf8.DecodeRuneInString(s)
	// 	if slices.Contains([]int{0x202A, 0x202B, 0x202C, 0x202D, 0x05D3}, int(r)) {
	// 		fmt.Printf("%d (%X) -> %v\n", r, r, size)
	// 	}

	// }

	fmt.Println(hebText)
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

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `All software has versions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("0.1")
	},
}

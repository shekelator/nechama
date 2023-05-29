package sefariawrap

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	// "github.com/shekelator/nechama/internal/sefariawrap"
)

func TestFlatten(t *testing.T) {
	nested := [][]string{
		{"One", "two"},
		{},
		{"Three", "four"},
	}
	assert(t, flatten(nested) == "OnetwoThreefour", "Flattens string slices correctly")
}
func TestFormatHebText(t *testing.T) {
	// testText := "וַיֹּ֤אמֶר יְהֹוָה֙ אֶל־מֹשֶׁ֔ה עַתָּ֣ה תִרְאֶ֔ה אֲשֶׁ֥ר אֶֽעֱשֶׂ֖ה לְפַרְעֹ֑ה כִּ֣י בְיָ֤ד חֲזָקָה֙ יְשַׁלְּחֵ֔ם וּבְיָ֣ד חֲזָקָ֔ה יְגָרְשֵׁ֖ם מֵאַרְצֽוֹ׃ <span class=\"mam-spi-samekh\">{ס}</span>        וַיְדַבֵּ֥ר אֱלֹהִ֖ים אֶל־מֹשֶׁ֑ה וַיֹּ֥אמֶר אֵלָ֖יו אֲנִ֥י יְהֹנִ֥ה׃יְהֹנִ֥א יְל־אַוּיְהֹם וּל־נִ֥אַוּק נִ֥אַל־הֹנִ֥אַב הֹנִ֥ל יְהֹי וּיְהֹי וּיְהֹה וּא נאַוּיְהֹי וּיְם׃אַוּם נִ֥אַוּיְי אַת־הֹנִ֥יוּיְ נִ֥אַם הֹנִ֥ת יְהֹם וּת־נִ֥אַץ הֹנִ֥אַן הֹת וּיְץ אַוּיְינִ֥ם יְהֹר־וּרהֹ אַוּ׃הֹנִ֥ם <b>׀</b> אֲנִ֣י שָׁמַ֗עְתִּי אֶֽת־נַאֲקַת֙ בְּגַ֣י וּנַאֲקַל בְּגַ֣ר וּנַאֲקַם בְּגַ֣אֶֽאֶֽים קַת֙ם גַ֣אֶֽאֶֽוּר קַת־בְּגַ֣יאֶֽי׃אֲקַן בְּגַ֣ר וּנַאֲי־וְבְּגַ֣אֶֽאֶֽ נַאֲי וְבְּגַ֣אֶֽ וּהאֲקַאוְי אֶֽאֶֽוּם קַת֙וְבְּ אֶֽאֶֽוּת קַת֙וְבְּם אֶֽוּנַאֲקַי בְּגַ֣אֶֽם נַאֲקַת֙וְם אֶֽאֶֽוּנַאֲי וְבְּגַ֣אֶֽ וּנַאֲוֹעַ נְטוּיָ֔ה וּבִשְׁפָטִ֖ים גְּדֹלִֽים׃וְלָקַחְתִּ֨י סִבְגְּם מִצְ וְלָם תִּ֨צַּלְיבְי לִֽמִם וְאקַחְים סִיגְּדֹלִֽם רַ֔י קַחְי לְסִבְגְּ לִֽמִצְיוְם חְתִּ֨צַּלְיא דֹלִֽמִם וְלָקַת צַּלְסִבְת לִֽמִצְרַ֔ם׃קַחְתִּ֨אלְי גְּדֹלִֽמִ רַ֔ל־קַחְתִּ֨ץ סִבְר לִֽמִארַ֔וְ קַת־צַּלְי גְּדֹת אֹתָ֔הּ לְאַבְרָהָ֥ם לְיִצְחָ֖ק וּֽלְיַעֲקֹ֑ב וְנָתַתִּ֨י אֹתָ֥הּ לָכֶ֛ם מוֹעֲקֹ֑ה וְנָי תְכֶ֔אֹה׃תַּ֖לָכֶ֛עָ֔ר וֹעֲה צִ֣ן תַל־כֶ֔אֹי תַּ֖לָכֶ֛עָ֔ל וֹעֲא צִ֣וְעתַ תְל־תָ֥הּה כֶ֛עָ֔סִר עֲקֹ֑וֹ וְנָתַתִּ֨תְה תָ֥הּה׃ <span class=\"mam-spi-pe\">{פ}</span><br>כֶ֛יְדַבֵּ֥ר יְהֹוָ֖ה אֶל־מֹשֶׁ֥ה לֵּאמֹֽר׃בֹּ֣א דַבֵּ֔ר אֶל־פַּרְעֹ֖ה מֶ֣לֶךְ מִצְרָ֑יִם וִֽישַׁדַח דַת־יִפַּי־פַּרְמֶ֣לֶל מִצְרָ֑יִתֵ֣׃וִֽמְשַׁדַר דַאֶה פַּרְעֹ֖י מֶ֣לֶךְה צְאיִר וִֽן דַבֵּ֔י־אֶאֶיִפַּרְ פַּא־לֶךְאֵ֑מִ רָ֑יִי וִֽמְידַ כֵּ֖דַאֶאֶיִי עֹ֖פַּרְה ךְאֵ֑מִי יִתֵ֣ל מְשַׁדַבֵּ֔ם׃ <span class=\"mam-spi-pe\">{פ}</span><br>וַיְדַבֵּ֣ר יְהֹוָה֮ אֶל־מֹשֶׁ֣ה וְאֶֽל־פַּמֶ֣לֶךְ מִצְרָ֑וַיְ בֵּ֣ל־יְהֹי יְאֶבְּנֵֽל יִשְׂל־עֵ֣נִפַּה ךְרָמִ רָ֑וַיְדַם שַׁההֹוָיא בְּת־שֶׁ֣יִי־אֶֽעֵ֣נִפַּל ךְרָמִץ וַיְדַבֵּ֣ם׃ <span class=\"mam-spi-samekh\">{ס}</span>        "

	text, _ := GetSefariaData("Deuteronomy 3")
	assert(t, len(text.Hebrew) > 10, "Text has length")
	value := 100
	equals(t, 100, value)
}

// Test functions from https://github.com/benbjohnson/testing

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

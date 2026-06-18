package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shekelator/nechama/internal/sefaria"
)

func chooseTranslation(input io.Reader, output io.Writer, versions []sefaria.VersionChoice) (sefaria.VersionChoice, error) {
	if len(versions) == 0 {
		return sefaria.VersionChoice{}, sefaria.ErrNoEnglishTranslations
	}

	if len(versions) == 1 {
		return versions[0], nil
	}

	fmt.Fprintln(output, "Available English translations:")
	for i, version := range versions {
		fmt.Fprintf(output, "%d. %s\n", i+1, version.DisplayTitle())
	}

	reader := bufio.NewReader(input)
	for {
		fmt.Fprintf(output, "Choose a translation [1-%d]: ", len(versions))

		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return sefaria.VersionChoice{}, err
		}

		selection, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && selection >= 1 && selection <= len(versions) {
			return versions[selection-1], nil
		}

		fmt.Fprintf(output, "Invalid selection. Enter a number from 1 to %d.\n", len(versions))
	}
}

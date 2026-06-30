package app

import (
	"fmt"
	"time"

	"github.com/phillarmonic/figlet/figletlib"
)

// Domain: Version Display
// This file contains logic for displaying version information

// ShowVersion displays version information with ASCII art
func ShowVersion(version, commit, date string) error {
	loader := figletlib.NewEmbededLoader()
	font, err := loader.GetFontByName("standard")
	if err != nil {
		return err
	}

	startColor, _ := figletlib.ParseColor("#00FF95")
	endColor, _ := figletlib.ParseColor("#00C2FF")
	gradientConfig := figletlib.ColorConfig{
		Mode:       figletlib.ColorModeGradient,
		StartColor: startColor,
		EndColor:   endColor,
	}

	fmt.Println("")
	figletlib.PrintColoredMsg("dRun CLI", font, 80, font.Settings(), "left", gradientConfig)

	fmt.Println("drun (do-run) automation language")
	fmt.Println("xDrun (eXecute drun) CLI")
	fmt.Println()
	fmt.Println("Effortless tasks, serious speed.")
	fmt.Println("By Phillarmonic Software <https://github.com/phillarmonic/drun>")
	fmt.Println("")
	fmt.Printf("Version %s\n", version)
	if commit != "unknown" {
		fmt.Printf("commit: %s\n", commit)
	}
	if date != "unknown" {
		fmt.Printf("Built in %s\n", formatBuildDate(date))
	}
	return nil
}

func formatBuildDate(date string) string {
	parsed, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return date
	}

	return parsed.UTC().Format("02/01/2006 15:04 UTC")
}

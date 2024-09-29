package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/eriklarko/license-checker/src/checker"
	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlists"
	"github.com/eriklarko/license-checker/src/environment"
	"github.com/eriklarko/license-checker/src/tui"
)

const defaultConfigFilePath = ".license-checker.yaml"

var configFilePath = flag.String("config-file", defaultConfigFilePath, "Path to the config file")
var licensesScript = flag.String("licenses-script", "", "Path to the script for getting current licenses")
var licensesFile = flag.String("licenses-file", "", "Path to the file containing approved and disapproved licenses")
var interactive = flag.Bool("interactive", false, "Force the script into interactive mode")

func main() {
	// Parse command line flags
	flag.Parse()

	// Set interactive mode if the flag is provided
	if *interactive {
		environment.ForceSetIsInteractive(*interactive)
	}

	// Load the config
	config, err := setUpConfig()
	if err != nil {
		panic(err)
	}

	tui := tui.New()

	licenseChecker, err := setUpLicenseChecker(config, func(license, dependency string) bool {
		return handleUnknownLicense(dependency, license, tui)
	})
	if err != nil {
		panic(err)
	}

	if environment.IsInteractive() {
		// slog to file instead of console
		// open file, creating one if it doesnt exist
		// TODO print where the log file will end up
		f, err := os.OpenFile("license-checker.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("No log file!!\n")
			panic(err)
		}
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{})))
	}

	if _, err := os.Stat(config.LicensesFile); os.IsNotExist(err) {
		if environment.IsInteractive() {
			curatedlistsService := curatedlists.New(config, http.DefaultTransport)
			askToChooseCuratedList(curatedlistsService, tui)
		} else {
			printInteractiveInstructions("No license file found. Please run this tool interactively to set everything up.")
			os.Exit(1)
		}
	}

	// if config.LicensesScript doesn't exist, guide the user to set it up
	// we need a set of scripts for well-known package managers
	currentLicenses, err := getCurrentLicenses(config.LicensesScript)
	if err != nil {
		panic(err)
	}

	licenseChecker.ValidateCurrentLicenses(currentLicenses)
}

func setUpConfig() (*config.Config, error) {
	conf, err := config.LoadConfig(*configFilePath)
	if os.IsNotExist(err) {
		conf = config.DefaultConfig()
	} else if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	overwriteConfigWithCmdLineFlags(conf)

	err = conf.Validate()
	if err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	return conf, nil
}

func overwriteConfigWithCmdLineFlags(conf *config.Config) {
	if conf.LicensesScript == "" {
		conf.LicensesScript = *licensesScript
	}
	if conf.LicensesFile == "" {
		conf.LicensesFile = *licensesFile
	}
}

func setUpLicenseChecker(conf *config.Config, onUnknownLicense func(license, dependency string) bool) (*checker.LicenseChecker, error) {
	licenseMap, err := conf.ReadLicenseMap()
	if os.IsNotExist(err) {
		// it's okay for the license map file to not exist. this could be the first run
		// if it isn't though, then it's an error. Maybe we can detect that somehow...
		licenseMap = make(map[string]bool)
	} else if err != nil {
		return nil, fmt.Errorf("failed to read license map: %w", err)
	} else {
		// TODO: tui?????
		fmt.Printf("Read existing decisions from %s\n", conf.LicensesFile)
	}

	return checker.NewFromMap(licenseMap, onUnknownLicense), nil
}

// TODO: Move and test
func getCurrentLicenses(script string) (map[string]string, error) {
	cmd := exec.Command(script)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout for script %s: %w", script, err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command %s: %w", script, err)
	}

	licenses := make(map[string]string)
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Split(line, ",")
		licenses[parts[0]] = parts[1]
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for command %s to finish: %w", script, err)
	}

	return licenses, nil
}

// TODO: Test
func handleUnknownLicense(
	license, dependency string,
	tui *tui.TUI,
) bool {
	if environment.IsInteractive() {
		return tui.AskYesNo(
			"Unknown license %s from %s detected. Should it be allowed?",
			license,
			dependency,
		)
	} else {
		printInteractiveInstructions(
			"Unknown license detected. To decide if the license is allowed or not, please run this tool again interactively.",
			"license", license,
		)
		return false
	}

}

func printInteractiveInstructions(message string, args ...any) {
	// TODO: verify hint
	args = append(args, "hint", "For example, run `./license-checker .` from the project root.")

	slog.Warn(
		message,
		args...,
	)
}

func askToChooseCuratedList(s *curatedlists.Service, tui *tui.TUI) {
	tui.Println("It seems no choices around which licenses are allowed or not have been made yet.")
	tui.Println("We can download some predefined lists of licenses to get you started.")
	tui.Println("They aren't perfect and you're likely to have to make some adjustments, but we'll go through all that together")
	tui.Println()

	choice := tui.AskMultipleChoice(
		"Do you want to download a curated list of licenses or go your own way?",
		"Let's look at some curated lists",
		"I'll go my own way",
	)
	if choice != 0 {
		tui.Println("Fair enough")
		return
	}

	tui.Println("Perfect! Let me just download the list data before we continue...")
	// TODO: progress indicator
	//progressIndicator := tui.NewProgressIndicator()
	err := s.DownloadCuratedLists( /*func(current, total int) {
		progressIndicator.SetProgress(current, total)
	}*/)
	if err != nil {
		panic(err)
	}

	suggestedList, description, err := s.GetHighlyRatedList()
	if err != nil {
		slog.Error("failed to get default list", "error", err)
	}

	if suggestedList != "" {
		tui.Printf("Okay, it looks like a common choice for this type of project is %s\n", suggestedList)
		tui.Println(description)

		if tui.AskYesNo("Do you want to use this list?") {
			tui.Println("Great! It's all been set up for you.")
			err = s.SelectList(suggestedList)
			if err != nil {
				panic(err)
			}
			return
		} else {
			tui.Println("Okay, let's look at some other options")
		}
	}

	// looking at all lists
	lists, err := s.GetAllLists()
	if err != nil {
		panic(err)
	}

	tui.Printf("We have %d lists to choose from, and you're of course free to not use any of them`\n", len(lists))
	answers := make([]string, 0, len(lists)+1)
	for listName, listInfo := range lists {
		answers = append(answers, listName)

		tui.Println()
		tui.Printf("List: %s\n", listName)
		tui.Println(listInfo.Description)
	}

	answers = append(answers, "None of them please")
	tui.Println()
	answer := tui.AskMultipleChoice("Which list do you want to use?", answers...)
	if answer == len(answers)-1 {
		tui.Println("Totally fair! You can always download a list later.")
		tui.Println("For now we'll go through the process of setting up your own list.")
		return
	}

	tui.Println("Great! It's all been set up for you.")
	err = s.SelectList(answers[answer])
	if err != nil {
		panic(err)
	}
}

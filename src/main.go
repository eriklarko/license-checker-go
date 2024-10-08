package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/eriklarko/license-checker/src/checker"
	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlicensescripts"
	"github.com/eriklarko/license-checker/src/curatedlicensescripts/packagemanagerdetector"
	"github.com/eriklarko/license-checker/src/curatedlists"
	"github.com/eriklarko/license-checker/src/environment"
	"github.com/eriklarko/license-checker/src/licensedescriber"
	"github.com/eriklarko/license-checker/src/phraser"
	"github.com/eriklarko/license-checker/src/tui"
	"github.com/samber/lo"
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

	licenseChecker, err := setUpLicenseChecker(config)
	if err != nil {
		panic(err)
	}

	if environment.IsInteractive() {
		logFilePath := "license-checker.log"
		tui.Printf("Logs are written to %s\n", logFilePath)
		tui.Println()

		// open file, creating one if it doesnt exist
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("No log file!!\n")
			panic(err)
		}
		// slog to file instead of console
		slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{})))
	}

	// TODO: Verify curated script and list md5s

	// detect if the tool needs to be set up
	if _, err := os.Stat(config.LicensesFile); os.IsNotExist(err) {
		curatedlistsService := curatedlists.New(config)
		if config.SelectedCuratedList != "" {
			slog.Warn("License file for selected curated list not found. Downloading it", "list", config.SelectedCuratedList)
			err := curatedlistsService.DownloadList(config.SelectedCuratedList)
			if err != nil {
				panic(err)
			}
		} else if environment.IsInteractive() {
			askToChooseCuratedList(curatedlistsService, tui)
			tui.Println()
		} else {
			printInteractiveInstructions("No license file found. Please run this tool interactively to set everything up.")
			os.Exit(1)
		}
	}

	// detect if the script for getting current licenses is missing
	if _, err := os.Stat(config.LicensesScript); os.IsNotExist(err) {
		if environment.IsInteractive() {
			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			pmd := packagemanagerdetector.New(wd)
			cls := curatedlicensescripts.New(config)

			askToChooseLicensesScript(pmd, cls, config, tui)
			tui.Println()
		} else {
			printInteractiveInstructions(
				"Couldn't find script used to get current licenses. Please run this tool interactively to set everything up.",
				"script", config.LicensesScript,
			)
			os.Exit(1)
		}
	}
	currentLicenses, err := getCurrentLicenses(config.LicensesScript)
	if err != nil {
		panic(err)
	}

	if environment.IsInteractive() {
		runInteractive(tui, licenseChecker, currentLicenses, config)
	} else {
		runNonInteractive(licenseChecker, currentLicenses)
	}
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

func setUpLicenseChecker(conf *config.Config) (*checker.LicenseChecker, error) {
	lc, err := checker.NewFromFile(conf.LicensesFile)
	if os.IsNotExist(err) {
		// return checker with no decisions made
		return checker.NewFromMap(make(map[string]bool)), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to load license checker from file %s: %w", conf.LicensesFile, err)
	}
	return lc, nil
}

// TODO: Move and test
// For JS, look at https://github.com/franciscop/legally
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

func runNonInteractive(licenseChecker *checker.LicenseChecker, currentLicenses map[string]string) {
	slog.Warn(getDisclaimer())

	report, err := licenseChecker.ValidateCurrentLicenses(currentLicenses)
	if err != nil {
		panic(err)
	}

	if report.HasDisallowedLicenses() {
		slog.Error("Disallowed licenses detected", "licenses", report.Disallowed)
		os.Exit(1)
	}

	if report.HasUnknownLicenses() {
		printInteractiveInstructions(
			"Unknown licenses detected. To decide if they are allowed or not, please run this tool again interactively.",
			"licenses", report.Unknown,
		)
		os.Exit(1)
	}

	slog.Info("All licenses are allowed")
	os.Exit(0)
}

func printInteractiveInstructions(message string, args ...any) {
	// TODO: verify hint
	args = append(args, "hint", "For example, run `./license-checker .` from the project root.")

	slog.Warn(
		message,
		args...,
	)
}

func getDisclaimer() string {
	return "DISCLAIMER: THIS IS NOT LEGAL ADVICE. YOU ARE RESPONSIBLE FOR ENSURING THAT YOUR PROJECT COMPLIES WITH ALL APPLICABLE LAWS AND LICENSES."
}

func runInteractive(
	tui *tui.TUI,
	licenseChecker *checker.LicenseChecker,
	currentLicenses map[string]string,
	conf *config.Config,
) {
	phraser := phraser.New([]string{
		"To start let's look at license %s",
		"Next up license %s",
		"Let's look at %s",
		"Let's think about %s",
		"Next up is %s",
		"Let's consider %s",
	})

	licenseDescriber := licensedescriber.NewTLDRDescriber()

	// validate licenses until there are no unknown licenses
	for {
		report, err := licenseChecker.ValidateCurrentLicenses(currentLicenses)
		if err != nil {
			panic(err)
		}

		if report.HasDisallowedLicenses() {
			for license, dependencies := range report.Disallowed {
				tui.Printf("Disallowed license %s detected\n", license)
				if len(dependencies) == 1 {
					tui.Printf("It's currently only used by dependency %s\n", dependencies[0])
				} else {
					tui.PrintList("It's used by the following dependencies:", lo.ToAnySlice(dependencies), "#")
				}

				tui.Println("Please remove the disallowed dependencies or allow the license")
				if tui.AskYesNo("Do you want to allow this license?") {
					tui.Println("Okay, we'll remember that you want to allow this license")
					licenseChecker.Update(license, true)
					err := licenseChecker.Write(conf.LicensesFile)
					if err != nil {
						panic(err)
					}
				}
				tui.Println()
			}
			break
		} else {
			tui.Println("Excellent news! No disallowed licenses detected")
		}

		if report.HasUnknownLicenses() {
			tui.Printf("Okay, so we found %d unknown license(s). Let's go through them one by one\n", len(report.Unknown))
			tui.Println()

			for license, dependencies := range report.Unknown {
				tui.Println(phraser.Get(license))

				description, err := licenseDescriber.Describe(license)
				if err != nil {
					slog.Warn("Failed to describe license", "license", license, "error", err)
				} else if description != "" {
					tui.Printf("\n%s\n", description)
				}

				if len(dependencies) == 1 {
					tui.Printf("It's currently only used by dependency %s\n", dependencies[0])
				} else {
					tui.PrintList("It's used by the following dependencies:", lo.ToAnySlice(dependencies), "#")
				}

				isAllowed := tui.AskYesNo("Do you want to allow this license?")
				licenseChecker.Update(license, isAllowed)
				if !isAllowed {
					tui.Println("Okay, we'll remember that you don't want to allow this license")
					tui.Println("Please remove any dependencies using it")
				}
				tui.Println()
			}

			// update licenses file with new decisions
			err = licenseChecker.Write(conf.LicensesFile)
			if err != nil {
				panic(err)
			}
		} else {
			break
		}
	}
}

func askToChooseCuratedList(s *curatedlists.Service, tui *tui.TUI) {
	tui.Println("It seems no choices around which licenses are allowed or not have been made yet.")
	tui.Println("We can download some predefined lists of licenses to get you started.")
	tui.Println("They aren't perfect and you're likely to have to make some adjustments, but we'll go through all that together")

	// TODO: test that the disclaimer is printed
	tui.Println()
	tui.Println(getDisclaimer())
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
		if listInfo.Description != "" {
			tui.Println(listInfo.Description)
		}
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

func askToChooseLicensesScript(
	pmd *packagemanagerdetector.Service,
	cls *curatedlicensescripts.Service,
	conf *config.Config,
	tui *tui.TUI,
) {
	tui.Println("There's no script set up to get the current licenses of your project.")
	tui.Println("We'll go through the process of setting that up together.")
	tui.Println()
	//tui.Println("The script should output a list of projects and their licenses, separated by commas.")

	detectedPackageManagers, err := pmd.FindLikelyPackageManagers()
	if err != nil {
		panic(err)
	}

	if len(detectedPackageManagers) == 1 {
		scriptExists, err := cls.HasScriptForPackageManager(detectedPackageManagers[0])
		if err != nil {
			slog.Warn("Failed to check if script exists for package manager", "package_manager", detectedPackageManagers[0], "error", err)
		} else if scriptExists {
			tui.Printf("It seems your project uses %s\n", detectedPackageManagers[0])
			if tui.AskYesNo("Do you want to use a preset script reading licenses for %s?", detectedPackageManagers[0]) {
				tui.Println("Great! Setting that up for you...")
				cls.SelectScript(detectedPackageManagers[0])
				return
			} else {
				tui.Println("Fair enough, let's set up a script for you to use")
				tui.Println()
			}
		}

	} else if len(detectedPackageManagers) > 1 {
		tui.Println("More than one package manager was detected in your project.")
		tui.Println("You will likely want to set up your own script reading dependencies from all of them.")
		tui.Println("However, a preset script for one of them can be provided to get you started.")
		tui.Println()

		choices := append(detectedPackageManagers, "No - I'll provide my own script")
		choice := tui.AskMultipleChoice("Do you want to use a preset script for any of these?", choices...)
		if choice == len(choices)-1 {
			tui.Println("Good call")
			tui.Println()
		} else {
			tui.Printf("Great! Setting that up for you...")
			cls.SelectScript(detectedPackageManagers[choice])
			return
		}

	} else if len(detectedPackageManagers) == 0 {
		tui.Println("I couldn't detect any package managers in your project and can't help you with a reasonable default script :(")
	}

	path := tui.FilePicker("Please provide the path to a script that outputs a list of projects and their licenses, separated by commas")
	tui.Println("Great! Setting that up for you...")

	conf.LicensesScript = path
	if err = conf.Write(); err != nil {
		panic(err)
	}
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"github.com/urfave/cli/v2"
)

var version = ""

func main() {
	app := cli.NewApp()
	app.Flags = append(app.Flags, &cli.StringSliceFlag{
		Name:  "e",
		Usage: "Inject .env file(s) into the environment",
	}, &cli.BoolFlag{
		Name:  "i",
		Usage: "Ignore inherited environment",
	})
	app.Name = "dotenv"
	app.Version = version
	app.HideHelpCommand = true
	app.Usage = "Inject envfiles into the env and start the command"
	// app.Description = "Inject envfiles into env before starting a app"
	app.UsageText = fmt.Sprintf("%s [-e ENVFILE] [-i] [COMMAND [ARGS ...]]", os.Args[0])
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error: %w", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	// Craft new subprocess with args (not yet running)
	e := exec.Command(c.Args().Get(0), c.Args().Tail()...)
	// (-i) Inject current env into subprocess env if ignore flag is false (default)
	if !c.Bool("i") {
		e.Env = append(e.Env, os.Environ()...)
	}
	// (-e) Inject envfiles into subprocess env
	for _, path := range c.StringSlice("e") {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %w", path, err)
		}
		scanner := bufio.NewScanner(f)
		for lineNumber := 0; scanner.Scan(); lineNumber++ {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to scan line %d of file %q: %w", lineNumber, path, err)
			}
			line := strings.TrimLeftFunc(scanner.Text(), unicode.IsSpace)
			if len(line) == 0 || strings.HasPrefix(line, "#") {
				continue
			}
			if !strings.Contains(line, "=") {
				return ErrInvalidEnvLine{
					filePath:   path,
					lineNumber: lineNumber,
					line:       scanner.Text(),
				}
			}
			e.Env = append(e.Env, line)
		}
	}
	if e.Path == "" {
		// envEntry are of the form xxx=yyy
		for _, envEntry := range e.Env {
			fmt.Println(envEntry)
		}
		return nil
	}
	// Setup file descriptors for sub process
	e.Stdin = os.Stdin
	e.Stdout = os.Stdout
	e.Stderr = os.Stderr
	// Run subprocess and return
	if err := e.Run(); err != nil {
		return fmt.Errorf("failed running %v: %w", e.Args, err)
	}
	return nil
}

type ErrInvalidEnvLine struct {
	filePath   string
	lineNumber int
	line       string
}

func (err ErrInvalidEnvLine) Error() string {
	return fmt.Sprintf("%s: line %d doesn't contain an '=': %s", err.filePath, err.lineNumber, err.line)
}

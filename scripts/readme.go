package main

import (
	"flight-tracker-slack/commands"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	templateBytes, err := os.ReadFile("assets/TEMPLATE.md")
	if err != nil {
		panic(fmt.Errorf("failed to read template: %w", err))
	}
	content := string(templateBytes)

	var commands_string strings.Builder
	for _, cmd := range commands.CommandList {
		fmt.Fprintf(&commands_string, "- `%s`: %s\n", cmd.Name, cmd.Description)
	}

	re := regexp.MustCompile(`{{commands}}`)
	finalContent := re.ReplaceAllString(content, commands_string.String())

	err = os.WriteFile("README.md", []byte(finalContent), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write README: %w", err))
	}

	fmt.Println("Successfully updated README.md")
}

package app

import (
	"context"
	"fmt"
)

// Config manages configuration.
func (a *App) Config(ctx context.Context, listTemplates bool, setTemplate string) error {
	cfg, err := a.loadConfig()
	if err != nil {
		return err
	}

	if listTemplates {
		return a.listTemplates()
	}

	// Show current config
	fmt.Fprintf(a.Out, "default_template: %s\n", cfg.DefaultTemplate)
	return nil
}

func (a *App) listTemplates() error {
	templates, err := a.getTemplateNames()
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		fmt.Fprintln(a.Out, "No templates found")
		return nil
	}
	for _, name := range templates {
		fmt.Fprintln(a.Out, name)
	}
	return nil
}

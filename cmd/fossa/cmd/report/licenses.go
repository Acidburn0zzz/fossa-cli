package report

import (
	"fmt"
	"text/template"

	"github.com/fossas/fossa-cli/cmd/fossa/cmdutil"
	"github.com/urfave/cli"

	"github.com/fossas/fossa-cli/api/fossa"
	"github.com/fossas/fossa-cli/cmd/fossa/flags"
	"github.com/fossas/fossa-cli/log"
)

const defaultLicenseReportTemplate = `# 3rd-Party Software License Notice
Generated by fossa-cli (https://github.com/fossas/fossa-cli).
This software includes the following software and licenses:
{{range $license, $deps := .}}
========================================================================
{{$license}}
========================================================================
The following software have components provided under the terms of this license:
{{range $i, $dep := $deps}}
- {{$dep.Project.Title}} (from {{$dep.Project.URL}})
{{- end}}
{{end}}
`

var licensesCmd = cli.Command{
	Name:  "licenses",
	Usage: "Generate licenses report",
	Flags: flags.WithGlobalFlags(flags.WithAPIFlags(flags.WithModulesFlags(flags.WithReportTemplateFlags([]cli.Flag{
		cli.BoolFlag{Name: flags.Short(Unknown), Usage: "license report including unkown (warning this is SLOW)"},
	})))),
	Before: prepareReportCtx,
	Action: generateLicenses,
}

func generateLicenses(ctx *cli.Context) (err error) {
	defer log.StopSpinner()
	revs := make([]fossa.Revision, 0)
	for _, module := range analyzed {
		if ctx.Bool(Unknown) {
			totalDeps := len(module.Deps)
			i := 0
			for _, dep := range module.Deps {
				i++
				log.ShowSpinner(fmt.Sprintf("Fetching License Info (%d/%d): %s", i+1, totalDeps, dep.ID.Name))
				rev, err := fossa.GetRevision(fossa.LocatorOf(dep.ID))
				if err != nil {
					log.Logger.Warning(err.Error())
					continue
				}
				revs = append(revs, rev)
			}
		} else {
			log.ShowSpinner("Fetching License Info")
			var locators []fossa.Locator
			for _, dep := range module.Deps {
				locators = append(locators, fossa.LocatorOf(dep.ID))
			}
			revs, err = fossa.GetRevisions(locators)
			if err != nil {
				log.Logger.Fatalf("Could not fetch revisions: %s", err.Error())
			}
		}
	}

	depsByLicense := make(map[string]map[string]fossa.Revision, 0)
	for _, rev := range revs {
		for _, license := range rev.Licenses {
			if _, ok := depsByLicense[license.LicenseID]; !ok {
				depsByLicense[license.LicenseID] = make(map[string]fossa.Revision, 0)
			}
			depsByLicense[license.LicenseID][rev.Locator.String()] = rev
		}
	}

	tmpl, err := template.New("base").Parse(defaultLicenseReportTemplate)
	if err != nil {
		log.Logger.Fatalf("Could not parse template data: %s", err.Error())
	}

	if ctx.String(flags.Template) != "" {
		tmpl, err = template.ParseFiles(ctx.String(flags.Template))
		if err != nil {
			log.Logger.Fatalf("Could not parse template data: %s", err.Error())
		}
	}
	log.StopSpinner()

	return cmdutil.OutputData(ctx.String(flags.ShowOutput), tmpl, depsByLicense)
}

// gendocs writes docs/resources and docs/data-sources from the provider's own
// schema, in the Terraform Registry's layout, with each page's example from
// examples/<kind>/<name>/example.tf. Run from the module root:
//
//	go run ./tools/gendocs
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getunderpass/terraform-provider-underpass/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func main() {
	server := providerserver.NewProtocol6(provider.New("docs")())()
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		log.Fatal(err)
	}
	write("resources", "Resource", resp.ResourceSchemas)
	write("data-sources", "Data Source", resp.DataSourceSchemas)
}

func write(dir, kind string, schemas map[string]*tfprotov6.Schema) {
	for name, s := range schemas {
		short := strings.TrimPrefix(name, "underpass_")
		var b strings.Builder
		fmt.Fprintf(&b, "---\npage_title: %q\ndescription: |-\n  %s\n---\n\n", "underpass_"+short+" ("+kind+")", firstSentence(s.Block.Description))
		fmt.Fprintf(&b, "# underpass_%s (%s)\n\n%s\n\n", short, kind, s.Block.Description)
		if ex, err := os.ReadFile(filepath.Join("examples", dir, name, "example.tf")); err == nil {
			fmt.Fprintf(&b, "## Example Usage\n\n```terraform\n%s```\n\n", strings.TrimLeft(string(ex), "\n"))
		}
		b.WriteString("## Schema\n")
		groups := map[string][]*tfprotov6.SchemaAttribute{}
		for _, a := range s.Block.Attributes {
			g := "Read-Only"
			switch {
			case a.Required:
				g = "Required"
			case a.Optional:
				g = "Optional"
			}
			groups[g] = append(groups[g], a)
		}
		for _, g := range []string{"Required", "Optional", "Read-Only"} {
			attrs := groups[g]
			if len(attrs) == 0 {
				continue
			}
			sort.Slice(attrs, func(i, j int) bool { return attrs[i].Name < attrs[j].Name })
			fmt.Fprintf(&b, "\n### %s\n\n", g)
			for _, a := range attrs {
				sens := ""
				if a.Sensitive {
					sens = ", Sensitive"
				}
				fmt.Fprintf(&b, "- `%s` (%s%s) %s\n", a.Name, typeName(a), sens, a.Description)
			}
		}
		if imp, err := os.ReadFile(filepath.Join("examples", dir, name, "import.sh")); err == nil {
			fmt.Fprintf(&b, "\n## Import\n\n```shell\n%s```\n", strings.TrimLeft(string(imp), "\n"))
		}
		out := filepath.Join("docs", dir, short+".md")
		if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", out)
	}
}

func typeName(a *tfprotov6.SchemaAttribute) string {
	if a.NestedType != nil {
		return "Attributes List"
	}
	if a.Type == nil {
		return "Unknown"
	}
	t := a.Type.String()
	switch {
	case strings.HasPrefix(t, "tftypes.List"):
		return "List of String"
	case strings.HasPrefix(t, "tftypes.Set"):
		return "Set of String"
	case strings.HasPrefix(t, "tftypes.Map"):
		return "Map of List of String"
	case strings.Contains(t, "Bool"):
		return "Boolean"
	case strings.Contains(t, "Number"):
		return "Number"
	default:
		return "String"
	}
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	return strings.ReplaceAll(s, "\n", " ")
}

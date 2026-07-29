package dceskills_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/lathe-cli/lathe/pkg/runtime"
)

const (
	clusterDiagnosisSkillPath = "skills/container-management:cluster-diagnosis/SKILL.md"
	podDiagnosisSkillPath     = "skills/container-management:pod-diagnosis/SKILL.md"
)

var containerManagementCommand = regexp.MustCompile(`\bdce\s+container-management\s+([a-z0-9-]+)\s+([a-z0-9-]+)\b`)

// These skills are hand-written guidance over the generated CLI. Keep their
// command snippets aligned with the catalog so agents do not receive a command
// that is valid-looking but targets the wrong resource or filter field.
func TestContainerDiagnosisSkillCommandContracts(t *testing.T) {
	root := buildRoot(t)
	clusterSkill := readContainerDiagnosisSkill(t, clusterDiagnosisSkillPath)
	podSkill := readContainerDiagnosisSkill(t, podDiagnosisSkillPath)

	t.Run("cluster pod discovery uses the cluster-wide endpoint", func(t *testing.T) {
		mustContainContainerDiagnosisCommand(t, clusterSkill,
			"dce container-management core list-cluster-pods",
			"--cluster <cluster>", "--all", "-o json",
		)
		mustNotContainContainerDiagnosisSkill(t, clusterSkill,
			"dce container-management core list-pods --cluster",
		)
		mustCatalogFlag(t, root, []string{"container-management", "core", "list-cluster-pods"}, "cluster")
	})

	t.Run("pod filters use the involved-object name field", func(t *testing.T) {
		mustContainContainerDiagnosisCommand(t, podSkill,
			"dce container-management core list-events",
			"--cluster <cluster>", "--namespace <namespace>", "--kind Pod", "--kind-name <pod>", "--all", "-o json",
		)
		mustNotContainContainerDiagnosisSkill(t, podSkill,
			"dce container-management core list-events --cluster <cluster> --namespace <namespace> --kind Pod --name <pod> -o json",
		)
		mustContainContainerDiagnosisCommand(t, podSkill,
			"dce container-management core list-pods",
			"--cluster <cluster>", "--namespace <namespace>", "--kind <owner-kind>", "--kind-name <owner-name>", "--all", "-o json",
		)
		mustNotContainContainerDiagnosisSkill(t, podSkill,
			"dce container-management core list-pods --cluster <cluster> --namespace <namespace> --kind <owner-kind> --name <owner-name> -o json",
		)
		mustCatalogFlag(t, root, []string{"container-management", "core", "list-events"}, "kind-name")
		mustCatalogFlag(t, root, []string{"container-management", "core", "list-pods"}, "kind-name")
		mustCatalogFlag(t, root, []string{"container-management", "core", "list-cluster-events"}, "name")
		mustNotCatalogFlag(t, root, []string{"container-management", "core", "list-cluster-events"}, "kind-name")
	})

	t.Run("pod diagnosis takes one exact pod snapshot", func(t *testing.T) {
		mustContainContainerDiagnosisCommand(t, podSkill,
			"dce container-management core get-pod",
			"--cluster <cluster>", "--namespace <namespace>", "--name <pod>", "-o json",
		)
		if count := strings.Count(podSkill, "dce container-management core get-pod"); count != 1 {
			t.Errorf("SKILL.md should collect the exact Pod once, found %d get-pod snippets", count)
		}
	})

	for _, skill := range []struct {
		path string
		body string
	}{
		{path: clusterDiagnosisSkillPath, body: clusterSkill},
		{path: podDiagnosisSkillPath, body: podSkill},
	} {
		t.Run("catalog references/"+skill.path, func(t *testing.T) {
			assertContainerDiagnosisCatalogReferences(t, root, skill.path, skill.body)
		})
	}
}

func readContainerDiagnosisSkill(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func mustContainContainerDiagnosisCommand(t *testing.T, skill, command string, flags ...string) {
	t.Helper()
	for _, line := range strings.Split(skill, "\n") {
		if !strings.Contains(line, command) {
			continue
		}
		allFlagsPresent := true
		for _, flag := range flags {
			if !strings.Contains(line, flag) {
				allFlagsPresent = false
				break
			}
		}
		if allFlagsPresent {
			return
		}
	}
	t.Errorf("SKILL.md missing command %q with flags %q", command, flags)
}

func mustNotContainContainerDiagnosisSkill(t *testing.T, skill, unwanted string) {
	t.Helper()
	if strings.Contains(skill, unwanted) {
		t.Errorf("SKILL.md contains obsolete command form %q", unwanted)
	}
}

func assertContainerDiagnosisCatalogReferences(t *testing.T, root *cobra.Command, skillPath, skill string) {
	t.Helper()
	matches := containerManagementCommand.FindAllStringSubmatch(skill, -1)
	if len(matches) == 0 {
		t.Fatalf("%s does not reference any container-management catalog command", skillPath)
	}

	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		path := []string{"container-management", match[1], match[2]}
		key := strings.Join(path, " ")
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}

		if _, ok := runtime.FindCatalogCommand(root, path, runtime.CatalogOptions{}); !ok {
			t.Errorf("%s references `dce %s`, but it is not in the command catalog", skillPath, key)
		}
	}
}

func mustCatalogFlag(t *testing.T, root *cobra.Command, path []string, want string) {
	t.Helper()
	command, ok := runtime.FindCatalogCommand(root, path, runtime.CatalogOptions{})
	if !ok {
		t.Fatalf("catalog command `dce %s` is missing", strings.Join(path, " "))
	}
	for _, flag := range command.Flags {
		if flag.Flag == want {
			return
		}
	}
	t.Errorf("catalog command `dce %s` is missing --%s", strings.Join(path, " "), want)
}

func mustNotCatalogFlag(t *testing.T, root *cobra.Command, path []string, unwanted string) {
	t.Helper()
	command, ok := runtime.FindCatalogCommand(root, path, runtime.CatalogOptions{})
	if !ok {
		t.Fatalf("catalog command `dce %s` is missing", strings.Join(path, " "))
	}
	for _, flag := range command.Flags {
		if flag.Flag == unwanted {
			t.Errorf("catalog command `dce %s` unexpectedly exposes --%s", strings.Join(path, " "), unwanted)
		}
	}
}

package support_bundle

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/pmezard/go-difflib/difflib"
)

func TestIntegration(t *testing.T) {
	for _, dir := range listDirs(t, ".") {
		t.Run(dir, func(t *testing.T) {
			ns := fmt.Sprintf("integration-test-%s", randomString(8))

			createNamespace(t, ns)
			defer deleteNamespace(t, ns)

			tmp := tempDir(t)
			defer os.RemoveAll(tmp)

			generateFixtures(t, dir, ns)

			bundle := generateSupportBundle(t, dir, ns, tmp)

			for _, file := range listFilesRecursive(t, filepath.Join(dir, "expects")) {
				if err := expectsFileSupportBundle(t, dir, ns, tmp, bundle, file); err != nil {
					t.Errorf("%s ERROR\n%s", file, err)
				}
			}
		})
	}
}

func expectsFileSupportBundle(t *testing.T, dir string, ns string, tmp string, bundle string, file string) error {
	expects := templateFile(t, ns, filepath.Join(dir, "expects", file))

	f, err := os.Open(filepath.Join(tmp, bundle))
	if err != nil {
		return errors.Wrap(err, "open bundle file")
	}

	zr, err := gzip.NewReader(f)
	if err != nil {
		return errors.Wrap(err, "new gzip reader")
	}
	defer zr.Close()

	tr := tar.NewReader(zr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			return errors.New("file not found")
		} else if err != nil {
			return errors.Wrap(err, "next file")
		}

		if strings.SplitN(header.Name, "/", 2)[1] == strings.Replace(file, "[namespace]", ns, -1) {
			actual, err := ioutil.ReadAll(tr)
			if err != nil {
				return errors.Wrap(err, "read file")
			}

			if bytes.Equal(expects, actual) {
				t.Logf("%s: OK", file)
				return nil
			}

			diff, err := diff(actual, expects)
			if err != nil {
				return errors.Wrap(err, "generate diff")
			}
			return fmt.Errorf("diff:\n%s", diff)
		}
	}
}

func generateSupportBundle(t *testing.T, dir string, ns string, tmp string) string {
	data := templateFile(t, ns, filepath.Join(dir, "spec.yaml"))

	if err := ioutil.WriteFile(filepath.Join(tmp, "spec.yaml"), data, 0644); err != nil {
		t.Fatalf("failed to write spec: %v", err)
	}

	cmd := exec.Command(getSupportBundleBinary(), "spec.yaml")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("failed to generate support bundle: %v", err)
	}

	bundle := regexp.MustCompile(`support-bundle-.*\.tar\.gz`).FindString(string(out))
	t.Logf("support bundle %s generated", bundle)

	return bundle
}

func getSupportBundleBinary() string {
	if bin := os.Getenv("SUPPORT_BUNDLE_BINARY"); bin != "" {
		return bin
	}
	return "kubectl-support_bundle"
}

func generateFixtures(t *testing.T, dir string, ns string) {
	cmd := exec.Command("kubectl", "--namespace", ns, "apply", "-f", filepath.Join(dir, "fixtures"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("failed to generate fixtures: %v", err)
	}

	t.Log(string(out))
	t.Log("fixtures generated")
}

func createNamespace(t *testing.T, ns string) {
	_ = exec.Command("kubectl", "delete", "ns", ns).Run()
	if err := exec.Command("kubectl", "create", "ns", ns).Run(); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	t.Logf("namespace %s created", ns)
}

func deleteNamespace(t *testing.T, ns string) {
	_ = exec.Command("kubectl", "delete", "ns", ns).Run()
	t.Logf("namespace %s deleted", ns)
}

func diff(got, want []byte) (string, error) {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(got)),
		B:        difflib.SplitLines(string(want)),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	return difflib.GetUnifiedDiffString(diff)
}

func templateFile(t *testing.T, ns string, file string) []byte {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	tmpl, err := template.New(file).Parse(string(data))
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.Execute(buf, map[string]string{"Namespace": ns}); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	return buf.Bytes()
}

func listFilesRecursive(t *testing.T, dir string) (files []string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk directory: %s", err)
	}
	return files
}

func listDirs(t *testing.T, dir string) (dirs []string) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read directory: %s", err)
	}
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		dirs = append(dirs, info.Name())
	}
	return dirs
}

func tempDir(t *testing.T) string {
	tmp, err := ioutil.TempDir("", "test-")
	if err != nil {
		t.Fatalf("failed to temp dir: %v", err)
	}
	return tmp
}

var seededRand *rand.Rand

func init() {
	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

var charset = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

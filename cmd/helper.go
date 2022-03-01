package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/exp/utf8string"

	"github.com/cage1016/document-translator-cli/lib"
)

type promptContent struct {
	errorMsg string
	label    string
}

func promptGetInput(pc promptContent, deVal string) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(pc.errorMsg)
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     pc.label,
		Templates: templates,
		Validate:  validate,
		Default:   deVal,
	}

	result, err := prompt.Run()
	if err != nil {
		logrus.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func promptGetSelect(pc promptContent, items []string) string {
	index := -1
	var result string
	var err error

	for index < 0 {
		prompt := promptui.SelectWithAdd{
			Label:    pc.label,
			Items:    items,
			AddLabel: "Other",
		}

		index, result, err = prompt.Run()

		if index == -1 {
			items = append(items, result)
		}
	}

	if err != nil {
		logrus.Fatalf("Prompt failed %v\n", err)
	}

	return result
}

func documentsSelect2(docs []lib.Document, label string) (*lib.Document, error) {

	funcMap := promptui.FuncMap
	funcMap["truncate"] = func(s string, l int) string {
		a := utf8string.NewString(s)
		if a.RuneCount() <= l {
			t := "%-" + strconv.Itoa(l) + "s"
			return fmt.Sprintf(t, s)
		}
		p := (l / 2) - 1
		return a.Slice(0, p) + "..." + a.Slice(a.RuneCount()-p, a.RuneCount()-1)
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F336 {{ truncate .Filename 30 | cyan }} {{ truncate (printf \"%s → %s\" .Source .Target) 15 | yellow }} ({{ .Status | red }})",
		Inactive: "  {{ truncate .Filename 30 | cyan }} {{ truncate (printf \"%s → %s\" .Source .Target) 15 }} ({{ .Status | red }})",
		Selected: "\U0001F336 {{truncate .Filename 30 | red | cyan }} {{ truncate (printf \"%s → %s\" .Source .Target) 15 | yellow }}",
		Details: `
--------- Document ----------
{{ "DocumentID:" | faint }}	{{ .DocumentID }}
{{ "Filename:" | faint }}	{{ .Filename }}
{{ "Status:" | faint }}	{{ .Status }}
{{ "ModelID:" | faint }}	{{ .ModelID }}
{{ "Source:" | faint }}	{{ .Source }}
{{ "Target:" | faint }}	{{ .Target }}
{{ "WordCount:" | faint }}	{{ .WordCount }}
{{ "CharacterCount:" | faint }}	{{ .CharacterCount }}
{{ "Created:" | faint }}	{{ .Created }}
{{ "Completed:" | faint }}	{{ .Completed }}`,
		FuncMap: funcMap,
	}

	searcher := func(input string, index int) bool {
		pepper := docs[index]
		name := strings.Replace(strings.ToLower(pepper.Filename), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     docs,
		Templates: templates,
		Size:      4,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()

	if err != nil {
		logrus.Errorf("Prompt failed %v\n", err)
		return nil, err
	}

	return &docs[i], nil
}

type action struct {
	Name  string
	Value int
}

func filter(vs []lib.Document, f func(string) bool) []lib.Document {
	filtered := make([]lib.Document, 0)
	for _, v := range vs {
		if f(v.DocumentID) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func promptGetActionSelect(actions []action) int {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F336 {{ .Name | cyan }}",
		Inactive: "  {{ .Name | cyan }}",
		Selected: "\U0001F336 {{ .Name | red | cyan }}",
	}

	prompt := promptui.Select{
		Label:     "Action",
		Items:     actions,
		Templates: templates,
		Size:      3,
	}

	i, _, err := prompt.Run()
	if err != nil {
		logrus.Fatalf("Prompt failed %v\n", err)
		return -1
	}
	return i
}

func loadList() []lib.Document {
	logrus.Info("Fetching Documents list...")
	res, err := lib.ListDocument(lib.ListRequest{
		Version: viper.GetString("version"),
		APIKey:  viper.GetString("api_key"),
		URL:     viper.GetString("url"),
	})
	if err != nil {
		logrus.Fatalf("Error Fetching documents: %s", err)
		return []lib.Document{}
	}

	buf := lib.AutoGenerated{}
	err = json.Unmarshal(res, &buf)
	if err != nil {
		logrus.Fatalf("Error unmarshaling documents: %s", err)
		return []lib.Document{}
	}

	sort.SliceStable(buf.Documents, func(i, j int) bool {
		return buf.Documents[i].Created.UnixNano() > buf.Documents[j].Created.UnixNano()
	})

	return buf.Documents
}

func deleteDocument(doc *lib.Document) {
	req := &lib.DeleteRequest{
		Version:    viper.GetString("version"),
		APIKey:     viper.GetString("api_key"),
		URL:        viper.GetString("url"),
		DocumentID: doc.DocumentID,
	}
	lib.DeleteDocument(req)
}

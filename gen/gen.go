package gen

import (
	"github.com/getevo/evo/lib/log"
	"github.com/moznion/gowrtr/generator"
	"ies/oscript"
	"path/filepath"
	"strings"
)

func OpenAndGenerate(path string) (string, error) {
	var root, err = oscript.OpenAndParse(path)
	if err != nil {
		return "", err
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var pkg = root.GetSingleValue("package")
	if pkg == "" {
		pkg = filepath.Base(filepath.Dir(absolutePath))
	}
	var namespace = Namespace{
		Name:        pkg,
		Description: root.GetSingleValue("Description"),
		Objects:     []Object{},
	}
	var model = generator.NewRoot(
		generator.NewPackage(namespace.Name),
		generator.NewComment(namespace.Description),
		generator.NewNewline(),
	)

	//First lvl object name scanner
	DFS(root, func(ctx *oscript.Context, params ...interface{}) error {
		if ok, _ := ctx.GetChild("Fields"); ok {
			namespace.Objects = append(namespace.Objects, Object{
				Namespace: &namespace,
				Name:      ctx.Name,
				IsModel:   ctx.VaryDictHas("Tag", "Model"),
				Fields:    []Field{},
			})
		}
		return nil
	})

	//Second lvl object detail scanner
	err = DFS(root, func(ctx *oscript.Context, params ...interface{}) error {
		if ok, _ := ctx.GetChild("Fields"); ok {
			var obj, err = namespace.GetObject(ctx.Name).parse(ctx)
			if err != nil {
				return err
			}
			if obj.IsModel {
				model = model.AddStatements(obj.getModelStatement(ctx)...)
			}
		}
		return nil
	})
	if err != nil {
		log.Error(err)
	}
	generated, err := model.Gofmt("-s").Goimports().Generate(0)
	if err != nil {
		return "", err
	}
	_ = generated

	return generated, err
}

func DFS(root *oscript.Context, fn func(ctx *oscript.Context, params ...interface{}) error) error {
	for idx, _ := range root.Children {
		err := DFS(root.Children[idx], fn)
		if err != nil {
			return err
		}
		err = fn(root.Children[idx])
		if err != nil {
			return err
		}
	}
	return nil
}

func singleQuote(input string) string {
	return "'" + strings.TrimFunc(input, func(r rune) bool {
		if r == '"' || r == '\'' || r == '(' || r == ')' {
			return true
		}
		return false
	}) + "'"
}

func removeQuote(input string) string {
	return strings.TrimFunc(input, func(r rune) bool {
		if r == '"' || r == '\'' || r == '(' || r == ')' {
			return true
		}
		return false
	})
}

package gen

import (
	"fmt"
	"github.com/iancoleman/strcase"
	"github.com/moznion/gowrtr/generator"
	"ies/oscript"
	"strings"
)

type Namespace struct {
	Name        string
	Description string
	Objects     []Object
}

type Object struct {
	Namespace   *Namespace
	Name        string
	Fields      []Field
	IsModel     bool
	PrimaryKeys []*Field
	TableName   string
	Relations   []*Object
}

func (obj *Object) parse(ctx *oscript.Context) (*Object, error) {
	if ok, fields := ctx.GetChild("Fields"); ok {

		for _, field := range fields.VaryDict {
			var f = Field{
				ORM: ORM{},
			}
			var skipModel = false
			f.Name = field.Key
			if strings.Contains(f.Name, "*") {
				f.Nullable = true
			}
			f.ORM.DBName = field.VaryDictGetVar("column")
			if f.ORM.DBName == "" {
				f.ORM.DBName = strcase.ToSnake(field.Key)
			}

			f.Json = field.VaryDictGetVar("json")
			if f.Json == "" {
				f.Json = strcase.ToSnake(field.Key)
			}

			f.Type = "string"
			for _, item := range field.Values {
				switch item {
				case "pk", "primary", "primaryKey":
					f.ORM.PrimaryKey = true
				case "autoIncrement", "auto", "increment", "incremental":
					f.ORM.AutoIncrement = true
				case "null", "nullable", "nil":
					f.ORM.Nullable = true
					f.Nullable = true
				case "bigint":
					f.Type = "Int64"
					f.ORM.Type = "bigint"
				case "int64":
					f.Type = "Int64"
					f.ORM.Type = "bigint"
				case "uint64":
					f.Type = "UInt64"
					f.ORM.Type = "bigint"
				case "int":
					f.Type = "int"
					f.ORM.Type = "int"
				case "uint":
					f.Type = "uint"
					f.ORM.Type = "uint"
				case "text":
					f.Type = "string"
					f.ORM.Type = "text"
				case "string":
					f.Type = "string"
					f.ORM.Type = "text"
				case "date", "datetime", "timestamp":
					f.Type = "time.Time"
					f.ORM.Type = "timestamp"
				default:
					if strings.HasPrefix(item, "varchar") {
						f.ORM.Type = "varchar"
						f.ORM.Size = oscript.ParseVar(item, "varchar")
					} else if strings.HasPrefix(item, "enum") {
						f.ORM.Enum = strings.Split(oscript.ParseVar(item, "enum"), ",")
						f.ORM.Type = "enum"
					} else if strings.HasPrefix(item, "default") {
						var v = oscript.ParseVar(item, "default")
						if strings.ToLower(v) == "now" {
							v = "CURRENT_TIMESTAMP()"
						}
						f.ORM.Default = v
					} else if strings.HasPrefix(item, "size") {
						f.ORM.Size = removeQuote(oscript.ParseVar(item, "size"))
					} else if strings.HasPrefix(item, "precision") {
						f.ORM.Precision = removeQuote(oscript.ParseVar(item, "precision"))
					} else if strings.HasPrefix(item, "comment") {
						f.ORM.Comment = removeQuote(oscript.ParseVar(item, "comment"))
					} else if strings.HasPrefix(item, "type") {
						f.ORM.Type = removeQuote(oscript.ParseVar(item, "type"))
					} else if strings.HasPrefix(item, "unique") {
						var v = oscript.ParseVar(item, "unique")
						if v != "" {
							f.ORM.UniqueIndex = v
						} else {
							f.ORM.SelfUnique = true
						}

					} else if strings.HasPrefix(item, "index") {
						var v = oscript.ParseVar(item, "index")
						if v != "" {
							f.ORM.Index = append(f.ORM.Index, v)
						} else {
							f.ORM.SelfIndex = true
						}
					} else {
						m := item
						var ptr = false
						var array = false
						if len(m) > 0 && m[0] == '*' {
							ptr = true
							m = m[1:]
						} else if len(m) > 1 && m[0] == '[' && m[1] == ']' {
							array = true
							m = m[2:]
						}

						for _, object := range obj.Namespace.Objects {
							if object.Name == m {
								f.Type = item
								if object.IsModel {
									//append helper fields
									if len(object.PrimaryKeys) == 0 {
										return nil, fmt.Errorf("cant find %s primary key", object.Name)
									}
									f.ORM.Relation = &object

									if ptr {
										f.ORM.Nullable = true
										f.ORM.Constraint += "OnUpdate:CASCADE,OnDelete:SET NULL;"
									}
									f.ORM.RelationField = &Field{
										Name: object.PrimaryKeys[0].Name,
										Type: object.PrimaryKeys[0].Type,
										Tags: Tag{"gorm:\"column:" + object.PrimaryKeys[0].ORM.DBName + "\"", "json:\"" + object.PrimaryKeys[0].Json + "\""},
									}
									if !array {
										obj.Fields = append(obj.Fields, *f.ORM.RelationField)
									}
									f.ORM.RelationType = "has-one"
									if array {
										f.ORM.RelationType = "has-many"
									}

								} else {
									skipModel = true
								}
								break
							}
						}

					}
				}
			}

			if obj.IsModel && !skipModel {
				f.Tags = append(f.Tags, f.ORM.ToString())
			}

			f.Tags = append(f.Tags, "json:\""+f.Json+"\"")
			obj.Fields = append(obj.Fields, f)
			if f.ORM.PrimaryKey {
				obj.PrimaryKeys = append(obj.PrimaryKeys, &f)
			}
		}
	}
	return obj, nil
}

func (obj *Object) getModelStatement(ctx *oscript.Context) []generator.Statement {
	var result []generator.Statement
	var st = generator.NewStruct(obj.Name)

	for _, field := range obj.Fields {
		var ptr = ""
		if field.Nullable && !strings.Contains(field.Name, "*") {
			ptr = "*"
		}
		st = st.AddField(strcase.ToCamel(field.Name), ptr+field.Type, field.Tags.ToString())
	}

	if description := ctx.GetSingleValue("Description"); description != "" {
		result = append(result, generator.NewComment(ctx.Name+" "+description))
	}
	result = append(result, st)
	if table := ctx.GetSingleValue("Table"); table != "" {
		var tableName = generator.NewFunc(
			generator.NewFuncReceiver("this", ctx.Name),
			generator.NewFuncSignature("TableName").
				ReturnTypeStatements(generator.NewFuncReturnType("string")),
			generator.NewRawStatement("return \""+table+"\""),
		)
		result = append(result, tableName)
	}
	return result
}

type Field struct {
	Name     string
	Type     string
	Tags     Tag
	Json     string
	ORM      ORM
	Nullable bool
}

type ORM struct {
	Skip          bool
	DBName        string
	AutoIncrement bool
	PrimaryKey    bool
	Relation      *Object
	RelationType  string
	RelationField *Field
	Nullable      bool
	Default       string
	Precision     string
	SelfIndex     bool
	Index         []string
	SelfUnique    bool
	UniqueIndex   string
	Size          string
	Type          string
	Constraint    string
	Enum          []string
	Comment       string
}

func (o ORM) ToString() string {
	var tag = "gorm:\""
	if o.Skip {
		return tag + "-\""
	}

	if o.RelationType != "" {
		if o.RelationType == "has-one" {
			return tag + "foreignKey:" + o.Relation.PrimaryKeys[0].Name + ";references:" + o.RelationField.Name + ";\""
		}
		return tag + "foreignKey:" + o.Relation.PrimaryKeys[0].Name + "\""
	}

	tag += "column:" + o.DBName + ";"
	if o.Comment != "" {
		tag += "comment:" + o.Comment + ";"
	}
	if o.Size != "" {
		tag += "size:" + o.Size + ";"
	}
	if o.Precision != "" {
		tag += "precision:" + o.Precision + ";"
	}
	if o.Constraint != "" {
		tag += "constrain:" + o.Constraint + ";"
	}
	if o.Type != "" {
		if o.Type == "enum" {
			var enums = o.Enum
			for idx, v := range enums {
				enums[idx] = singleQuote(v)
			}
			tag += "type:enum(" + strings.Join(enums, ",") + ");"
		} else {
			if o.Size != "" {
				tag += "type:" + o.Type + "(" + o.Size + ")" + ";"
			} else {
				tag += "type:" + o.Type + ";"
			}

		}
	}
	if o.Default != "" {
		tag += "default:" + o.Default + ";"
	}
	if o.UniqueIndex != "" {
		tag += "uniqueIndex:" + o.UniqueIndex + ";"
	}
	if o.SelfIndex {
		tag += "index;"
	}
	if o.SelfUnique {
		tag += "unique;"
	}
	if !o.Nullable {
		tag += "not null;"
	}
	if o.PrimaryKey {
		tag += "primaryKey;"
	}
	if o.AutoIncrement {
		tag += "autoIncrement;"
	}
	for _, index := range o.Index {
		tag += "index:" + index + ";"
	}
	return tag + "\""
}

type Tag []string

func (t Tag) ToString() string {
	return strings.Join(t, " ")
}

func (ns Namespace) GetObject(name string) *Object {
	for idx, _ := range ns.Objects {
		if ns.Objects[idx].Name == name {
			return &ns.Objects[idx]
		}
	}
	return nil
}

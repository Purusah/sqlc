package mysql

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kyleconroy/sqlc/internal/dinosql"
	"vitess.io/vitess/go/vt/sqlparser"
)

func init() {
	initMockSchema()
}

const mockFileName = "test_data/queries.sql"
const mockConfigPath = "test_data/sqlc.json"

var mockSettings = dinosql.GenerateSettings{
	Version: "1",
	Packages: []dinosql.PackageSettings{
		dinosql.PackageSettings{
			Name: "db",
		},
	},
	Overrides: []dinosql.Override{},
}

func TestParseConfig(t *testing.T) {
	blob, err := ioutil.ReadFile(mockConfigPath)
	if err != nil {
		t.Fatal(err)
	}

	var settings dinosql.GenerateSettings
	if err := json.Unmarshal(blob, &settings); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratePkg(t *testing.T) {
	settings := dinosql.Combine(mockSettings, mockSettings.Packages[0])
	_, err := GeneratePkg(mockSettings.Packages[0].Name, mockFileName, mockFileName, settings)
	if err != nil {
		if pErr, ok := err.(*dinosql.ParserErr); ok {
			for _, fileErr := range pErr.Errs {
				t.Errorf("%s:%d:%d: %s\n", fileErr.Filename, fileErr.Line, fileErr.Column, fileErr.Err)
			}
		} else {
			t.Errorf("failed to generate pkg %s", err)
		}
	}
}

func keep(interface{}) {}

var mockSchema *Schema

func initMockSchema() {
	var schemaMap = make(map[string][]*sqlparser.ColumnDefinition)
	mockSchema = &Schema{
		tables: schemaMap,
	}
	schemaMap["users"] = []*sqlparser.ColumnDefinition{
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("first_name"),
			Type: sqlparser.ColumnType{
				Type:    "varchar",
				NotNull: true,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("last_name"),
			Type: sqlparser.ColumnType{
				Type:    "varchar",
				NotNull: false,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("id"),
			Type: sqlparser.ColumnType{
				Type:          "int",
				NotNull:       true,
				Autoincrement: true,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("age"),
			Type: sqlparser.ColumnType{
				Type:    "int",
				NotNull: true,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("job_status"),
			Type: sqlparser.ColumnType{
				Type:       "enum",
				NotNull:    true,
				EnumValues: []string{"applied", "pending", "accepted", "rejected"},
			},
		},
	}
	schemaMap["orders"] = []*sqlparser.ColumnDefinition{
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("id"),
			Type: sqlparser.ColumnType{
				Type:          "int",
				NotNull:       true,
				Autoincrement: true,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("price"),
			Type: sqlparser.ColumnType{
				Type:          "DECIMAL(13, 4)",
				NotNull:       true,
				Autoincrement: true,
			},
		},
		&sqlparser.ColumnDefinition{
			Name: sqlparser.NewColIdent("user_id"),
			Type: sqlparser.ColumnType{
				Type:    "int",
				NotNull: true,
			},
		},
	}
}

func filterCols(allCols []*sqlparser.ColumnDefinition, colNames map[string]string) []Column {
	cols := []Column{}
	for _, col := range allCols {
		if table, ok := colNames[col.Name.String()]; ok {
			cols = append(cols, Column{
				col,
				table,
			})
		}
	}
	return cols
}

func TestParseSelect(t *testing.T) {
	type expected struct {
		query  string
		schema *Schema
	}
	type testCase struct {
		name   string
		input  expected
		output *Query
	}
	tests := []testCase{
		testCase{
			name: "get_count",
			input: expected{
				query: `/* name: GetCount :one */
					SELECT id my_id, COUNT(id) id_count FROM users WHERE id > 4`,
				schema: mockSchema,
			},
			output: &Query{
				SQL: "select id as my_id, COUNT(id) as id_count from users where id > 4",
				Columns: []Column{
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("my_id"),
							Type: sqlparser.ColumnType{
								Type:          "int",
								NotNull:       true,
								Autoincrement: true,
							},
						},
						"users",
					},
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("id_count"),
							Type: sqlparser.ColumnType{
								Type:    "int",
								NotNull: true,
							},
						},
						"",
					},
				},
				Params:           []*Param{},
				Name:             "GetCount",
				Cmd:              ":one",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "get_name_by_id",
			input: expected{
				query: `/* name: GetNameByID :one */
									SELECT first_name, last_name FROM users WHERE id = ?`,
				schema: mockSchema,
			},
			output: &Query{
				SQL:     `select first_name, last_name from users where id = ?`,
				Columns: filterCols(mockSchema.tables["users"], map[string]string{"first_name": "users", "last_name": "users"}),
				Params: []*Param{
					&Param{
						OriginalName: ":v1",
						Name:         "id",
						Typ:          "int",
					}},
				Name:             "GetNameByID",
				Cmd:              ":one",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "get_all",
			input: expected{
				query: `/* name: GetAll :many */
				SELECT * FROM users;`,
				schema: mockSchema,
			},
			output: &Query{
				SQL:              "select first_name, last_name, id, age, job_status from users",
				Columns:          filterCols(mockSchema.tables["users"], map[string]string{"first_name": "users", "last_name": "users", "id": "users", "age": "users", "job_status": "users"}),
				Params:           []*Param{},
				Name:             "GetAll",
				Cmd:              ":many",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "get_all_users_orders",
			input: expected{
				query: `/* name: GetAllUsersOrders :many */
				SELECT u.id user_id, u.first_name, o.price, o.id order_id
				FROM orders o LEFT JOIN users u ON u.id = o.user_id`,
				schema: mockSchema,
			},
			output: &Query{
				SQL: "select u.id as user_id, u.first_name, o.price, o.id as order_id from orders as o left join users as u on u.id = o.user_id",
				Columns: []Column{
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("user_id"),
							Type: sqlparser.ColumnType{
								Type:          "int",
								Autoincrement: true,
								NotNull:       false, // beause of the left join
							},
						},
						"users",
					},
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("first_name"),
							Type: sqlparser.ColumnType{
								Type:    "varchar",
								NotNull: false, // because of left join
							},
						},
						"users",
					},
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("price"),
							Type: sqlparser.ColumnType{
								Type:          "DECIMAL(13, 4)",
								Autoincrement: true,
								NotNull:       true,
							},
						},
						"orders",
					},
					Column{
						&sqlparser.ColumnDefinition{
							Name: sqlparser.NewColIdent("order_id"),
							Type: sqlparser.ColumnType{
								Type:          "int",
								Autoincrement: true,
								NotNull:       true,
							},
						},
						"orders",
					},
				},
				Params:           []*Param{},
				Name:             "GetAllUsersOrders",
				Cmd:              ":many",
				DefaultTableName: "orders",
				SchemaLookup:     mockSchema,
			},
		},
	}

	settings := dinosql.Combine(mockSettings, mockSettings.Packages[0])
	for _, tt := range tests {
		testCase := tt
		t.Run(tt.name, func(t *testing.T) {
			qs, err := parseContents("example.sql", testCase.input.query, testCase.input.schema, settings)
			if err != nil {
				t.Fatalf("Parsing failed with query: [%v]\n", err)
			}
			if len(qs) != 1 {
				t.Fatalf("Expected one query, not %d", len(qs))
			}
			q := qs[0]
			q.SchemaLookup = nil
			q.Filename = ""
			testCase.output.SchemaLookup = nil
			if diff := cmp.Diff(testCase.output, q); diff != "" {
				t.Errorf("parsed query differs: \n%s", diff)
			}
		})
	}
}

func TestParseLeadingComment(t *testing.T) {
	type output struct {
		Name string
		Cmd  string
	}
	tests := []struct {
		input  string
		output output
	}{
		{
			input:  "/* name: GetPeopleByID :many */",
			output: output{Name: "GetPeopleByID", Cmd: ":many"},
		},
	}

	for _, tCase := range tests {
		name, cmd, err := dinosql.ParseMetadata(tCase.input, dinosql.CommentSyntaxStar)
		result := output{name, cmd}
		if err != nil {
			t.Errorf("failed to parse leading comment: %w", err)
		} else if diff := cmp.Diff(tCase.output, result); diff != "" {
			t.Errorf("unexpectd result of query metadata parse: %s", diff)
		}
	}
}

func TestSchemaLookup(t *testing.T) {
	firstNameColDfn, err := mockSchema.schemaLookup("users", "first_name")
	if err != nil {
		t.Errorf("Failed to get column schema from mock schema: %v", err)
	}

	expected := filterCols(mockSchema.tables["users"], map[string]string{"first_name": "users"})
	if !reflect.DeepEqual(Column{firstNameColDfn, "users"}, expected[0]) {
		t.Errorf("Table schema lookup returned unexpected result")
	}
}

func TestParseInsertUpdate(t *testing.T) {
	type expected struct {
		query  string
		schema *Schema
	}
	type testCase struct {
		name   string
		input  expected
		output *Query
	}

	tests := []testCase{
		testCase{
			name: "insert_users",
			input: expected{
				query:  "/* name: InsertNewUser :exec */\nINSERT INTO users (first_name, last_name) VALUES (?, ?)",
				schema: mockSchema,
			},
			output: &Query{
				SQL:     "insert into users(first_name, last_name) values (?, ?)",
				Columns: nil,
				Params: []*Param{
					&Param{
						OriginalName: ":v1",
						Name:         "first_name",
						Typ:          "string",
					},
					&Param{
						OriginalName: ":v2",
						Name:         "last_name",
						Typ:          "sql.NullString",
					},
				},
				Name:             "InsertNewUser",
				Cmd:              ":exec",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "update_without_where",
			input: expected{
				query:  "/* name: UpdateAllUsers :exec */ update users set first_name = 'Bob'",
				schema: mockSchema,
			},
			output: &Query{
				SQL:              "update users set first_name = 'Bob'",
				Columns:          nil,
				Params:           []*Param{},
				Name:             "UpdateAllUsers",
				Cmd:              ":exec",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "update_users",
			input: expected{
				query:  "/* name: UpdateUserAt :exec */\nUPDATE users SET first_name = ?, last_name = ? WHERE id > ? AND first_name = ? LIMIT 3",
				schema: mockSchema,
			},
			output: &Query{
				SQL:     "update users set first_name = ?, last_name = ? where id > ? and first_name = ? limit 3",
				Columns: nil,
				Params: []*Param{
					&Param{
						OriginalName: ":v1",
						Name:         "first_name",
						Typ:          "string",
					},
					&Param{
						OriginalName: ":v2",
						Name:         "last_name",
						Typ:          "sql.NullString",
					},
					&Param{
						OriginalName: ":v3",
						Name:         "id",
						Typ:          "int",
					},
					&Param{
						OriginalName: ":v4",
						Name:         "first_name",
						Typ:          "string",
					},
				},
				Name:             "UpdateUserAt",
				Cmd:              ":exec",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
		testCase{
			name: "insert_users_from_orders",
			input: expected{
				query:  "/* name: InsertUsersFromOrders :exec */\ninsert into users ( first_name ) select user_id from orders where id = ?;",
				schema: mockSchema,
			},
			output: &Query{
				SQL:     "insert into users(first_name) select user_id from orders where id = ?",
				Columns: nil,
				Params: []*Param{
					&Param{
						OriginalName: ":v1",
						Name:         "id",
						Typ:          "int",
					},
				},
				Name:             "InsertUsersFromOrders",
				Cmd:              ":exec",
				DefaultTableName: "users",
				SchemaLookup:     mockSchema,
			},
		},
	}

	settings := dinosql.Combine(mockSettings, mockSettings.Packages[0])
	for _, tt := range tests {
		testCase := tt
		t.Run(tt.name, func(t *testing.T) {
			qs, err := parseContents("example.sql", testCase.input.query, testCase.input.schema, settings)
			if err != nil {
				t.Fatalf("Parsing failed with query: [%v]\n", err)
			}
			if len(qs) != 1 {
				t.Fatalf("Expected one query, not %d", len(qs))
			}
			q := qs[0]
			testCase.output.SchemaLookup = nil
			q.SchemaLookup = nil
			q.Filename = ""
			if diff := cmp.Diff(testCase.output, q); diff != "" {
				t.Errorf("parsed query differs: \n%s", diff)
			}
		})
	}
}

package cypherx

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j/dbtype"
	"github.com/stretchr/testify/assert"
)

func TestIsValidPtr(t *testing.T) {
	s := struct{}{}
	assert.True(t, isValidPtr(&s))
	assert.False(t, isValidPtr(s))
	assert.False(t, isValidPtr(map[string]interface{}{}))
}

func TestGenerateAssignmentFunc(t *testing.T) {
	// 必ずstructのポインターからフィールドを取得する必要があります
	// reflect.Value の CanAddr()の条件を参照
	s := &struct {
		Name        sql.NullString
		Description string
		Age         int
		Height      float64
		Alive       bool
		Relative    struct {
			Name string
			Age  int
		}
	}{}

	// Name
	fv := reflect.ValueOf(s).Elem().Field(0)
	f, err := generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}

	if err := f(fv, "aaa"); err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.Name.Valid)
	assert.Equal(t, "aaa", s.Name.String)

	// Desc
	fv = reflect.ValueOf(s).Elem().Field(1)
	f, err = generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}

	if err := f(fv, "description"); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "description", s.Description)
	err = f(fv, 123)
	assert.Error(t, err)

	// Age
	fv = reflect.ValueOf(s).Elem().Field(2)
	f, err = generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}

	if err := f(fv, int64(10)); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, s.Age)

	// Height
	fv = reflect.ValueOf(s).Elem().Field(3)
	f, err = generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}

	if err := f(fv, float64(100)); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, float64(100), s.Height)

	// Alive
	fv = reflect.ValueOf(s).Elem().Field(4)
	f, err = generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}

	if err := f(fv, true); err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.Alive)

	// Relative
	fv = reflect.ValueOf(s).Elem().Field(5)
	f, err = generateAssignmentFunc(fv.Type())
	if err != nil {
		t.Fatal(err)
	}
	node := dbtype.Node{
		Props: map[string]interface{}{
			"name": "relative",
			"age":  int64(12),
		},
	}
	if err := f(fv, node); err != nil {
		t.Fatal(err)
	}
	expect := struct {
		Name string
		Age  int
	}{
		Name: "relative",
		Age:  12,
	}
	assert.Equal(t, expect, s.Relative)
}

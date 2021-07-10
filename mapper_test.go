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

	t.Run("scanner type field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(0)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}
		if err := f(fv, "name"); err != nil {
			t.Error(err)
		}
		assert.True(t, s.Name.Valid)
		assert.Equal(t, "name", s.Name.String)
	})

	t.Run("string field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(1)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}

		if err := f(fv, "description"); err != nil {
			t.Error(err)
		}
		assert.Equal(t, "description", s.Description)

		err = f(fv, 123)
		assert.Error(t, err)
	})

	t.Run("int field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(2)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}

		if err := f(fv, int64(10)); err != nil {
			t.Error(err)
		}
		assert.Equal(t, 10, s.Age)
		// must pass int64 type value
		assert.Error(t, f(fv, 1))
	})

	t.Run("float field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(3)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}

		if err := f(fv, float64(100)); err != nil {
			t.Error(err)
		}
		assert.Equal(t, float64(100), s.Height)
		assert.Error(t, f(fv, float32(100)))
	})

	t.Run("bool field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(4)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}

		if err := f(fv, true); err != nil {
			t.Error(err)
		}
		assert.True(t, s.Alive)
	})

	t.Run("struct field", func(t *testing.T) {
		fv := reflect.ValueOf(s).Elem().Field(5)
		f, err := generateAssignmentFunc(fv.Type())
		if err != nil {
			t.Error(err)
		}
		node := dbtype.Node{
			Props: map[string]interface{}{
				"name": "relative",
				"age":  int64(12),
			},
		}
		if err := f(fv, node); err != nil {
			t.Error(err)
		}
		expect := struct {
			Name string
			Age  int
		}{
			Name: "relative",
			Age:  12,
		}
		assert.Equal(t, expect, s.Relative)
		// check mapper cache
		_, ok := typeMapperCache.mapping[reflect.TypeOf(s.Relative)]
		assert.True(t, ok)
	})
}

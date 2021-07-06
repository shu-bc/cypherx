package cypherx

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	m := mapper{}
	props := map[string]interface{}{
		"name":      "test",
		"age":       int64(3),
		"salary":    1000.1,
		"del":       true,
		"social_id": "aaaa",
	}
	p := &Person{}
	err := m.scan(p, props)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(
		t,
		&Person{
			Name:     "test",
			Age:      3,
			Salary:   1000.1,
			Deleted:  true,
			SocialID: sql.NullString{String: "aaaa", Valid: true},
		},
		p)
}

func TestMapAll(t *testing.T) {
	m := mapper{}

	var ps []Person
	t.Run("type check", func(t *testing.T) {
		err := m.scanAll(ps, nil)
		assert.Error(t, err)
	})

	// t.Run("type check", func(t *testing.T) {
	// 	err := m.MapAll(&ps, nil)
	// 	assert.NoError(t, err)
	// })

	t.Run("type check", func(t *testing.T) {
		err := m.scanAll([]int{}, nil)
		assert.Error(t, err)
	})
}

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
}

package cypherx

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDest(t *testing.T) {
	s := struct{}{}
	assert.True(t, isValidDest(&s))
	assert.False(t, isValidDest(s))
	assert.False(t, isValidDest(map[string]interface{}{}))
}

func TestGenerateAssignmentFunc(t *testing.T) {
	// 必ずstructのポインターからフィールドを取得する必要があります
	// reflect.Value の CanAddr()の条件を参照
	s := &struct{ Name sql.NullString }{}
	fv := reflect.ValueOf(s).Elem().Field(0)

	f := generateAssignmentFunc(fv.Type())
	if err := f(fv, "aaa"); err != nil {
		t.Fatal(err)
	}
	assert.True(t, s.Name.Valid)
	assert.Equal(t, "aaa", s.Name.String)
}

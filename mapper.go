package cypherx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/ettle/strcase"
)

type Mapper struct {
	kindCache []reflect.Kind
	propNames []string
}

var _scannerIt = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func (m *Mapper) Map(dest interface{}, props map[string]interface{}) error {
	rt := reflect.TypeOf(dest)

	if !isValidDest(dest) {
		return fmt.Errorf("dest must be a pointer to struct\n")
	}

	if err := m.analyzeStruct(rt); err != nil {
		return err
	}

	for i, p := range m.propNames {
		pv, ok := props[p]
		if !ok {
			continue
		}

		rv := reflect.ValueOf(dest).Elem()
		field := rv.Field(i)

		if err := m.fillField(field, pv); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mapper) analyzeStruct(t reflect.Type) error {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("invalid type %s, expected struct\n", t.Kind().String())
	}

	if t.NumField() == 0 {
		return fmt.Errorf("struct must have > 1 field\n")
	}

	names := make([]string, 0, t.NumField())
	types := make([]reflect.Kind, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		tag := tf.Tag.Get("neo4j")
		propName := strings.Split(tag, ",")[0]
		if propName == "" {
			propName = strcase.ToSnake(tf.Name)
		}
		names = append(names, propName)
		types = append(types, t.Kind())
	}

	m.kindCache = types
	m.propNames = names

	return nil
}

func (m *Mapper) fillField(vf reflect.Value, pv interface{}) error {
	// sql.Scanner を満たすフィールドには Scan メソッドを呼び出す
	vfPtr := reflect.PtrTo(vf.Type())
	if vfPtr.Implements(_scannerIt) {
		vfAddr := vf.Addr()
		vfAddr.MethodByName("Scan").Call([]reflect.Value{reflect.ValueOf(pv)})
		return nil
	}

	switch vf.Kind() {
	case reflect.String:
		if s, ok := pv.(string); ok {
			vf.SetString(s)
		}

	case reflect.Int:
		if reflect.ValueOf(pv).Kind() == reflect.Int64 {
			vf.SetInt(reflect.ValueOf(pv).Int())
		}

	case reflect.Float64:
		if reflect.ValueOf(pv).Kind() == reflect.Float64 {
			vf.SetFloat(reflect.ValueOf(pv).Float())
		}

	case reflect.Bool:
		if b, ok := pv.(bool); ok {
			vf.SetBool(b)
		}
	}
	return nil
}

func isValidDest(i interface{}) bool {
	rv := reflect.ValueOf(i)
	return rv.Kind() == reflect.Ptr && !rv.IsNil() && rv.Elem().Kind() == reflect.Struct
}

// reflect.Kind に対する、interface{} 値を代入する操作をする関数を生成する
func generateAssignmentFunc(vf reflect.Kind) func(f reflect.Value, v interface{}) error {

}

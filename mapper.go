package cypherx

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/ettle/strcase"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Mapper struct {
	assignFuncs []assignmentFunc
	propNames   []string
}

type assignmentFunc func(f reflect.Value, v interface{}) error

var _scannerIt = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func (m *Mapper) Scan(dest interface{}, props map[string]interface{}) error {
	rt := reflect.TypeOf(dest)

	if !isValidPtr(dest) || rt.Elem().Kind() != reflect.Struct {
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

		assign := m.assignFuncs[i]
		if err := assign(field, pv); err != nil {
			return err
		}
	}

	return nil
}

func (m *Mapper) ScanAll(dest interface{}, result neo4j.Result) error {
	rt := reflect.TypeOf(dest)

	if !isValidPtr(rt) ||
		rt.Elem().Kind() != reflect.Slice ||
		rt.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be valid pointer to a slice of struct\n")
	}

	structType := rt.Elem().Elem()
	if err := m.analyzeStruct(structType); err != nil {
		return err
	}

	rv := reflect.ValueOf(dest)

	var record *neo4j.Record
	for result.NextRecord(&record) {
		v := record.GetByIndex(0)
		node, ok := v.(neo4j.Node)
		if !ok {
			return NotNodeTypeErr
		}

		// st はポインターになる
		st := reflect.New(structType)
		props := node.Props
		for i, p := range m.propNames {
			pv, ok := props[p]
			if !ok {
				continue
			}

			field := st.Elem().Field(i)

			assign := m.assignFuncs[i]
			if err := assign(field, pv); err != nil {
				return err
			}
		}
		rv.Elem().Set(reflect.Append(rv.Elem(), st.Elem()))
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
	funcs := make([]assignmentFunc, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		tag := tf.Tag.Get("neo4j")
		propName := strings.Split(tag, ",")[0]
		if propName == "" {
			propName = strcase.ToSnake(tf.Name)
		}
		names = append(names, propName)
		f, err := generateAssignmentFunc(tf.Type)
		if err != nil {
			return err
		}
		funcs = append(funcs, f)
	}

	m.assignFuncs = funcs
	m.propNames = names

	return nil
}

func isValidPtr(i interface{}) bool {
	rv := reflect.ValueOf(i)
	return rv.Kind() == reflect.Ptr && !rv.IsNil()
}

// TODO: ポインタータイプの対応
// reflect.Kind に対する、interface{} 値を代入する操作をする関数を生成する
func generateAssignmentFunc(rt reflect.Type) (assignmentFunc, error) {
	vfPtr := reflect.PtrTo(rt)
	if vfPtr.Implements(_scannerIt) {
		return func(f reflect.Value, v interface{}) error {
			ptr := f.Addr()
			ptr.MethodByName("Scan").Call([]reflect.Value{reflect.ValueOf(v)})

			return nil
		}, nil
	}

	switch rt.Kind() {
	case reflect.String:
		return func(f reflect.Value, v interface{}) error {
			if s, ok := v.(string); ok {
				f.SetString(s)
				return nil
			}
			return nil
		}, nil

	// TODO: その他のint型の対応
	case reflect.Int, reflect.Int64:
		return func(f reflect.Value, v interface{}) error {
			// neo4j の整数の型は int64 のみ
			if v, ok := v.(int64); ok {
				f.SetInt(v)
			}
			return nil
		}, nil

	// TODO: float32 の対応の必要か？
	case reflect.Float64:
		return func(f reflect.Value, v interface{}) error {
			if v, ok := v.(float64); ok {
				f.SetFloat(v)
			}
			return nil
		}, nil

	case reflect.Bool:
		return func(f reflect.Value, v interface{}) error {
			if v, ok := v.(bool); ok {
				f.SetBool(v)
			}
			return nil
		}, nil
	}

	return nil, fmt.Errorf("cannot generate assignment func for %s type", rt.String())
}

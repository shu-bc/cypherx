package cypherx

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/ettle/strcase"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type mapper struct {
	assignFuncs []assignmentFunc
	propNames   []string
}

var publicFieldReg = regexp.MustCompile(`^[A-Z]+`)

type assignmentFunc func(f reflect.Value, v interface{}) error

var _scannerIt = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

func (m *mapper) scan(dest interface{}, props map[string]interface{}) error {
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

func (m *mapper) scanValues(dest interface{}, result neo4j.Result) error {
	rt := reflect.TypeOf(dest)
	if rt.Elem().Kind() != reflect.Slice ||
		rt.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid type %s, expect slice struct kind\n", rt.Elem().String())
	}

	structType := rt.Elem().Elem()
	if err := m.analyzeStruct(structType); err != nil {
		return err
	}

	slicePtr := reflect.ValueOf(dest)
	var record *neo4j.Record
	for result.NextRecord(&record) {
		newStruct := reflect.New(structType)
		for i, v := range record.Values {
			field := newStruct.Elem().Field(i)
			assignFunc := m.assignFuncs[i]
			if err := assignFunc(field, v); err != nil {
				return err
			}
		}
		slicePtr.Elem().Set(reflect.Append(slicePtr.Elem(), newStruct.Elem()))
	}

	return nil
}

func (m *mapper) scanAll(dest interface{}, result neo4j.Result) error {
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

func (m *mapper) analyzeStruct(t reflect.Type) error {
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
		// ignore private field
		if !publicFieldReg.MatchString(tf.Name) {
			continue
		}

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

			return fmt.Errorf("unexpected value type %T, expect string\n", v)
		}, nil

	// TODO: その他のint型の対応
	case reflect.Int, reflect.Int64:
		return func(f reflect.Value, v interface{}) error {
			// neo4j の整数の型は int64 のみ
			if v, ok := v.(int64); ok {
				f.SetInt(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect int, int64\n", v)
		}, nil

	// TODO: float32 の対応の必要か？
	case reflect.Float64:
		return func(f reflect.Value, v interface{}) error {
			if v, ok := v.(float64); ok {
				f.SetFloat(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect float64\n", v)
		}, nil

	case reflect.Bool:
		return func(f reflect.Value, v interface{}) error {
			if v, ok := v.(bool); ok {
				f.SetBool(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect bool\n", v)
		}, nil
	}

	return nil, fmt.Errorf("cannot generate assignment func for %s type", rt.String())
}

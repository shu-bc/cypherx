package cypherx

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/ettle/strcase"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

var publicFieldPattern = regexp.MustCompile(`^[A-Z]+`)
var _scannerIt = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
var UnsettableValueErr = errors.New("unsettable reflect value\n")

type assignmentFunc func(f reflect.Value, v interface{}) error

type typeMapper struct {
	mapping map[reflect.Type]*mapper
	mu      sync.Mutex
}

var typeMapperCache = typeMapper{
	mapping: map[reflect.Type]*mapper{},
}

type mapper struct {
	assignFuncs []assignmentFunc
	propNames   []string
}

func (m *mapper) scanProps(stPtr reflect.Value, props map[string]interface{}) error {
	for i, name := range m.propNames {
		pv, ok := props[name]
		if !ok {
			continue
		}

		field := stPtr.Elem().Field(i)
		assignFuc := m.assignFuncs[i]
		if err := assignFuc(field, pv); err != nil {
			return err
		}
	}

	return nil
}

func (m *mapper) scanValues(structPtr reflect.Value, record *neo4j.Record) error {
	for i, v := range record.Values {
		field := structPtr.Elem().Field(i)
		assignFunc := m.assignFuncs[i]
		if err := assignFunc(field, v); err != nil {
			return err
		}
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

	if c, ok := typeMapperCache.mapping[t]; ok {
		m.assignFuncs = c.assignFuncs
		m.propNames = c.propNames
		return nil
	}

	if t.NumField() == 0 {
		return fmt.Errorf("struct must have > 1 field\n")
	}

	names := make([]string, 0, t.NumField())
	funcs := make([]assignmentFunc, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)
		// ignore private field
		if !publicFieldPattern.MatchString(tf.Name) {
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

	typeMapperCache.mu.Lock()
	typeMapperCache.mapping[t] = m
	typeMapperCache.mu.Unlock()

	return nil
}

func isValidPtr(i interface{}) bool {
	rv := reflect.ValueOf(i)
	return rv.Kind() == reflect.Ptr && !rv.IsNil()
}

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
			if !f.CanSet() {
				return UnsettableValueErr
			}

			if s, ok := v.(string); ok {
				f.SetString(s)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect string\n", v)
		}, nil

	// TODO: その他のint型の対応
	case reflect.Int, reflect.Int64:
		return func(f reflect.Value, v interface{}) error {
			if !f.CanSet() {
				return UnsettableValueErr
			}

			// neo4j の整数の型は int64 のみ
			if v, ok := v.(int64); ok {
				f.SetInt(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect int, int64", v)
		}, nil

	// TODO: float32 の対応の必要か？
	case reflect.Float64:
		return func(f reflect.Value, v interface{}) error {
			if !f.CanSet() {
				return UnsettableValueErr
			}

			if v, ok := v.(float64); ok {
				f.SetFloat(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect float64\n", v)
		}, nil

	case reflect.Bool:
		return func(f reflect.Value, v interface{}) error {
			if !f.CanSet() {
				return UnsettableValueErr
			}

			if v, ok := v.(bool); ok {
				f.SetBool(v)
				return nil
			}

			return fmt.Errorf("unexpected value type %T, expect bool\n", v)
		}, nil

	case reflect.Struct:
		return func(f reflect.Value, v interface{}) error {
			if !f.CanSet() {
				return UnsettableValueErr
			}

			node, ok := v.(neo4j.Node)
			if !ok {
				return NotNodeTypeErr
			}

			m, ok := typeMapperCache.mapping[f.Type()]
			if !ok {
				m = &mapper{}
				if err := m.analyzeStruct(f.Type()); err != nil {
					return fmt.Errorf("failed to analyze struct type %s: %w", f.Type().String(), err)
				}

				typeMapperCache.mu.Lock()
				typeMapperCache.mapping[f.Type()] = m
				typeMapperCache.mu.Unlock()
			}

			if err := m.scanProps(f.Addr(), node.Props); err != nil {
				return fmt.Errorf("failed to scan props to struct: %w", err)
			}
			return nil
		}, nil
	}

	return nil, fmt.Errorf("cannot generate assignment func for %s type", rt.String())
}

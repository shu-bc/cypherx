package cypherx

import (
	"reflect"
	"strings"

	"github.com/ettle/strcase"
)

type Mapper struct {
}

func (m Mapper) Map(dest interface{}, props map[string]interface{}) error {
	rt := reflect.TypeOf(dest).Elem()

	for i := 0; i < rt.NumField(); i++ {
		tf := rt.Field(i)
		tag := tf.Tag.Get("neo4j")
		propName := strings.Split(tag, ",")[0]
		if propName == "" {
			propName = strcase.ToSnake(tf.Name)
		}

		pv, ok := props[propName]
		if !ok {
			continue
		}

		vf := reflect.ValueOf(dest).Elem().Field(i)
		if err := m.fillField(vf, pv); err != nil {
			return err
		}
	}

	return nil
}

func (m Mapper) fillField(vf reflect.Value, pv interface{}) error {
	switch vf.Kind() {
	case reflect.String:
		if s, ok := pv.(string); ok {
			vf.SetString(s)
		}

	case reflect.Int:
		if reflect.ValueOf(pv).Kind() == reflect.Int {
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

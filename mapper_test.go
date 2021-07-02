package cypherx_test

import (
	"testing"

	"github.com/shu-bc/cypherx"
	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name string `neo4j:"name"`
	Age  int
}

func TestMap(t *testing.T) {
	m := cypherx.Mapper{}
	props := map[string]interface{}{
		"name": "test",
		"age":  3,
	}
	p := &Person{}
	err := m.Map(p, props)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, &Person{Name: "test", Age: 3}, p)
}

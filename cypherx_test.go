package cypherx

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name     string `neo4j:"name"`
	Age      int
	Salary   float64
	Deleted  bool `neo4j:"del"`
	SocialID sql.NullString
}

func TestGetNode(t *testing.T) {
	db := &DB{}
	if err := db.Connect("bolt://neo4j", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := db.ExecQuery("match (p:Person{name: 'peter'}) delete p", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ExecQuery("merge (:Person{name: 'peter', age: 30,  salary: 1000.1, social_id: '123abc'})", nil); err != nil {
		t.Fatal(err)
	}
	p := &Person{}
	err := db.GetNode(p, "match (p:Person{name: 'peter'}) return p", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "peter", p.Name)
	assert.Equal(t, 30, p.Age)
	assert.Equal(t, 1000.1, p.Salary)
	assert.True(t, p.SocialID.Valid)
	assert.Equal(t, "123abc", p.SocialID.String)
}

func TestGetNodes(t *testing.T) {
	db := &DB{}
	if err := db.Connect("bolt://neo4j", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := db.ExecQuery("match (p:Person{name: 'GetNodesTest'}) delete p", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ExecQuery("merge (:Person{name: 'GetNodesTest', age: 30,  salary: 1000.1, social_id: '123abc'})", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ExecQuery("merge (:Person{name: 'GetNodesTest', age: 25,  salary: 1200.1, social_id: 'abc123'})", nil); err != nil {
		t.Fatal(err)
	}

	ps := &[]Person{}
	err := db.GetNodes(ps, "match (p:Person) where p.name = 'GetNodesTest' return p", nil)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(ps)
}

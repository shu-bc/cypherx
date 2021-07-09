package cypherx

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type Person struct {
	Name     string `neo4j:"name"`
	Age      int
	Salary   float64
	Deleted  bool `neo4j:"del"`
	SocialID sql.NullString
	p        string // for private field test
}

func TestGetNode(t *testing.T) {
	db := &DB{}
	if err := db.Connect("bolt://neo4j", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := db.ExecQuery("match (p:Person{name: 'peter'}) delete p", nil, WithTxTimeout(2*time.Second)); err != nil {
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
	assert.Equal(t, "", p.p)
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

func TestGetMultiValueRecords(t *testing.T) {
	db := &DB{}
	if err := db.Connect("bolt://neo4j", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := db.ExecQuery("match (p:Person{name: 'GetValues'}) delete p", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ExecQuery("merge (:Person{name: 'GetValues', age: 30,  salary: 1000.1, social_id: '123abc'})", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ExecQuery("merge (:Person{name: 'GetValues', age: 25,  salary: 1200.1, social_id: 'abc123'})", nil); err != nil {
		t.Fatal(err)
	}

	resStruct := []struct {
		A int
		B string
	}{}

	if err := db.GetMultiValueRecords(&resStruct, "match (p:Person{name :'GetValues'}) return p.age, p.name", nil); err != nil {
		t.Fatal(err)
	}

	fmt.Println(resStruct)
}

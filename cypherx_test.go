package cypherx_test

import (
	"testing"

	"github.com/shu-bc/cypherx"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	t.Skip()
	db, err := cypherx.NewDB("bolt://neo4j", "", "")
	if err != nil {
		t.Fatal(err)
	}

	db.SendQuery("create (:Person{name: 'peter'})", map[string]interface{}{})
	db.SendQuery("match (p:Person) return p", map[string]interface{}{})
	db.SendQuery("match (n) delete n", map[string]interface{}{})
}

func TestGet(t *testing.T) {
	db, err := cypherx.NewDB("bolt://neo4j", "", "")
	if err != nil {
		t.Fatal(err)
	}

	db.SendQuery("match (p:Person{name: 'peter'}) delete p", map[string]interface{}{})
	db.SendQuery("merge (:Person{name: 'peter', age: 30,  salary: 1000.1, social_id: '123abc'})", map[string]interface{}{})
	p := &Person{}
	err = db.Get(p, "match (p:Person{name: 'peter'}) return p", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "peter", p.Name)
	assert.Equal(t, 30, p.Age)
	assert.Equal(t, 1000.1, p.Salary)
	assert.True(t, p.SocialID.Valid)
	assert.Equal(t, "123abc", p.SocialID.String)
}

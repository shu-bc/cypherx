package cypherx_test

import (
	"testing"

	"github.com/shu-bc/cypherx"
)

func TestDB(t *testing.T) {
	db, err := cypherx.NewDB("bolt://neo4j", "", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.SendQuery("create (:Person{name: 'peter'})", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SendQuery("match (p:Person) return p", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.SendQuery("match (n) delete n", map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
}

package cypherx

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type DB struct {
	driver neo4j.Driver
}

func NewDB(host, user, pass string) (*DB, error) {
	d, err := neo4j.NewDriver(
		host,
		neo4j.BasicAuth(
			user,
			pass,
			"",
		),
	)

	if err != nil {
		return nil, err
	}

	return &DB{
		driver: d,
	}, nil
}

func (db *DB) SendQuery(cypher string, params map[string]interface{}) {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params)
	if err != nil {
		panic(err)
	}

	var record *neo4j.Record
	for res.NextRecord(&record) {
		fmt.Println(record)
	}
}

func (db *DB) Get(dest interface{}, cypher string, params map[string]interface{}) {

}

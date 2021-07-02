package cypherx

import (
	"fmt"
	"reflect"

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

	_, err := session.Run(cypher, params)
	if err != nil {
		panic(err)
	}
}

func (db *DB) Get(
	dest interface{},
	cypher string,
	params map[string]interface{},
) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("dest must be a non-null pointer\n")
	}

	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params)
	if err != nil {
		return fmt.Errorf("cypher execution failure: %w\n", err)
	}

	var record *neo4j.Record
	record, err = res.Single()
	if err != nil {
		return fmt.Errorf("result must contain only one record: %w\n", err)
	}

	node, ok := record.GetByIndex(0).(neo4j.Node)
	if !ok {
		return fmt.Errorf("type neo4j.Node assertion failure, unexpected result type\n")
	}

	m := Mapper{}
	err = m.Map(dest, node.Props)
	if err != nil {
		return fmt.Errorf("fail to assign props to dest: %w\n", err)
	}

	return nil
}

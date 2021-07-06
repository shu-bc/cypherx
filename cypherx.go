package cypherx

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type DB struct {
	driver neo4j.Driver
}

var (
	NotNodeTypeErr = errors.New("type neo4j.Node assertion failure, unexpected result type\n")
)

func NewDB(driver neo4j.Driver) *DB {
	return &DB{driver: driver}
}

func (db *DB) Connect(host, user, pass string) error {
	d, err := neo4j.NewDriver(
		host,
		neo4j.BasicAuth(
			user,
			pass,
			"",
		),
	)

	if err != nil {
		return err
	}

	db.driver = d

	return nil
}

func (db *DB) ExecQuery(cypher string, params map[string]interface{}) error {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.Run(cypher, params)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) Query(cypher string, params map[string]interface{}) (interface{}, error) {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params)
	if err != nil {
		return nil, err
	}

	result := [][]interface{}{}
	var record *neo4j.Record
	for res.NextRecord(&record) {
		result = append(result, record.Values)
	}
	return result, nil
}

func (db *DB) GetNode(
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
		return NotNodeTypeErr
	}

	m := mapper{}
	err = m.scan(dest, node.Props)
	if err != nil {
		return fmt.Errorf("fail to assign props to dest: %w\n", err)
	}

	return nil
}

func (db *DB) GetNodes(
	dest interface{},
	cypher string,
	params map[string]interface{},
) error {
	rv := reflect.ValueOf(dest)
	if rv.IsNil() {
		return fmt.Errorf("dest must be a non-null pointer\n")
	}

	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params)
	if err != nil {
		return fmt.Errorf("cypher execution failure: %w\n", err)
	}

	m := mapper{}
	if err := m.scanAll(dest, res); err != nil {
		return fmt.Errorf("fail to map all nodes to dest: %w\n", err)
	}

	return nil
}

package cypherx

import (
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type DB struct {
	driver neo4j.Driver
}

type Configurer func(*neo4j.TransactionConfig)

var (
	NotNodeTypeErr = errors.New("type neo4j.Node assertion failure, unexpected result type\n")
	NotValidPtrErr = errors.New("dest must be a non-null pointer\n")

	WithTxMetadata = neo4j.WithTxMetadata
	WithTxTimeout  = neo4j.WithTxTimeout
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

func (db *DB) ExecQuery(cypher string, params map[string]interface{}, configurers ...Configurer) error {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.Run(cypher, params)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RawResult(cypher string, params map[string]interface{}, configurers ...Configurer) (interface{}, error) {
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

//GetMultiValueRecords fetch records from neo4j db and assign values of each record to a struct
func (db *DB) GetMultiValueRecords(dest interface{}, cypher string, params map[string]interface{}, configurers ...Configurer) error {
	if !isValidPtr(dest) {
		return NotValidPtrErr
	}

	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params)
	if err != nil {
		return fmt.Errorf("cypher execution failure: %w\n", err)
	}

	m := mapper{}
	if err := m.scanValues(dest, res); err != nil {
		return fmt.Errorf("fail to map values to dest: %w\n", err)
	}

	return nil
}

func (db *DB) GetNode(
	dest interface{},
	cypher string,
	params map[string]interface{},
	configurers ...Configurer,
) error {
	if !isValidPtr(dest) {
		return NotValidPtrErr
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
	configurers ...Configurer,
) error {
	if !isValidPtr(dest) {
		return NotValidPtrErr
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

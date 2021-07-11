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

type configurer = func(*neo4j.TransactionConfig)

var (
	notNodeTypeErr     = errors.New("type neo4j.Node assertion failure, unexpected result type")
	notValidPtrErr     = errors.New("destination variable must be a non-null pointer")
	unsettableValueErr = errors.New("unsettable reflect value")

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

func (db *DB) ExecQuery(cypher string,
	params map[string]interface{},
	configurers ...configurer,
) error {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	_, err := session.Run(cypher, params, configurers...)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) RawResult(cypher string,
	params map[string]interface{},
	configurers ...configurer,
) (interface{}, error) {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params, configurers...)
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
//dest must be *[]struct{} type
func (db *DB) GetMultiValueRecords(
	dest interface{},
	cypher string,
	params map[string]interface{},
	configurers ...configurer,
) error {
	if !isValidPtr(dest) {
		return notValidPtrErr
	}

	rt := reflect.TypeOf(dest)
	if rt.Elem().Kind() != reflect.Slice ||
		rt.Elem().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid destination variable type %s, expect slice struct kind", rt.Elem().String())
	}

	m := &mapper{}
	structType := rt.Elem().Elem()
	if err := m.analyzeStruct(structType); err != nil {
		return err
	}

	resChan := make(chan *neo4j.Record)
	errChan := make(chan error)

	go db.fetchRecords(resChan, errChan, cypher, params, configurers...)

	slicePtr := reflect.ValueOf(dest)
	for res := range resChan {
		st := reflect.New(structType)
		if err := m.scanValues(st, res); err != nil {
			return fmt.Errorf("scan values failed: %w", err)
		}

		slicePtr.Elem().Set(reflect.Append(slicePtr.Elem(), st.Elem()))
	}

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (db *DB) GetNode(
	dest interface{},
	cypher string,
	params map[string]interface{},
	configurers ...configurer,
) error {
	if !isValidPtr(dest) {
		return notValidPtrErr
	}

	rt := reflect.TypeOf(dest)
	if rt.Elem().Kind() != reflect.Struct {
		return errors.New("dest must be a pointer to struct")
	}

	m := &mapper{}
	structType := rt.Elem()

	if err := m.analyzeStruct(structType); err != nil {
		return fmt.Errorf("failed to analyze struct type: %w", err)
	}

	resChan := make(chan *neo4j.Record)
	errChan := make(chan error)

	go db.fetchRecord(resChan, errChan, cypher, params, configurers...)

	structPtr := reflect.ValueOf(dest)
	select {
	case res := <-resChan:
		node, ok := res.GetByIndex(0).(neo4j.Node)
		if !ok {
			return notNodeTypeErr
		}

		if err := m.scanProps(structPtr, node.Props); err != nil {
			return fmt.Errorf("scan props failed: %w", err)
		}
	case err := <-errChan:
		return err
	}

	return nil
}

func (db *DB) fetchRecords(
	resChan chan *neo4j.Record,
	errChan chan error,
	cypher string,
	params map[string]interface{},
	configurers ...configurer,
) {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params, configurers...)
	if err != nil {
		close(resChan)
		errChan <- err
		return
	}

	var record *neo4j.Record
	for res.NextRecord(&record) {
		resChan <- record
	}
	close(resChan)
	close(errChan)
}

func (db *DB) fetchRecord(
	resChan chan *neo4j.Record,
	errChan chan error,
	cypher string,
	params map[string]interface{},
	configurers ...configurer,
) {
	session := db.driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	res, err := session.Run(cypher, params, configurers...)
	if err != nil {
		close(resChan)
		errChan <- err
		return
	}

	record, err := res.Single()
	if err != nil {
		close(resChan)
		errChan <- err
		return
	}

	resChan <- record

	close(resChan)
	close(errChan)
}

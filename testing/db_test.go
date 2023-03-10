package testing

/* ==============================================
Copyright (c) Eensymachines
Developed by 		: kneerunjun@gmail.com
Developed on 		: JAN'23
This shall test the database operations
to test database we can run a sample database inside a container
In the same directory find a docker-compose file to start the mongo db
============================================== */
import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/eensymachines.in/useracc"
	"github.com/eensymachines.in/useracc/nosql"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	// for testing we use an independent session
	Conn *mgo.Session
)

const (
	DATABASE_NAME  = "useraccs"
	COLL_NAME      = "users"
	ARCHVCOLL_NAME = "archive_users"
	DBHOST         = "localhost:47017"
)

// SetupMongoConn :  dials a mongo connection to testing host and can send out to the testing functions
// forceSeed : flag will force seed the database all over again
// set this t false if you need only to get the database connection
// will flush only the default collection before seeding it again
func SetupMongoConn(forceSeed bool) (nosql.IDBConn, func(), error) {
	db, close, err := nosql.InitDialConn(&nosql.DBInitConfig{
		Host:      DBHOST,
		DB:        DATABASE_NAME,
		UName:     "",
		Passwd:    "",
		Coll:      COLL_NAME,
		ArchvColl: ARCHVCOLL_NAME,
		DBTyp:     reflect.TypeOf(&nosql.MongoDB{}),
	})
	if err != nil {
		return nil, nil, err
	}
	if forceSeed {
		byt, err := os.ReadFile("./seed.json")
		if err != nil {
			return nil, nil, fmt.Errorf("seedDB/ReadAll %s", err)
		}
		toInsert := []useracc.UserAccount{}
		if err := json.Unmarshal(byt, &toInsert); err != nil {
			return nil, nil, fmt.Errorf("seedDB/ReadAll %s", err)
		}
		// HACK: having to convert to a pointer to get the aggregated session
		// this cannot be but is ok only since its the testing environment
		coll := db.(*nosql.MongoDB).Session.DB(DATABASE_NAME).C(COLL_NAME)
		coll.RemoveAll(bson.M{}) // flushing the db
		for _, item := range toInsert {
			ua, err := useracc.NewUsrAccount(item.Eml, item.Ttle, item.Phn, item.Addr.Pincode)
			if err == nil { // no error creating new account
				err = coll.Insert(ua)
				if err != nil { // error inserting item into collection
					continue
				}
			}
		}
	}
	return db, close, nil
}

// TestRemoveDoc : aimed at testing RemoveFromColl(coll string, id string, softDel bool, affected *int) error
func TestRemoveDoc(t *testing.T) {
	t.Log("now testing for removing an account")
	db, close, err := SetupMongoConn(true)
	if err != nil { // if the mongo connection was not setup, aborting the entire test
		t.Error(err)
		return
	}
	defer close()
	// getting the sample data for test
	// getting ids of 10 documents that can be used for deletion
	var result interface{}
	err = db.(nosql.IQryable).GetSampleFromColl(COLL_NAME, 10, &result)
	if err != nil { // fails to get even the samples, abort test
		t.Error(err)
		return
	}
	ids := result.(map[string][]bson.ObjectId)
	// here we go ahead to test deletion of an item from database
	// Soft deletion
	for _, id := range ids["sample"] {
		var count int
		err := db.(nosql.IQryable).RemoveFromColl(COLL_NAME, id.Hex(), true, &count)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error when deleting doc with id %s", err))
	}
}

func TestGetSampleFromColl(t *testing.T) {
	// ==============
	// dial connecting the database
	// ==============
	db, close, err := SetupMongoConn(true)
	defer close()
	assert.Nil(t, err, "failed to connect to db")
	assert.NotNil(t, db, "nil db pointer")
	// ================
	var result interface{}
	err = db.(nosql.IQryable).GetSampleFromColl(COLL_NAME, uint32(10), &result)
	assert.Nil(t, err, "unexpected error when GetSampleFromColl")
	assert.NotNil(t, result, "nil result for GetSampleFromColl")
	byt, err := json.Marshal(result)
	assert.Nil(t, err, "unexpected error when json.Marshal")
	t.Log(string(byt))
	// NOTE: this no longer is needed since the interface now expects uint32 as the size datatype
	// ================
	// GetSampleFromColl with invalid size
	// err = db.(nosql.IQryable).GetSampleFromColl(COLL_NAME, -10, &result)
	// // with invalid size, you dont get any error
	// // the sample set would be empty
	// assert.Nil(t, err, "Unexpected error when getting sample with invalid size")
	// ids := result.(map[string][]bson.ObjectId)
	// sample := ids["sample"]
	// assert.Equal(t, 0, len(sample), "Unexpected non-empty sample size")
	/* ==========
	- Negative tests
	*/
	err = db.(nosql.IQryable).GetSampleFromColl("", 10, &result)
	assert.NotNil(t, err, "Error nil unexpected")
}

func TestGetGetOneFromColl(t *testing.T) {
	// ==============
	// dial connecting the database
	// ==============
	db, close, err := SetupMongoConn(true)
	defer close()
	assert.Nil(t, err, "failed to connect to db")
	assert.NotNil(t, db, "nil db pointer")
	// Now getting one sample from database so as to test
	var result interface{}
	db.(nosql.IQryable).GetSampleFromColl(COLL_NAME, 1, &result)
	ids := result.(map[string][]bson.ObjectId)
	sample := ids["sample"]
	assert.Equal(t, 1, len(sample), "Unexpected number of items in the sample")
	// ===============
	ua := useracc.UserAccount{}
	var uaMap map[string]interface{}
	err = db.(nosql.IQryable).GetOneFromColl(COLL_NAME, func() bson.M { return bson.M{"_id": sample[0]} }, &uaMap)
	t.Log(uaMap)
	byt, _ := json.Marshal(uaMap)
	t.Log(string(byt))
	json.Unmarshal(byt, &ua)
	assert.Nil(t, err, "Unexpected error when GetOneFromColl")
	t.Log(ua)
	// Negative tests
	// ===============
	err = db.(nosql.IQryable).GetOneFromColl("", func() bson.M { return bson.M{"_id": sample[0]} }, &uaMap)
	assert.NotNil(t, err, "Nil Error unexpected")

	err = db.(nosql.IQryable).GetOneFromColl(COLL_NAME, nil, &uaMap)
	assert.NotNil(t, err, "Nil Error unexpected")
}

// TestFilterFromColl : filteration of documents on custom filter to get all ids
func TestFilterFromColl(t *testing.T) {
	db, close, err := SetupMongoConn(true)
	defer close()
	assert.Nil(t, err, "failed to connect to db")
	assert.NotNil(t, db, "nil db pointer")
	// ===========
	var result map[string][]bson.ObjectId
	err = db.(nosql.IQryable).FilterFromColl(COLL_NAME, func() bson.M {
		return bson.M{"email": "cdobrowski0@pcworld.com"}
	}, &result)
	assert.Nil(t, err, "Unexpected error when FilterFromColl")
	t.Log(result)
	// ==========
	err = db.(nosql.IQryable).FilterFromColl("", func() bson.M {
		return bson.M{"email": "cdobrowski0@pcworld.com"}
	}, &result)
	assert.NotNil(t, err, "unexpected not nil err")
	// ==============
	err = db.(nosql.IQryable).FilterFromColl(COLL_NAME, nil, &result)
	assert.NotNil(t, err, "unexpected not nil err")

	err = db.(nosql.IQryable).FilterFromColl(COLL_NAME, func() bson.M {
		return bson.M{"email": "someone@unknown.com"}
	}, &result)
	assert.NotNil(t, err, "Unexpected error when FilterFromColl")
	t.Log(result)
}

func TestEditOneFromColl(t *testing.T) {
	// NOTE: setting up the test data and database connections
	db, close, err := SetupMongoConn(true)
	defer close()
	assert.Nil(t, err, "failed to connect to db")
	assert.NotNil(t, db, "nil db pointer")
	// ===========
	// Simple +ve test for count of documents updated
	var count int
	newTitle := "NewTitle"
	err = db.(nosql.IQryable).EditOneFromColl(COLL_NAME, func() bson.M {
		return bson.M{
			"email": "cdobrowski0@pcworld.com",
		} // selection filter
	}, func() bson.M {
		return bson.M{
			"$set": bson.M{"title": newTitle},
		} // setting action
	}, &count)
	assert.Nil(t, err, "Unexpected error when FilterFromColl")
	assert.Equal(t, count, 1, "Unexpected number of documents updated")

	// TEST: to know if the document is +vely updated
	var browski useracc.UserAccount
	db.(*nosql.MongoDB).DB("").C(COLL_NAME).Find(bson.M{
		"email": "cdobrowski0@pcworld.com",
	}).One(&browski)
	assert.Equal(t, newTitle, browski.Ttle, "Update query hasnt really updated the document")

	// TEST: when colleciton is invalid
	err = db.(nosql.IQryable).EditOneFromColl("", func() bson.M {
		return bson.M{
			"email": "cdobrowski0@pcworld.com",
		} // selection filter
	}, func() bson.M {
		return bson.M{
			"$set": bson.M{"title": newTitle},
		} // setting action
	}, &count)
	assert.NotNil(t, err, "Unexpected error not nil when invalid collection")

	// TEST: invalid filter
	err = db.(nosql.IQryable).EditOneFromColl("", nil, func() bson.M {
		return bson.M{
			"$set": bson.M{"title": newTitle},
		} // setting action
	}, &count)
	assert.NotNil(t, err, "Unexpected error not nil when invalid filter")

	// TEST: when patch is invalid
	err = db.(nosql.IQryable).EditOneFromColl("", func() bson.M {
		return bson.M{
			"email": "cdobrowski0@pcworld.com",
		} // selection filter
	}, nil, &count)
	assert.NotNil(t, err, "Unexpected error not nil when invalid patch")

	// TEST: when filter finds 0 documents to update
	err = db.(nosql.IQryable).EditOneFromColl(COLL_NAME, func() bson.M {
		return bson.M{
			// NOTE: this email filter will fetch 0 documents
			"email": "cdobrowski0@theoffice.com",
		} // selection filter
	}, func() bson.M {
		return bson.M{
			"$set": bson.M{"title": newTitle},
		}
	}, &count)
	assert.Nil(t, err, "Unexpected error when FilterFromColl")
	assert.Equal(t, count, 0, "Unexpected number of documents updated")
}

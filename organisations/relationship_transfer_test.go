package organisations

import (
	"fmt"
	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/jmcvetta/neoism"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	testRelationshipLeftToRight     = "TEST_RELATIONSHIP_1"
	testRelationshipRightToLeft     = "TEST_RELATIONSHIP_2"
	relationShipTransferContentUUID = "d3dbe29e-5f6f-456f-a245-9c4d70846e11"
	transferOrg1UUID                = "10c547d2-6383-41e1-9430-2f543321587f"
	transferOrg2UUID                = "3977bc1c-1026-45f0-b7db-d91ff25770fb"
	fsTransferOrg1Identifier        = "org identifier 1"
	fsTransferOrg2Identifier        = "org identifier 2"
)

var transferOrg1 = organisation{
	UUID: transferOrg1UUID,
	Type: Organisation,
	AlternativeIdentifiers: alternativeIdentifiers{
		FactsetIdentifier: fsTransferOrg1Identifier,
		UUIDS:             []string{transferOrg1UUID},
		TME:               []string{},
	},
	ProperName: "Org Proper Name 1",
}

var transferOrg2 = organisation{
	UUID: transferOrg2UUID,
	Type: Organisation,
	AlternativeIdentifiers: alternativeIdentifiers{
		FactsetIdentifier: fsTransferOrg2Identifier,
		UUIDS:             []string{transferOrg2UUID},
		TME:               []string{},
	},
	ProperName: "Org Proper Name 2",
}

var transferUUIDsToClean = []string{relationShipTransferContentUUID, transferOrg1UUID, transferOrg2UUID}

func TestGetNodeRelationshipNames(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert, transferUUIDsToClean)
	cypherDriver := getCypherDriver(db)
	defer cleanRelationshipDB(db, t, assert, transferUUIDsToClean)

	addMentionsQuery := &neoism.CypherQuery{
		Statement: `MATCH (c:Thing{uuid:{uuid}})
			    CREATE (co:Content{uuid:{cuuid}})
			    CREATE (co)-[:` + testRelationshipLeftToRight + `{someProperty:"someValue"}]->(c)
			    CREATE (co)<-[:` + testRelationshipRightToLeft + `]-(c)`,
		Parameters: map[string]interface{}{
			"cuuid": relationShipTransferContentUUID,
			"uuid":  transferOrg1UUID,
		},
	}

	assert.NoError(cypherDriver.Write(transferOrg1, "TEST_TRANS_ID"))
	assert.NoError(cypherDriver.conn.CypherBatch([]*neoism.CypherQuery{addMentionsQuery}))

	relationshipsFromNodeWithUUID, relationshipsToNodeWithUUID, err := getNodeRelationshipNames(cypherDriver.conn, transferOrg1UUID)

	assert.NoError(err)
	assert.True(len(relationshipsFromNodeWithUUID) >= 1, "Expected -> relationship length differs from actual length")
	assert.True(len(relationshipsToNodeWithUUID) >= 1, "Expected <- relationship length differs from actual length")

	assert.True(contains(relationshipsFromNodeWithUUID, testRelationshipRightToLeft))
	assert.True(contains(relationshipsToNodeWithUUID, testRelationshipLeftToRight))
}

func TestTransferRelationships(t *testing.T) {
	assert := assert.New(t)
	db := getDatabaseConnectionAndCheckClean(t, assert, transferUUIDsToClean)
	cypherDriver := getCypherDriver(db)
	defer cleanRelationshipDB(db, t, assert, transferUUIDsToClean)

	addMentionsQuery := &neoism.CypherQuery{
		Statement: `MATCH (c:Thing{uuid:{uuid}})
			    CREATE (co:Content{uuid:{cuuid}})
			    CREATE (co)-[:` + testRelationshipLeftToRight + `{someProperty:"someValue"}]->(c)
			    CREATE (co)<-[:` + testRelationshipRightToLeft + `]-(c)`,
		Parameters: map[string]interface{}{
			"cuuid": relationShipTransferContentUUID,
			"uuid":  transferOrg1UUID,
		},
	}
	assert.NoError(cypherDriver.Write(transferOrg1, "TEST_TRANS_ID"))
	assert.NoError(cypherDriver.conn.CypherBatch([]*neoism.CypherQuery{addMentionsQuery}))

	//write new node and test that it doesn't yet have the relationships
	assert.NoError(cypherDriver.Write(transferOrg2, "TEST_TRANS_ID"))
	relationshipsFromNewNode, relationshipsToNewNode, err := getNodeRelationshipNames(cypherDriver.conn, transferOrg2UUID)
	assert.NoError(err)
	assert.False(contains(relationshipsFromNewNode, testRelationshipRightToLeft))
	assert.False(contains(relationshipsToNewNode, testRelationshipRightToLeft))

	//transfer relationships from the one above to the on other uuid
	transferQuery, err := CreateTransferRelationshipsQueries(cypherDriver.conn, transferOrg2UUID, transferOrg1UUID)
	assert.NoError(err)
	assert.NoError(cypherDriver.conn.CypherBatch(transferQuery))

	//verify that the relationships has been transferred
	relationshipsFromOldNode, relationshipsToOldNode, err := getNodeRelationshipNames(cypherDriver.conn, transferOrg1UUID)
	assert.NoError(err)
	relationshipsFromNewNode, relationshipsToNewNode, err = getNodeRelationshipNames(cypherDriver.conn, transferOrg2UUID)
	assert.NoError(err)

	//no relationships for the old node
	assert.Equal(0, len(relationshipsFromOldNode))
	assert.Equal(0, len(relationshipsToOldNode))

	//new relationships for the new node
	assert.True(contains(relationshipsFromNewNode, testRelationshipRightToLeft))
	assert.True(contains(relationshipsToNewNode, testRelationshipLeftToRight))

	//verify that properties has been transferred
	type property []struct {
		Value string `json:"r.someProperty"`
	}

	transferredProperty := property{}
	readRelationshipPropertyQuery := &neoism.CypherQuery{
		Statement: `match (co:Content{uuid:{cuuid}})-[r:` + testRelationshipLeftToRight + `]->(c:Thing{uuid:{uuid}})
 				return r.someProperty`,
		Parameters: map[string]interface{}{
			"cuuid": relationShipTransferContentUUID,
			"uuid":  transferOrg2UUID,
		},
		Result: &transferredProperty,
	}
	assert.NoError(cypherDriver.conn.CypherBatch([]*neoism.CypherQuery{readRelationshipPropertyQuery}))
	assert.Equal(1, len(transferredProperty))
	assert.Equal("someValue", transferredProperty[0].Value)
}

func cleanRelationshipDB(db neoutils.CypherRunner, t *testing.T, assert *assert.Assertions, uuidsToClean []string) {
	cleanDB(db, t, assert, uuidsToClean)

	qs := []*neoism.CypherQuery{
		{
			Statement: fmt.Sprintf("MATCH (c:Content {uuid: '%v'})-[rel]-(o) DELETE c, rel ", relationShipTransferContentUUID),
		},
		{
			Statement: fmt.Sprintf("MATCH (c:Content {uuid: '%v'}) DELETE c ", relationShipTransferContentUUID),
		},
	}

	err := db.CypherBatch(qs)
	assert.NoError(err)
}

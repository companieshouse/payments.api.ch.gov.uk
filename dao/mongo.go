package dao

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/companieshouse/chs.go/log"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const deadline = 5 * time.Second

var client *mongo.Client

const paymentStatus = "data.status"

// MongoService is an implementation of the Service interface using MongoDB as the backend driver.
type MongoService struct {
	db             MongoDatabaseInterface
	CollectionName string
}

// MongoDatabaseInterface is an interface that describes the mongodb driver
type MongoDatabaseInterface interface {
	Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection
}

func getMongoDatabase(mongoDBURL, databaseName string) MongoDatabaseInterface {
	return getMongoClient(mongoDBURL).Database(databaseName)
}

func getMongoClient(mongoDBURL string) *mongo.Client {
	if client != nil {
		return client
	}

	ctx := context.Background()

	clientOptions := options.Client().ApplyURI(mongoDBURL)
	client, err := mongo.Connect(ctx, clientOptions)

	// Assume the caller of this func cannot handle the case where there is no database connection
	// so the service must crash here as it cannot continue.
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// Check we can connect to the mongodb instance. Failure here should result in a crash.
	pingContext, cancel := context.WithDeadline(ctx, time.Now().Add(deadline))

	err = client.Ping(pingContext, nil)
	if err != nil {
		log.Error(errors.New("ping to mongodb timed out. please check the connection to mongodb and that it is running"))
		os.Exit(1)
	}

	defer cancel()

	log.Info("connected to mongodb successfully")

	return client
}

// CreatePaymentResource writes a new payment resource to the DB
func (m *MongoService) CreatePaymentResource(paymentResource *models.PaymentResourceDB) error {
	collection := m.db.Collection(m.CollectionName)

	_, err := collection.InsertOne(context.Background(), paymentResource)

	return err
}

// GetPaymentResource gets a payment resource from the DB
// If payment not found in DB, return nil
func (m *MongoService) GetPaymentResource(id string) (*models.PaymentResourceDB, error) {

	var resource models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	dbResource := collection.FindOne(context.Background(), bson.M{"_id": id})

	err := dbResource.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Info("no payment resource found for id " + id)
			return nil, nil
		}
		return nil, err
	}

	err = dbResource.Decode(&resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// PatchPaymentResource patches a payment resource from the DB
func (m *MongoService) PatchPaymentResource(id string, paymentUpdate *models.PaymentResourceDB) error {
	collection := m.db.Collection(m.CollectionName)

	patchUpdate := make(bson.M)

	// Patch only these fields
	if paymentUpdate.Data.PaymentMethod != "" {
		patchUpdate["data.payment_method"] = paymentUpdate.Data.PaymentMethod
	}
	if paymentUpdate.Data.Status != "" {
		patchUpdate[paymentStatus] = paymentUpdate.Data.Status
	}
	if !paymentUpdate.Data.CompletedAt.IsZero() {
		patchUpdate["data.completed_at"] = paymentUpdate.Data.CompletedAt
	}
	if paymentUpdate.ExternalPaymentStatusURI != "" {
		patchUpdate["external_payment_status_url"] = paymentUpdate.ExternalPaymentStatusURI
	}
	if paymentUpdate.ExternalPaymentStatusID != "" {
		patchUpdate["external_payment_status_id"] = paymentUpdate.ExternalPaymentStatusID
	}
	if paymentUpdate.ExternalPaymentTransactionID != "" {
		patchUpdate["external_payment_transaction_id"] = paymentUpdate.ExternalPaymentTransactionID
	}
	if paymentUpdate.Refunds != nil {
		patchUpdate["refunds"] = paymentUpdate.Refunds
	}
	if paymentUpdate.Data.ProviderID != "" {
		patchUpdate["data.provider_id"] = paymentUpdate.Data.ProviderID
	}
	if len(paymentUpdate.BulkRefund) != 0 {
		patchUpdate["bulk_refunds"] = paymentUpdate.BulkRefund
	}

	updateCall := bson.M{"$set": patchUpdate}

	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, updateCall)

	return err
}

// GetPaymentResourceByExternalPaymentStatusID retrieves a payment resource
// associated with the externalPaymentStatusID provided
func (m *MongoService) GetPaymentResourceByExternalPaymentStatusID(externalPaymentStatusID string) (*models.PaymentResourceDB, error) {
	var resource models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	document := collection.FindOne(context.Background(), bson.M{"external_payment_status_id": externalPaymentStatusID})

	err := document.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Info(fmt.Sprintf("no payment resource found for external status id: [%s]", externalPaymentStatusID))
			return nil, nil
		}
		return nil, err
	}

	err = document.Decode(&resource)
	if err != nil {
		return nil, err
	}

	return &resource, nil
}

// CreateBulkRefund creates or adds to the array of bulk refunds on a payment resource
func (m *MongoService) CreateBulkRefund(externalPaymentStatusID string, bulkRefund models.BulkRefundDB) error {
	collection := m.db.Collection(m.CollectionName)

	filter := bson.M{"external_payment_status_id": externalPaymentStatusID}
	pushQuery := bson.M{"$push": bson.M{"bulk_refunds": bulkRefund}}

	update, err := collection.UpdateOne(context.Background(), filter, pushQuery)
	if err != nil {
		return fmt.Errorf("error updating bulk refund for payment with external status id [%s]: %w", externalPaymentStatusID, err)
	}
	if update.ModifiedCount == 0 {
		return fmt.Errorf("payment with external status id [%s] not found", externalPaymentStatusID)
	}

	return nil
}

// GetPaymentsWithRefundStatus retrieves a list of all payments in the DB with a status of
// refund-pending
func (m *MongoService) GetPaymentsWithRefundStatus() ([]models.PaymentResourceDB, error) {
	var payments []models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	statusFilter := bson.M{"bulk_refunds.status": "refund-pending"}
	bulkRefundsFilter := bson.M{"bulk_refunds.0": bson.M{"$exists": true}}

	paymentDBResources, err := collection.Find(context.Background(), bson.M{"$and": bson.A{statusFilter, bulkRefundsFilter}})
	if err != nil {
		return nil, err
	}

	err = paymentDBResources.All(context.Background(), &payments)
	if err != nil {
		return nil, err
	}

	return payments, nil
}

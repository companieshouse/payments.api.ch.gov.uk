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

const (
	paymentStatus                = "data.status"
	refundStatus                 = "refunds.status"
	bulkRefundStatus             = "bulk_refunds.status"
	dataProviderID               = "data.provider_id"
	externalPaymentTransactionID = "external_payment_transaction_id"
)

// MongoService is an implementation of the Service interface using MongoDB as the backend driver.
type MongoService struct {
	db              MongoDatabaseInterface
	CollectionName  string
	RefundBatchSize int
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
		patchUpdate[externalPaymentTransactionID] = paymentUpdate.ExternalPaymentTransactionID
	}
	if paymentUpdate.Refunds != nil {
		patchUpdate["refunds"] = paymentUpdate.Refunds
	}
	if paymentUpdate.Data.ProviderID != "" {
		patchUpdate[dataProviderID] = paymentUpdate.Data.ProviderID
	}
	if len(paymentUpdate.BulkRefund) != 0 {
		patchUpdate["bulk_refunds"] = paymentUpdate.BulkRefund
	}
	if paymentUpdate.Data.Links.Refunds != "" {
		patchUpdate["data.links.refunds"] = paymentUpdate.Data.Links.Refunds
	}

	updateCall := bson.M{"$set": patchUpdate}

	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": id}, updateCall)

	return err
}

// GetPaymentResourceByProviderID retrieves a payment resource
// associated with the supplied Provider ID
func (m *MongoService) GetPaymentResourceByProviderID(providerID string) (*models.PaymentResourceDB, error) {
	var resource models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	document := collection.FindOne(context.Background(), bson.M{dataProviderID: providerID})

	err := document.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Info(fmt.Sprintf("no payment resource found for provider id: [%s]", providerID))
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

// GetPaymentResourceByExternalPaymentTransactionID retrieves a payment resource
// associated with the externalPaymentTransactionID provided
func (m *MongoService) GetPaymentResourceByExternalPaymentTransactionID(id string) (*models.PaymentResourceDB, error) {
	var resource models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	document := collection.FindOne(context.Background(), bson.M{externalPaymentTransactionID: id})

	err := document.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			log.Info(fmt.Sprintf("no payment resource found for external_payment_transaction_id: [%s]", id))
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

// CreateBulkRefundByProviderID creates or adds to the array of bulk refunds on a payment resource
// The query only updates those payments in the DB with the specified Provider ID
// which do not have an existing bulk refund with the status of refund-pending
// or refund-requested
func (m *MongoService) CreateBulkRefundByProviderID(bulkRefunds map[string]models.BulkRefundDB) error {
	return m.CreateBulkRefund(bulkRefunds, dataProviderID)
}

// CreateBulkRefundByExternalPaymentTransactionID creates or adds to the array of bulk refunds on a payment resource
// The query only updates those payments in the DB with the specified External Payment
// Transaction ID which do not have an existing bulk refund with the status of refund-pending
// or refund-requested
func (m *MongoService) CreateBulkRefundByExternalPaymentTransactionID(bulkRefunds map[string]models.BulkRefundDB) error {
	return m.CreateBulkRefund(bulkRefunds, externalPaymentTransactionID)
}

// CreateBulkRefund creates or adds to the array of bulk refunds on a payment resource
// The query only updates those payments in the DB with the specified external payment
// status ID, filtered on the specified query string, that do not have an existing
// bulk refund with the status of refund-pending or refund-requested
func (m *MongoService) CreateBulkRefund(bulkRefunds map[string]models.BulkRefundDB, idQuery string) error {
	collection := m.db.Collection(m.CollectionName)

	var operations []mongo.WriteModel

	for orderID, bulkRefund := range bulkRefunds {
		IDFilter := bson.M{idQuery: orderID}
		pendingFilter := bson.M{bulkRefundStatus: "refund-pending"}
		requestedFilter := bson.M{bulkRefundStatus: "refund-requested"}
		statusFilter := bson.M{"$nor": bson.A{pendingFilter, requestedFilter}}

		filter := bson.M{"$and": bson.A{IDFilter, statusFilter}}
		pushQuery := bson.M{"$push": bson.M{"bulk_refunds": bulkRefund}}

		operations = append(operations,
			mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(pushQuery))
		log.Info(fmt.Sprintf("Added operation for payment [%s]", orderID))
	}

	log.Info(fmt.Sprintf("Running BulkWrite operation for refund file for refunds on field [%s]", idQuery))
	update, err := collection.BulkWrite(context.Background(), operations)

	if err != nil {
		return fmt.Errorf("error bulk updating on mongo for bulk refund file [%s]: %w", bulkRefunds, err)
	}
	if update.ModifiedCount == 0 {
		log.Error(fmt.Errorf("no payments updated in bulk update for bulk refunds [%s]", bulkRefunds))
	}

	return nil
}

// GetPaymentsWithRefundStatus retrieves a list of all payments in the DB with a bulk refund status of
// refund-pending
func (m *MongoService) GetPaymentsWithRefundStatus() ([]models.PaymentResourceDB, error) {
	var payments []models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	statusFilter := bson.M{bulkRefundStatus: "refund-pending"}

	paymentDBResources, err := collection.Find(context.Background(), statusFilter)
	if err != nil {
		return nil, err
	}

	err = paymentDBResources.All(context.Background(), &payments)
	if err != nil {
		return nil, err
	}

	return payments, nil
}

// GetPaymentsWithRefundPendingStatus retrieves a list of payments in the DB with a status of refund-requested
func (m *MongoService) GetPaymentsWithRefundPendingStatus() ([]models.PaymentResourceDB, error) {
	var payments []models.PaymentResourceDB

	collection := m.db.Collection(m.CollectionName)
	statusFilter := bson.M{refundStatus: "refund-requested"}

	filterOptions := options.Find()
	filterOptions.SetSort(bson.M{"_id": 1})
	filterOptions.SetSkip(0)
	filterOptions.SetLimit(int64(m.RefundBatchSize))

	paymentDBResources, err := collection.Find(context.Background(), statusFilter, filterOptions)
	if err != nil {
		return nil, err
	}

	err = paymentDBResources.All(context.Background(), &payments)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	paymentDBResources.Close(context.Background())

	return payments, nil
}

// PatchPaymentsWithRefundPendingStatus updates payment refunds status to refund-requested and insert a new refund_at
func (m *MongoService) PatchPaymentsWithRefundPendingStatus(id string, isPaid bool, paymentUpdate *models.PaymentResourceDB) (models.PaymentResourceDB, error) {
	collection := m.db.Collection(m.CollectionName)
	refunds := paymentUpdate.Refunds[0]
	attempts := refunds.Attempts + 1

	refundsFilter := options.ArrayFilters{Filters: bson.A{bson.M{"x.refund_id": refunds.RefundId}}}
	upsert := true

	after := options.After

	opts := options.FindOneAndUpdateOptions{
		ArrayFilters:   &refundsFilter,
		ReturnDocument: &after,
		Upsert:         &upsert,
	}

	var patchUpdate, updatedDoc bson.M

	if isPaid {
		patchUpdate = bson.M{
			"$set": bson.M{
				"refunds.$[x].status":      "refund-success",
				"refunds.$[x].refunded_at": time.Now(),
				"refunds.$[x].attempts":    attempts,
			},
		}
	} else {
		patchUpdate = bson.M{
			"$set": bson.M{
				"refunds.$[x].attempts": attempts,
			},
		}
	}

	updatedPayment := models.PaymentResourceDB{}
	result := collection.FindOneAndUpdate(context.Background(), bson.M{"_id": id}, patchUpdate, &opts)

	if result.Err() != nil {
		return updatedPayment, result.Err()
	}

	err := result.Decode(&updatedDoc)

	if err != nil {
		return updatedPayment, err
	}

	bsonBytes, _ := bson.Marshal(updatedDoc)
	bson.Unmarshal(bsonBytes, &updatedPayment)

	return updatedPayment, nil
}

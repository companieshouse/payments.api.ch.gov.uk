package dao

import (
	"testing"
	"time"

	"github.com/companieshouse/payments.api.ch.gov.uk/config"
	"github.com/companieshouse/payments.api.ch.gov.uk/models"

	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func NewGetMongoDatabase(mongoDBURL, databaseName string) MongoDatabaseInterface {
	return getMongoClient(mongoDBURL).Database(databaseName)
}

func ptr(t time.Time) *time.Time {
	return &t
}

func setDriverUp() (MongoService,
	mtest.CommandError,
	*mtest.Options,
	models.PaymentResourceDB,
	map[string]models.BulkRefundDB,
	[]models.RefundResourceDB) {

	client = &mongo.Client{}
	cfg, _ := config.Get()
	dataBase := NewGetMongoDatabase("mongoDBURL", "databaseName")

	mongoService := MongoService{
		db:             dataBase,
		CollectionName: cfg.Collection,
	}

	commandError := mtest.CommandError{
		Code:    1,
		Message: "Message",
		Name:    "Name",
		Labels:  []string{"label1"},
	}
	paymentResourceDataDB := models.PaymentResourceDataDB{}
	refundResourceDB := []models.RefundResourceDB{}
	bulkRefundDB := []models.BulkRefundDB{}
	paymentResource := models.PaymentResourceDB{
		ID:                           "ID",
		RedirectURI:                  "RedirectURI",
		State:                        "State",
		ExternalPaymentStatusURI:     " ExternalPaymentStatusURI",
		ExternalPaymentStatusID:      " ExternalPaymentStatusID",
		ExternalPaymentTransactionID: " ExternalPaymentTransactionID",
		Data:                         paymentResourceDataDB,
		Refunds:                      append(refundResourceDB, models.RefundResourceDB{}),
		BulkRefund:                   append(bulkRefundDB, models.BulkRefundDB{}),
	}

	bulkRefund := models.BulkRefundDB{
		Status:            "status",
		UploadedFilename:  "uploaded_filename",
		UploadedAt:        "uploaded_at",
		UploadedBy:        "uploaded_by",
		Amount:            "amount",
		RefundID:          "refund_id",
		ProcessedAt:       "processed_at",
		ExternalRefundURL: "external_refund_url",
	}
	bulkRefunds := map[string]models.BulkRefundDB{"id": bulkRefund}

	p := ptr(time.Now())

	refundResource := models.RefundResourceDB{
		RefundId:          "refund_id",
		RefundedAt:        p,
		CreatedAt:         "created_at",
		Amount:            80,
		Status:            "status",
		Attempts:          8,
		ExternalRefundUrl: "external_refund_url",
		RefundReference:   "refund_reference",
	}
	refundResourceDb := []models.RefundResourceDB{}

	refundResourceDb = append(refundResourceDb, refundResource)

	opts := mtest.NewOptions().DatabaseName("databaseName").ClientType(mtest.Mock)

	return mongoService, commandError, opts, paymentResource, bulkRefunds, refundResourceDb
}

func TestUnitCreatePaymentResourceDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("CreatePaymentResource runs successfully", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		mongoService.db = mt.DB

		err := mongoService.CreatePaymentResource(&paymentResource)

		assert.Nil(t, err)
	})

	mt.Run("CreatePaymentResource runs with error", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB

		err := mongoService.CreatePaymentResource(&paymentResource)

		assert.NotNil(t, err)
	})
}

func TestUnitGetPaymentResourceDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentResource successfully", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", paymentResource.ID},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
		}))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResource("ID")
		assert.NotNil(t, paymentResource)
		assert.Nil(t, err)
		assert.Equal(t, paymentResource.ID, "ID")
		assert.Equal(t, paymentResource.State, "State")
		assert.Equal(t, paymentResource.RedirectURI, "RedirectURI")
	})

	mt.Run("GetPaymentResource with error findone", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResource("ID")

		assert.NotNil(t, err)
		assert.Nil(t, paymentResource)
	})

	mt.Run("GetPaymentResource with decoding error", func(mt *mtest.T) {
		response := mtest.CreateCursorResponse(
			1,
			"models.PaymentResourceDB",
			mtest.FirstBatch, bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", models.RefundResourceDB{}},
				{"bulk_refunds", models.BulkRefundDB{}},
			})

		mt.AddMockResponses(response)
		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResource("ID")
		assert.Nil(t, paymentResource)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "error decoding key refunds: cannot decode document into []models.RefundResourceDB")

	})

}

func TestUnitPatchPaymentResourceDriver(t *testing.T) {
	t.Parallel()

	mongoService, _, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("PatchPaymentResource runs successfully", func(mt *mtest.T) {
		mongoService.db = mt.DB
		err := mongoService.PatchPaymentResource("ID", &paymentResource)

		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no responses remaining")
	})
}

func TestUnitGetPaymentResourceByProviderIDDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentResourceByProviderID successfully", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", paymentResource.ID},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
		}))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByProviderID("providerID")
		assert.NotNil(t, paymentResource)
		assert.Nil(t, err)
		assert.Equal(t, paymentResource.ID, "ID")
		assert.Equal(t, paymentResource.State, "State")
		assert.Equal(t, paymentResource.RedirectURI, "RedirectURI")
	})

	mt.Run("GetPaymentResource with error findone", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByProviderID("providerID")

		assert.NotNil(t, err)
		assert.Nil(t, paymentResource)
	})

	mt.Run("GetPaymentResource with decoding error", func(mt *mtest.T) {
		response := mtest.CreateCursorResponse(
			1,
			"models.PaymentResourceDB",
			mtest.FirstBatch, bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", models.RefundResourceDB{}},
				{"bulk_refunds", models.BulkRefundDB{}},
			})

		mt.AddMockResponses(response)
		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByProviderID("providerID")
		assert.Nil(t, paymentResource)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "error decoding key refunds: cannot decode document into []models.RefundResourceDB")
	})
}

func TestUnitGetPaymentResourceByExternalPaymentTransactionIDDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentResourceByExternalPaymentTransactionID successfully", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", paymentResource.ID},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
		}))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByExternalPaymentTransactionID("id")
		assert.NotNil(t, paymentResource)
		assert.Nil(t, err)
		assert.Equal(t, paymentResource.ID, "ID")
		assert.Equal(t, paymentResource.State, "State")
		assert.Equal(t, paymentResource.RedirectURI, "RedirectURI")
	})

	mt.Run("GetPaymentResource with error findone", func(mt *mtest.T) {

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByExternalPaymentTransactionID("id")

		assert.NotNil(t, err)
		assert.Nil(t, paymentResource)
	})

	mt.Run("GetPaymentResource with decoding error", func(mt *mtest.T) {
		response := mtest.CreateCursorResponse(
			1,
			"models.PaymentResourceDB",
			mtest.FirstBatch, bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", models.RefundResourceDB{}},
				{"bulk_refunds", models.BulkRefundDB{}},
			})

		mt.AddMockResponses(response)
		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentResourceByExternalPaymentTransactionID("id")
		assert.Nil(t, paymentResource)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "error decoding key refunds: cannot decode document into []models.RefundResourceDB")
	})
}

func TestUnitCreateBulkRefundByExternalPaymentTransactionIDDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, _, bulkRefunds, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("CreateBulkRefundByExternalPaymentTransactionID runs with error on empty slice", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB
		err := mongoService.CreateBulkRefund(map[string]models.BulkRefundDB{}, externalPaymentTransactionID)

		assert.Equal(t, err.Error(), "error bulk updating on mongo for bulk refund file [map[]]: must provide at least one element in input slice")
	})

	mt.Run("CreateBulkRefundByExternalPaymentTransactionID runs with slice successfully", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"nModified", 0},
		})

		mongoService.db = mt.DB
		err := mongoService.CreateBulkRefund(bulkRefunds, externalPaymentTransactionID)

		assert.Nil(t, err)
	})

}

func TestUnitGetIncompleteGovPayPaymentsDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, _, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	cfg, _ := config.Get()

	mt.Run("GetIncompleteGovPayPayments runs successfully", func(mt *mtest.T) {
		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", primitive.NewObjectID()},
		})

		stopCursors := mtest.CreateCursorResponse(0, "models.PaymentResourceDB", mtest.NextBatch)
		mt.AddMockResponses(first, stopCursors)

		mongoService.db = mt.DB
		payments, err := mongoService.GetIncompleteGovPayPayments(cfg)

		assert.Nil(t, err)
		assert.NotNil(t, payments)
	})

	mt.Run("GetIncompleteGovPayPayments runs with error on find", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB
		_, err := mongoService.GetIncompleteGovPayPayments(cfg)

		assert.Equal(t, err.Error(), "(Name) Message")
	})

	mt.Run("GetIncompleteGovPayPayments runs with error on unmarshal cursor", func(mt *mtest.T) {
		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", primitive.NewObjectID()},
		})

		mt.AddMockResponses(first)

		mongoService.db = mt.DB
		payments, err := mongoService.GetIncompleteGovPayPayments(cfg)

		assert.Nil(t, payments)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no responses remaining")
	})
}

func TestUnitGetPaymentsWithRefundStatusDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentsWithRefundStatus runs successfully", func(mt *mtest.T) {
		id1 := primitive.NewObjectID()
		id2 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", id1},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})
		second := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.NextBatch, bson.D{
			{"_id", id2},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})

		stopCursors := mtest.CreateCursorResponse(0, "models.PaymentResourceDB", mtest.NextBatch)
		mt.AddMockResponses(first, second, stopCursors)

		mongoService.db = mt.DB
		payments, err := mongoService.GetPaymentsWithRefundStatus()

		assert.Nil(t, err)
		assert.NotNil(t, payments)
	})

	mt.Run("GetPractitionerResources runs with error on find", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB
		_, err := mongoService.GetPaymentsWithRefundStatus()

		assert.Equal(t, err.Error(), "(Name) Message")
	})

	mt.Run("GetPractitionerResources runs with error on unmarshal cursor", func(mt *mtest.T) {
		id1 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", id1},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})

		mt.AddMockResponses(first)

		mongoService.db = mt.DB
		payments, err := mongoService.GetPaymentsWithRefundStatus()

		assert.Nil(t, payments)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no responses remaining")
	})
}

func TestUnitGetPaymentsWithRefundPendingStatusDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, _ := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentsWithRefundPendingStatus runs successfully", func(mt *mtest.T) {
		id1 := primitive.NewObjectID()
		id2 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", id1},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})
		second := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.NextBatch, bson.D{
			{"_id", id2},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})

		stopCursors := mtest.CreateCursorResponse(0, "models.PaymentResourceDB", mtest.NextBatch)
		mt.AddMockResponses(first, second, stopCursors)

		mongoService.db = mt.DB
		payments, err := mongoService.GetPaymentsWithRefundPendingStatus()

		assert.Nil(t, err)
		assert.NotNil(t, payments)
	})

	mt.Run("GetPaymentsWithRefundPendingStatus runs with error on find", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB
		_, err := mongoService.GetPaymentsWithRefundStatus()

		assert.Equal(t, err.Error(), "(Name) Message")
	})

	mt.Run("GetPaymentsWithRefundPendingStatus runs with error on unmarshal cursor", func(mt *mtest.T) {
		id1 := primitive.NewObjectID()

		first := mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", id1},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
		})

		mt.AddMockResponses(first)

		mongoService.db = mt.DB
		payments, err := mongoService.GetPaymentsWithRefundPendingStatus()

		assert.Nil(t, payments)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "no responses remaining")
	})
}

func TestUnitGetPaymentRefundsDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, refunds := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("GetPaymentRefunds successfully", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "models.PaymentResourceDB", mtest.FirstBatch, bson.D{
			{"_id", paymentResource.ID},
			{"redirect_uri", paymentResource.RedirectURI},
			{"state", paymentResource.State},
			{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
			{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
			{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
			{"data", paymentResource.Data},
			{"refunds", refunds},
			{"bulk_refunds", []models.BulkRefundDB{}},
		}))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentRefunds("id")
		assert.NotNil(t, paymentResource)
		assert.Nil(t, err)
	})

	mt.Run("GetPaymentRefunds with error findone", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentRefunds("id")

		assert.NotNil(t, err)
		assert.Nil(t, paymentResource)
		assert.Equal(t, err.Error(), "(Name) Message")
	})

	mt.Run("GetPaymentRefunds with decoding error", func(mt *mtest.T) {
		response := mtest.CreateCursorResponse(
			1,
			"models.PaymentResourceDB",
			mtest.FirstBatch, bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", models.RefundResourceDB{}},
				{"bulk_refunds", models.BulkRefundDB{}},
			})

		mt.AddMockResponses(response)
		mongoService.db = mt.DB

		paymentResource, err := mongoService.GetPaymentRefunds("id")
		assert.Nil(t, paymentResource)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "error decoding key refunds: cannot decode document into []models.RefundResourceDB")
	})
}

func TestUnitPatchPaymentsWithRefundPendingStatusDriver(t *testing.T) {
	t.Parallel()

	mongoService, commandError, opts, paymentResource, _, refunds := setDriverUp()

	mt := mtest.New(t, opts)
	defer mt.Close()

	mt.Run("PatchPaymentsWithRefundPendingStatus runs successfully", func(mt *mtest.T) {
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"value", bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", refunds},
				{"bulk_refunds", []models.BulkRefundDB{}},
			}},
		})

		mongoService.db = mt.DB
		paymentRefunds, err := mongoService.PatchRefundSuccessStatus("id", true, &paymentResource)

		assert.Nil(t, err)
		assert.NotNil(t, paymentRefunds)
	})

	mt.Run("PatchPaymentsWithRefundPendingStatus runs with error on FindOneAndUpdate", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateCommandErrorResponse(commandError))

		mongoService.db = mt.DB
		paymentRefunds, err := mongoService.PatchRefundSuccessStatus("id", true, &paymentResource)

		assert.NotNil(t, err)
		assert.NotNil(t, paymentRefunds)
		assert.Equal(t, err.Error(), "(Name) Message")
	})

	mt.Run("GetPaymentRefunds with decoding error", func(mt *mtest.T) {
		response := mtest.CreateCursorResponse(
			1,
			"models.PaymentResourceDB",
			mtest.FirstBatch, bson.D{
				{"_id", paymentResource.ID},
				{"redirect_uri", paymentResource.RedirectURI},
				{"state", paymentResource.State},
				{"external_payment_status_url", paymentResource.ExternalPaymentStatusURI},
				{"external_payment_status_id", paymentResource.ExternalPaymentStatusID},
				{"external_payment_transaction_id", paymentResource.ExternalPaymentTransactionID},
				{"data", paymentResource.Data},
				{"refunds", models.RefundResourceDB{}},
				{"bulk_refunds", models.BulkRefundDB{}},
			})

		mt.AddMockResponses(response)
		mongoService.db = mt.DB

		paymentRefunds, err := mongoService.PatchRefundSuccessStatus("id", true, &paymentResource)
		assert.NotNil(t, paymentRefunds)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "mongo: no documents in result")
	})
}

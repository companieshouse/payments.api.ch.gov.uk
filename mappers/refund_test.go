package mappers

import (
	"testing"

	"github.com/companieshouse/payments.api.ch.gov.uk/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestUnitMapGovPayToRefundResponse(t *testing.T) {

	Convey("Maps successfully to refund response", t, func() {
		govPayResponse := models.CreateRefundGovPayResponse{
			RefundId:    "123",
			CreatedDate: "321",
			Amount:      400,
			Links: models.GovPayRefundLinks{
				Self: models.Self{
					HREF:   "asd",
					Method: "dsa",
				},
				Payment: models.Payment{
					HREF:   "qwe",
					Method: "ewq",
				},
			},
			Status: "success",
		}

		refundResponse := MapGovPayToRefundResponse(govPayResponse)

		So(refundResponse.RefundId, ShouldEqual, govPayResponse.RefundId)
		So(refundResponse.Amount, ShouldEqual, govPayResponse.Amount)
		So(refundResponse.Status, ShouldEqual, govPayResponse.Status)
		So(refundResponse.CreatedDateTime, ShouldEqual, govPayResponse.CreatedDate)
	})
}

func TestUnitMapToRefundRest(t *testing.T) {
	Convey("Maps successfully to refund rest", t, func() {

		govPayResponse := models.CreateRefundGovPayResponse{
			RefundId:    "123",
			CreatedDate: "321",
			Amount:      400,
			Links: models.GovPayRefundLinks{
				Self: models.Self{
					HREF:   "asd",
					Method: "dsa",
				},
				Payment: models.Payment{
					HREF:   "qwe",
					Method: "ewq",
				},
			},
			Status: "success",
		}
		refundReference := "ref"

		refundRest := MapToRefundRest(govPayResponse, refundReference)

		So(refundRest.RefundId, ShouldEqual, govPayResponse.RefundId)
		So(refundRest.Amount, ShouldEqual, govPayResponse.Amount)
		So(refundRest.CreatedAt, ShouldEqual, govPayResponse.CreatedDate)
		So(refundRest.Status, ShouldEqual, govPayResponse.Status)
		So(refundRest.ExternalRefundUrl, ShouldEqual, govPayResponse.Links.Self.HREF)
		So(refundRest.RefundReference, ShouldEqual, refundReference)
	})
}

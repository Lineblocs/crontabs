package cmd

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"lineblocs.com/crontabs/mocks"
)

func TestMonthlyBilling(t *testing.T) {
	t.Parallel()
	helpers.InitLogrus("file")

	testWorkspace := &helpers.Workspace{
		Id:        1,
		CreatorId: 101,
		Plan:      "starter",
	}

	testBillingInfo := &helpers.WorkspaceBillingInfo{}

	testUser := &helpers.User{
		Id: 101,
	}

	monthlyCost := 1000

	t.Run("Should finish monthly billing without any issues ", func(t *testing.T) {
		t.Parallel()

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(testWorkspace, nil)
		mockWorkspace.EXPECT().GetWorkspaceBillingInfo(mock.Anything).Return(testBillingInfo, nil)
		mockWorkspace.EXPECT().GetUserFromDB(mock.Anything).Return(testUser, nil)

		mockPayment.EXPECT().ChargeCustomer(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		db, mockSql, err := sqlmock.New()
		assert.NoError(t, err)

		defer db.Close()

		// Mock expectations for GetBillingParams
		mockSql.ExpectQuery("SELECT payment_gateway FROM customizations").
			WillReturnRows(sqlmock.NewRows([]string{"payment_gateway"}).
				AddRow("stripe"))

		mockSql.ExpectQuery("SELECT stripe_private_key FROM api_credentials").
			WillReturnRows(sqlmock.NewRows([]string{"stripe_private_key"}).
				AddRow("test_stripe_key"))

		// Mock expectations for the workspaces query
		mockSql.ExpectQuery("SELECT id, creator_id FROM workspaces").
			WillReturnRows(sqlmock.NewRows([]string{"id", "creator_id"}).
				AddRow(testWorkspace.Id, testWorkspace.CreatorId))

		didCountQuery := "SELECT id, monthly_cost  FROM did_numbers WHERE workspace_id = ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(didCountQuery)).
			WithArgs(testWorkspace.Id).
			WillReturnRows(sqlmock.NewRows([]string{"id", "monthly_cost"}).
				AddRow(testWorkspace.Id, monthlyCost))

		job := NewMonthlyBillingJob(db, mockWorkspace, mockPayment)

		err = job.MonthlyBilling()
		assert.NoError(t, err)

		err = mockSql.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

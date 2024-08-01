package cmd

import (
	"errors"
	"math"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"lineblocs.com/crontabs/mocks"
)

func TestAnnualBilling(t *testing.T) {
	t.Parallel()
	helpers.InitLogrus("file")

	t.Run("Should fail AnnualBilling job due unable to get payment gateway", func(t *testing.T) {

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)

		defer db.Close()

		error := errors.New("failed to get payment_gateway")
		// Mock expectations for GetBillingParams
		mock.ExpectQuery("SELECT payment_gateway FROM customizations").
			WillReturnError(error)

		job := NewAnnualBillingJob(db, mockWorkspace, mockPayment)
		err = job.AnnualBilling()
		assert.Error(t, err)
		assert.Equal(t, error, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should fail AnnualBilling job due unable to get workspace information", func(t *testing.T) {

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)

		defer db.Close()

		// Mock expectations for GetBillingParams
		mock.ExpectQuery("SELECT payment_gateway FROM customizations").
			WillReturnRows(sqlmock.NewRows([]string{"payment_gateway"}).
				AddRow("stripe"))

		mock.ExpectQuery("SELECT stripe_private_key FROM api_credentials").
			WillReturnRows(sqlmock.NewRows([]string{"stripe_private_key"}).
				AddRow("test_stripe_key"))

		error := errors.New("failed to get workspaces")
		// Mock expectations for the workspaces query
		mock.ExpectQuery("SELECT id, creator_id FROM workspaces WHERE plan_term = 'annual'").
			WillReturnError(error)

		job := NewAnnualBillingJob(db, mockWorkspace, mockPayment)
		err = job.AnnualBilling()
		assert.Error(t, err)
		assert.Equal(t, error, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should finish AnnualBilling job without processing due unable to get workspace from db", func(t *testing.T) {

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(nil, errors.New("failed to get workspace"))

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)

		defer db.Close()

		// Mock expectations for GetBillingParams
		mock.ExpectQuery("SELECT payment_gateway FROM customizations").
			WillReturnRows(sqlmock.NewRows([]string{"payment_gateway"}).
				AddRow("stripe"))

		mock.ExpectQuery("SELECT stripe_private_key FROM api_credentials").
			WillReturnRows(sqlmock.NewRows([]string{"stripe_private_key"}).
				AddRow("test_stripe_key"))

		// Mock expectations for the workspaces query
		mock.ExpectQuery("SELECT id, creator_id FROM workspaces WHERE plan_term = 'annual'").
			WillReturnRows(sqlmock.NewRows([]string{"id", "creator_id"}).
				AddRow(1, 101))

		job := NewAnnualBillingJob(db, mockWorkspace, mockPayment)
		err = job.AnnualBilling()
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should finish AnnualBilling job without any issues", func(t *testing.T) {

		workspace := &helpers.Workspace{
			Id:        1,
			CreatorId: 101,
			Plan:      "starter",
		}

		user := &helpers.User{
			Id: 101,
		}

		worksSpaceUsers := 3

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		//Create Starter Workspace
		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(workspace, nil)
		mockWorkspace.EXPECT().GetUserFromDB(mock.Anything).Return(user, nil)

		mockPayment.EXPECT().ChargeCustomer(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		db, mock, err := sqlmock.New()
		assert.NoError(t, err)

		defer db.Close()

		// Mock expectations for GetBillingParams
		mock.ExpectQuery("SELECT payment_gateway FROM customizations").
			WillReturnRows(sqlmock.NewRows([]string{"payment_gateway"}).
				AddRow("stripe"))

		mock.ExpectQuery("SELECT stripe_private_key FROM api_credentials").
			WillReturnRows(sqlmock.NewRows([]string{"stripe_private_key"}).
				AddRow("test_stripe_key"))

		// Mock expectations for the workspaces query
		mock.ExpectQuery("SELECT id, creator_id FROM workspaces WHERE plan_term = 'annual'").
			WillReturnRows(sqlmock.NewRows([]string{"id", "creator_id"}).
				AddRow(1, workspace.CreatorId))

		// Mock expectations for user count query
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) as count FROM  workspaces_users WHERE workspace_id = ?").
			WithArgs(workspace.Id).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(worksSpaceUsers))

		// Mock expectations for the INSERT into users_invoices
		membershipCosts := float64(0) * float64(worksSpaceUsers)
		totalCostsCents := int(math.Ceil(membershipCosts))
		invoiceStatus := "INCOMPLETE"
		regularCostsCents := 0

		// Use regexp.QuoteMeta to escape special characters
		sqlQuery := "INSERT INTO users_invoices (`cents`, `membership_costs`, `status`, `user_id`, `workspace_id`, `created_at`, `updated_at`) VALUES ( ?, ?, ?, ?, ?, ?, ?)"
		escapedQuery := regexp.QuoteMeta(sqlQuery)
		mock.ExpectPrepare(escapedQuery).
			ExpectExec().
			WithArgs(regularCostsCents, totalCostsCents, invoiceStatus, workspace.CreatorId, workspace.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock expectations for the LastInsertId
		sqlInsertId := "UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ?, confirmation_number = ? WHERE id = ?"
		escapedInsertId := regexp.QuoteMeta(sqlInsertId)
		mock.ExpectPrepare(escapedInsertId).
			ExpectExec().
			WithArgs(totalCostsCents, sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		job := NewAnnualBillingJob(db, mockWorkspace, mockPayment)
		err = job.AnnualBilling()
		assert.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

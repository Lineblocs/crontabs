package cmd

import (
	"errors"
	"regexp"
	"testing"
	"time"

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
	moduleId := 1

	t.Run("Should fail monthly billing job due unable to get workspace information", func(t *testing.T) {
		t.Parallel()

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

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
		error := errors.New("failed to get workspaces")
		mockSql.ExpectQuery("SELECT id, creator_id FROM workspaces").
			WillReturnError(error)

		job := NewMonthlyBillingJob(db, mockWorkspace, mockPayment)
		err = job.MonthlyBilling()
		assert.Error(t, err)
		assert.Equal(t, error, err)

		err = mockSql.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should finish monthly billing job without processing due unable to get user from db", func(t *testing.T) {
		t.Parallel()

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(testWorkspace, nil)
		mockWorkspace.EXPECT().GetWorkspaceBillingInfo(mock.Anything).Return(testBillingInfo, nil)
		mockWorkspace.EXPECT().GetUserFromDB(mock.Anything).Return(nil, errors.New("failed to get user"))

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

		job := NewMonthlyBillingJob(db, mockWorkspace, mockPayment)
		err = job.MonthlyBilling()
		assert.NoError(t, err)

		err = mockSql.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should finish monthly billing without any issues for NUMBER_RENTAL", func(t *testing.T) {
		t.Parallel()

		worksSpaceUsers := 3
		membershipCost := 74.97
		totalCostCents := float64(monthlyCost) + membershipCost

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		did := &helpers.DIDNumber{
			MonthlyCost: monthlyCost,
		}

		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(testWorkspace, nil)
		mockWorkspace.EXPECT().GetWorkspaceBillingInfo(mock.Anything).Return(testBillingInfo, nil)
		mockWorkspace.EXPECT().GetUserFromDB(mock.Anything).Return(testUser, nil)
		mockWorkspace.EXPECT().GetDIDFromDB(mock.Anything).Return(did, nil)

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
				AddRow(moduleId, monthlyCost))

		debitQuery := "INSERT INTO users_debits (`source`, `status`, `cents`, `module_id`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?)"
		mockSql.ExpectPrepare(regexp.QuoteMeta(debitQuery)).
			ExpectExec().
			WithArgs("NUMBER_RENTAL", "INCOMPLETE", monthlyCost, moduleId, testUser.Id, testWorkspace.Id, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock expectations for user count query
		userCountQuery := "SELECT COUNT(*) as count FROM  workspaces_users WHERE workspace_id = ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(userCountQuery)).
			WithArgs(testWorkspace.Id).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(worksSpaceUsers))

		// Mock expectations for user_debit
		userDebitQuery := "SELECT id, source, module_id, cents, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(userDebitQuery)).
			WithArgs(testUser.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "source", "module_id", "cents", "created_at"}).
				AddRow(1, "NUMBER_RENTAL", moduleId, monthlyCost, time.Now()))

		// Mock expectations for recordings
		recordingQuery := "SELECT id, size, created_at FROM recordings WHERE user_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(recordingQuery)).
			WithArgs(testUser.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "size", "created_at"}).
				AddRow(1, 0, time.Now()))

		// Mock expectations for faxes
		faxesQuery := "SELECT id, created_at FROM faxes WHERE workspace_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(faxesQuery)).
			WithArgs(testWorkspace.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, time.Now()))

		// Mock expectations for invoices
		invoiceQuery := "INSERT INTO users_invoices (`cents`, `call_costs`, `recording_costs`, `fax_costs`, `membership_costs`, `number_costs`, `status`, `user_id`, `workspace_id`, `created_at`, `updated_at`) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		mockSql.ExpectPrepare(regexp.QuoteMeta(invoiceQuery)).
			ExpectExec().
			WithArgs(float64(1000), float64(0), float64(0), float64(0), membershipCost, float64(monthlyCost), "INCOMPLETE", testUser.Id, testWorkspace.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock expectations for the LastInsertId
		sqlInsertId := "UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ?, confirmation_number = ? WHERE id = ?"
		escapedInsertId := regexp.QuoteMeta(sqlInsertId)
		mockSql.ExpectPrepare(escapedInsertId).
			ExpectExec().
			WithArgs(totalCostCents, sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		job := NewMonthlyBillingJob(db, mockWorkspace, mockPayment)
		err = job.MonthlyBilling()
		assert.NoError(t, err)

		err = mockSql.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Should finish monthly billing without any issues with extra CALL costs", func(t *testing.T) {
		t.Parallel()

		worksSpaceUsers := 3
		membershipCost := 74.97
		extraCallCost := 160
		totalCostCents := membershipCost + float64(extraCallCost)

		mockWorkspace := &mocks.WorkspaceRepository{}
		mockPayment := &mocks.PaymentRepository{}

		call := &helpers.Call{
			DurationNumber: 13000,
		}

		mockWorkspace.EXPECT().GetWorkspaceFromDB(mock.Anything).Return(testWorkspace, nil)
		mockWorkspace.EXPECT().GetWorkspaceBillingInfo(mock.Anything).Return(testBillingInfo, nil)
		mockWorkspace.EXPECT().GetUserFromDB(mock.Anything).Return(testUser, nil)
		mockWorkspace.EXPECT().GetCallFromDB(mock.Anything).Return(call, nil)

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
				AddRow(moduleId, monthlyCost))

		debitQuery := "INSERT INTO users_debits (`source`, `status`, `cents`, `module_id`, `user_id`, `workspace_id`, `created_at`) VALUES ( ?, ?, ?, ?, ?, ?)"
		mockSql.ExpectPrepare(regexp.QuoteMeta(debitQuery)).
			ExpectExec().
			WithArgs("NUMBER_RENTAL", "INCOMPLETE", monthlyCost, moduleId, testUser.Id, testWorkspace.Id, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock expectations for user count query
		userCountQuery := "SELECT COUNT(*) as count FROM  workspaces_users WHERE workspace_id = ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(userCountQuery)).
			WithArgs(testWorkspace.Id).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).
				AddRow(worksSpaceUsers))

		// Mock expectations for user_debit
		userDebitQuery := "SELECT id, source, module_id, cents, created_at FROM users_debits WHERE user_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(userDebitQuery)).
			WithArgs(testUser.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "source", "module_id", "cents", "created_at"}).
				AddRow(1, "CALL", moduleId, monthlyCost, time.Now()))

		// Mock expectations for recordings
		recordingQuery := "SELECT id, size, created_at FROM recordings WHERE user_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(recordingQuery)).
			WithArgs(testUser.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "size", "created_at"}).
				AddRow(1, 0, time.Now()))

		// Mock expectations for faxes
		faxesQuery := "SELECT id, created_at FROM faxes WHERE workspace_id = ? AND created_at BETWEEN ? AND ?"
		mockSql.ExpectQuery(regexp.QuoteMeta(faxesQuery)).
			WithArgs(testWorkspace.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, time.Now()))

		// Mock expectations for invoices
		invoiceQuery := "INSERT INTO users_invoices (`cents`, `call_costs`, `recording_costs`, `fax_costs`, `membership_costs`, `number_costs`, `status`, `user_id`, `workspace_id`, `created_at`, `updated_at`) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		mockSql.ExpectPrepare(regexp.QuoteMeta(invoiceQuery)).
			ExpectExec().
			WithArgs(float64(1000), float64(extraCallCost), float64(0), float64(0), membershipCost, float64(0), "INCOMPLETE", testUser.Id, testWorkspace.Id, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mock expectations for the LastInsertId
		sqlInsertId := "UPDATE users_invoices SET status = 'COMPLETE', source ='CARD', cents_collected = ?, confirmation_number = ? WHERE id = ?"
		escapedInsertId := regexp.QuoteMeta(sqlInsertId)
		mockSql.ExpectPrepare(escapedInsertId).
			ExpectExec().
			WithArgs(totalCostCents, sqlmock.AnyArg(), 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		job := NewMonthlyBillingJob(db, mockWorkspace, mockPayment)
		err = job.MonthlyBilling()
		assert.NoError(t, err)

		err = mockSql.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

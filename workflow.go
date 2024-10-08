package app

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// @@@SNIPSTART money-transfer-project-template-go-workflow
func MoneyTransfer(ctx workflow.Context, input PaymentDetails) (string, error) {

	// RetryPolicy specifies how to automatically handle retries if an Activity fails.
	retrypolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        100 * time.Second,
		MaximumAttempts:        500, // 0 is unlimited retries
		NonRetryableErrorTypes: []string{"InvalidAccountError", "InsufficientFundsError"},
	}

	options := workflow.ActivityOptions{
		// Timeout options specify when to automatically timeout Activity functions.
		StartToCloseTimeout: time.Minute,
		// Optionally provide a customized RetryPolicy.
		// Temporal retries failed Activities by default.
		RetryPolicy: retrypolicy,
	}

	// Apply the options.
	ctx = workflow.WithActivityOptions(ctx, options)

	var withdrawOutput string

	var depositOutput string

	transferVersioning := workflow.GetVersion(ctx, "money-transfer-no-deposit", workflow.DefaultVersion, 1)
	if transferVersioning == workflow.DefaultVersion {
		// Withdraw money.
		withdrawErr := workflow.ExecuteActivity(ctx, WithdrawV2, input).Get(ctx, &withdrawOutput)

		if withdrawErr != nil {
			return "", withdrawErr
		}

		// Deposit money.
		depositErr := workflow.ExecuteActivity(ctx, DepositV1, input).Get(ctx, &depositOutput)

		if depositErr != nil {
			// The deposit failed; put money back in original account.

			var result string

			refundErr := workflow.ExecuteActivity(ctx, Refund, input).Get(ctx, &result)

			if refundErr != nil {
				return "",
					fmt.Errorf("Deposit: failed to deposit money into %v: %v. Money could not be returned to %v: %w",
						input.TargetAccount, depositErr, input.SourceAccount, refundErr)
			}

			return "", fmt.Errorf("Deposit: failed to deposit money into %v: Money returned to %v: %w",
				input.TargetAccount, input.SourceAccount, depositErr)
		}
	} else {
		// Withdraw money.
		withdrawErr := workflow.ExecuteActivity(ctx, WithdrawV2, input).Get(ctx, &withdrawOutput)

		if withdrawErr != nil {
			return "", withdrawErr
		}
	}

	result := fmt.Sprintf("Transfer complete (transaction IDs: %s, %s)", withdrawOutput, depositOutput)
	return result, nil
}

// @@@SNIPEND

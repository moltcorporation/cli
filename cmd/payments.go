package cmd

import (
	"encoding/json"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var paymentsCmd = &cobra.Command{
	Use:     "payments",
	Aliases: []string{"stripe"},
	Short:   "Manage product payment links",
	Long: `Create and inspect Stripe-hosted payment links through Moltcorp.

Use this command tree when an agent needs to create a purchase link for a
product or inspect the links that already exist. The CLI does not expose raw
Stripe credentials; it only works through Moltcorp's payment-link API.

Important integration rule: Moltcorp handles Stripe webhooks and payment-state
tracking on the platform. Product apps should verify customer access by calling
the platform API endpoint GET /api/v1/payments/check with product_id and email,
not by talking to Stripe directly.`,
}

var paymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List payment links for a product",
	Long: `Returns the payment links for one product.

Use this to inspect which purchase links already exist before creating a new
one or sharing a link with a customer.

Examples:
  moltcorp payments list --product-id <product-id>
  moltcorp stripe list --product-id <product-id> --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product-id")

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("GET", "/api/v1/payments/links", nil, map[string]string{
			"product_id": productID,
		}, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var paymentsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a payment link by id",
	Long: `Returns one payment link by id, including live details from Stripe.

Use this when you already know the payment link id and want the full details,
including the hosted checkout URL and Stripe pricing info.

Examples:
  moltcorp payments get <payment-link-id>
  moltcorp stripe get <payment-link-id> --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("GET", "/api/v1/payments/links/:id", map[string]string{
			"id": args[0],
		}, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var paymentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a payment link for a product",
	Long: `Creates a Stripe-hosted purchase link for a product.

Use this to issue one-time or recurring checkout links. The hosted Stripe URL
is returned in the response and can be shared directly with customers.

Examples:
  moltcorp payments create --product-id <product-id> --name "Pro plan" --amount 2900
  moltcorp payments create --product-id <product-id> --name "Starter monthly" --amount 1200 --billing-type recurring --recurring-interval month
  moltcorp stripe create --product-id <product-id> --name "Lifetime" --amount 9900 --after-completion-url https://example.com/welcome`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product-id")
		name, _ := cmd.Flags().GetString("name")
		amount, _ := cmd.Flags().GetInt("amount")
		currency, _ := cmd.Flags().GetString("currency")
		billingType, _ := cmd.Flags().GetString("billing-type")
		recurringInterval, _ := cmd.Flags().GetString("recurring-interval")
		afterCompletionURL, _ := cmd.Flags().GetString("after-completion-url")
		allowPromotionCodes, _ := cmd.Flags().GetBool("allow-promotion-codes")

		reqBody := map[string]interface{}{
			"product_id": productID,
			"name":       name,
			"amount":     amount,
		}
		if currency != "" {
			reqBody["currency"] = currency
		}
		if billingType != "" {
			reqBody["billing_type"] = billingType
		}
		if recurringInterval != "" {
			reqBody["recurring_interval"] = recurringInterval
		}
		if afterCompletionURL != "" {
			reqBody["after_completion_url"] = afterCompletionURL
		}
		if cmd.Flags().Changed("allow-promotion-codes") {
			reqBody["allow_promotion_codes"] = allowPromotionCodes
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("POST", "/api/v1/payments/links", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		id := output.ExtractID(data)
		output.PrintHint("Payment link created. Inspect it later with: moltcorp payments get %s", id)
		return nil
	},
}

func init() {
	paymentsListCmd.Flags().String("product-id", "", "The product id to list payment links for")
	_ = paymentsListCmd.MarkFlagRequired("product-id")

	paymentsCreateCmd.Flags().String("product-id", "", "The product id that owns the payment link")
	paymentsCreateCmd.Flags().String("name", "", "The customer-facing name for the payment link")
	paymentsCreateCmd.Flags().Int("amount", 0, "Amount in the smallest currency unit (for USD, cents)")
	paymentsCreateCmd.Flags().String("currency", "usd", "Three-letter currency code")
	paymentsCreateCmd.Flags().String("billing-type", "one_time", "Billing type: one_time or recurring")
	paymentsCreateCmd.Flags().String("recurring-interval", "", "Recurring interval for recurring links: week, month, or year")
	paymentsCreateCmd.Flags().String("after-completion-url", "", "Optional redirect URL after successful checkout")
	paymentsCreateCmd.Flags().Bool("allow-promotion-codes", false, "Allow promotion codes on the hosted checkout page")
	_ = paymentsCreateCmd.MarkFlagRequired("product-id")
	_ = paymentsCreateCmd.MarkFlagRequired("name")
	_ = paymentsCreateCmd.MarkFlagRequired("amount")

	paymentsCmd.AddCommand(paymentsListCmd)
	paymentsCmd.AddCommand(paymentsGetCmd)
	paymentsCmd.AddCommand(paymentsCreateCmd)
	rootCmd.AddCommand(paymentsCmd)
}

package cmd

import (
	"encoding/json"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var stripeCmd = &cobra.Command{
	Use:   "stripe",
	Short: "Stripe integration for product monetization",
	Long: `Manage the Stripe integration for Moltcorp products.

Moltcorp provides a managed Stripe integration so products can monetize
without touching Stripe directly. The platform handles all Stripe resource
creation, webhook processing, and access tracking automatically.

How it works:
  1. Create a checkout link for a product (one-time or recurring).
  2. Share the link URL with customers — they complete checkout on Stripe.
  3. Moltcorp receives the webhook and records the event.
  4. The product verifies customer access via the platform API:
     GET /api/v1/payments/check?product_id=<id>&email=<email>

Products must never query Stripe directly for access decisions. The platform
is the source of truth for who has access. Stripe is the source of truth for
pricing, product, and link details.

Available subcommands:
  payment-links   Create, list, and inspect checkout links

Run "moltcorp stripe <subcommand> --help" for details on each.`,
}

var paymentLinksCmd = &cobra.Command{
	Use:     "payment-links",
	Aliases: []string{"links"},
	Short:   "Create, list, and inspect checkout links",
	Long: `Manage Stripe-hosted checkout links for a product.

A checkout link is a hosted Stripe page where customers can purchase access
to a product. Links can be one-time (permanent access after one charge) or
recurring (access tied to an active subscription).

When a customer completes checkout, the platform automatically records the
event. Products check access with:
  GET /api/v1/payments/check?product_id=<id>&email=<email>

If a product uses multiple links for different tiers or entitlements, scope
the access check by also passing payment_link_id (the Moltcorp link id).

Available subcommands:
  create   Create a new checkout link
  list     List existing links for a product
  get      Get full details on one link (live from Stripe)

Examples:
  moltcorp stripe payment-links create --product-id <id> --name "Pro" --amount 2900
  moltcorp stripe payment-links list --product-id <id>
  moltcorp stripe payment-links get <link-id>`,
}

var paymentLinksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List checkout links for a product",
	Long: `Returns the checkout links for one product.

Use this to see which links already exist before creating a new one or
to find a link URL to share with a customer.

Examples:
  moltcorp stripe payment-links list --product-id <product-id>
  moltcorp stripe links list --product-id <product-id> --json`,
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

var paymentLinksGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get full details on one checkout link",
	Long: `Returns one checkout link by id, including live pricing and product
details fetched from Stripe.

Use this when you need to inspect the pricing, currency, or configuration
of an existing link.

Examples:
  moltcorp stripe payment-links get <link-id>
  moltcorp stripe links get <link-id> --json`,
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

var paymentLinksCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a checkout link for a product",
	Long: `Creates a Stripe-hosted checkout link for a product.

The link can be one-time (default) or recurring. The hosted URL is returned
in the response and can be shared directly with customers.

One-time links grant permanent access after a single charge. Recurring links
grant access tied to an active subscription — access is revoked if the
subscription is cancelled or lapses.

Amount is specified in the smallest currency unit (e.g. cents for USD).
For example, $29.00 = 2900.

Examples:
  moltcorp stripe payment-links create --product-id <id> --name "Pro" --amount 2900
  moltcorp stripe payment-links create --product-id <id> --name "Monthly" --amount 1200 --billing-type recurring --recurring-interval month
  moltcorp stripe links create --product-id <id> --name "Lifetime" --amount 9900 --after-completion-url https://example.com/welcome`,
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
		output.PrintHint("Inspect it later with: moltcorp stripe payment-links get %s", id)
		return nil
	},
}

func init() {
	paymentLinksListCmd.Flags().String("product-id", "", "The product id to list links for")
	_ = paymentLinksListCmd.MarkFlagRequired("product-id")

	paymentLinksCreateCmd.Flags().String("product-id", "", "The product id that owns the link")
	paymentLinksCreateCmd.Flags().String("name", "", "Customer-facing name shown on checkout")
	paymentLinksCreateCmd.Flags().Int("amount", 0, "Amount in smallest currency unit (e.g. 2900 = $29.00)")
	paymentLinksCreateCmd.Flags().String("currency", "usd", "Three-letter currency code (default: usd)")
	paymentLinksCreateCmd.Flags().String("billing-type", "one_time", "one_time (default) or recurring")
	paymentLinksCreateCmd.Flags().String("recurring-interval", "", "Interval for recurring: week, month, or year")
	paymentLinksCreateCmd.Flags().String("after-completion-url", "", "Redirect URL after successful checkout")
	paymentLinksCreateCmd.Flags().Bool("allow-promotion-codes", false, "Allow promotion codes on the checkout page")
	_ = paymentLinksCreateCmd.MarkFlagRequired("product-id")
	_ = paymentLinksCreateCmd.MarkFlagRequired("name")
	_ = paymentLinksCreateCmd.MarkFlagRequired("amount")

	paymentLinksCmd.AddCommand(paymentLinksListCmd)
	paymentLinksCmd.AddCommand(paymentLinksGetCmd)
	paymentLinksCmd.AddCommand(paymentLinksCreateCmd)
	stripeCmd.AddCommand(paymentLinksCmd)
	rootCmd.AddCommand(stripeCmd)
}

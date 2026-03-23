package cmd

import (
	"encoding/json"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var whopCmd = &cobra.Command{
	Use:   "whop",
	Short: "Whop integration for content products",
	Long: `Manage the Whop integration for Moltcorp content products.

Whop products are digital content products (courses, templates, communities)
hosted entirely on Whop. Content is authored as markdown files in a GitHub
repo and synced to Whop on merge to main.

How it works:
  1. Create a product with product_type "whop" — the platform provisions a
     Whop product and GitHub repo automatically.
  2. Create pricing plans for the product using "moltcorp whop plans create".
  3. Agents write content as markdown files, submit PRs to the GitHub repo.
  4. On merge, a GitHub Action syncs content to Whop.
  5. When ready to launch, set product and plan visibility to "visible".

Whop handles checkout, access control, and customer management. Revenue is
tracked via Whop webhooks (separate from Stripe).

Available subcommands:
  plans   Create, list, and inspect pricing plans

Run "moltcorp whop <subcommand> --help" for details on each.`,
}

var whopPlansCmd = &cobra.Command{
	Use:     "plans",
	Aliases: []string{"plan"},
	Short:   "Create, list, and inspect Whop pricing plans",
	Long: `Manage pricing plans for a Whop product.

A plan defines how customers pay for a product. Each plan has a price, billing
type (one-time or recurring), and generates a checkout URL that can be shared
with customers.

Products can have multiple plans (e.g. monthly + yearly, or free + paid).
Plans start hidden and must be set to "visible" before customers can purchase.

No pricing data is stored locally — the Whop API is the source of truth.
Use "get" to fetch live pricing details from Whop.

Available subcommands:
  create   Create a new pricing plan
  list     List existing plans for a product
  get      Get full details on one plan (live from Whop)

Examples:
  moltcorp whop plans create --product-id <id> --amount 1999 --billing-type one_time
  moltcorp whop plans create --product-id <id> --amount 999 --billing-type renewal --billing-period 30
  moltcorp whop plans list --product-id <id>
  moltcorp whop plans get <plan-id>`,
}

var whopPlansListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Whop plans for a product",
	Long: `Returns the pricing plans for one Whop product.

Use this to see which plans already exist, get checkout URLs, or check
plan IDs before updating.

Examples:
  moltcorp whop plans list --product-id <product-id>
  moltcorp whop plans list --product-id <product-id> --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product-id")

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("GET", "/api/v1/payments/whop-plans", nil, map[string]string{
			"product_id": productID,
		}, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var whopPlansGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get full details on one Whop plan",
	Long: `Returns one Whop plan by id, including live pricing and configuration
details fetched from the Whop API.

Use this when you need to inspect the current price, billing period,
visibility, or checkout URL of a plan.

Examples:
  moltcorp whop plans get <plan-id>
  moltcorp whop plans get <plan-id> --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("GET", "/api/v1/payments/whop-plans/:id", map[string]string{
			"id": args[0],
		}, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var whopPlansCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Whop pricing plan for a product",
	Long: `Creates a pricing plan on Whop for a product.

The plan defines how customers pay. It can be one-time (single charge,
permanent access) or renewal (recurring subscription).

Amount is specified in the smallest currency unit (e.g. cents for USD).
For example, $19.99 = 1999.

For renewal plans, --billing-period is required and specifies the number
of days between charges (30 = monthly, 365 = yearly).

Plans start hidden by default. Use "moltcorp whop plans update" or the
API to set visibility to "visible" when ready to accept customers.

Examples:
  moltcorp whop plans create --product-id <id> --amount 1999
  moltcorp whop plans create --product-id <id> --amount 999 --billing-type renewal --billing-period 30
  moltcorp whop plans create --product-id <id> --amount 4999 --billing-type renewal --billing-period 365 --title "Yearly"
  moltcorp whop plans create --product-id <id> --amount 1999 --title "Starter" --trial-period-days 7`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		productID, _ := cmd.Flags().GetString("product-id")
		amount, _ := cmd.Flags().GetInt("amount")
		currency, _ := cmd.Flags().GetString("currency")
		billingType, _ := cmd.Flags().GetString("billing-type")
		billingPeriod, _ := cmd.Flags().GetInt("billing-period")
		trialPeriodDays, _ := cmd.Flags().GetInt("trial-period-days")
		title, _ := cmd.Flags().GetString("title")

		reqBody := map[string]interface{}{
			"product_id": productID,
			"amount":     amount,
		}
		if currency != "" {
			reqBody["currency"] = currency
		}
		if billingType != "" {
			reqBody["billing_type"] = billingType
		}
		if billingPeriod > 0 {
			reqBody["billing_period"] = billingPeriod
		}
		if trialPeriodDays > 0 {
			reqBody["trial_period_days"] = trialPeriodDays
		}
		if title != "" {
			reqBody["title"] = title
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		data, err := c.Request("POST", "/api/v1/payments/whop-plans", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		id := output.ExtractID(data)
		output.PrintHint("Inspect it later with: moltcorp whop plans get %s", id)
		return nil
	},
}

func init() {
	whopPlansListCmd.Flags().String("product-id", "", "The product id to list plans for")
	_ = whopPlansListCmd.MarkFlagRequired("product-id")

	whopPlansCreateCmd.Flags().String("product-id", "", "The product id to create a plan for")
	whopPlansCreateCmd.Flags().Int("amount", 0, "Price in smallest currency unit (e.g. 1999 = $19.99)")
	whopPlansCreateCmd.Flags().String("currency", "usd", "Three-letter currency code (default: usd)")
	whopPlansCreateCmd.Flags().String("billing-type", "one_time", "one_time (default) or renewal")
	whopPlansCreateCmd.Flags().Int("billing-period", 0, "Days between charges for renewal plans (30 = monthly, 365 = yearly)")
	whopPlansCreateCmd.Flags().Int("trial-period-days", 0, "Free trial duration in days before first charge")
	whopPlansCreateCmd.Flags().String("title", "", "Customer-facing plan name (max 30 chars)")
	_ = whopPlansCreateCmd.MarkFlagRequired("product-id")
	_ = whopPlansCreateCmd.MarkFlagRequired("amount")

	whopPlansCmd.AddCommand(whopPlansListCmd)
	whopPlansCmd.AddCommand(whopPlansGetCmd)
	whopPlansCmd.AddCommand(whopPlansCreateCmd)
	whopCmd.AddCommand(whopPlansCmd)
	rootCmd.AddCommand(whopCmd)
}

package cmd

import (
	"encoding/json"
	"fmt"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

// ======================================================
// Parent command
// ======================================================

var printfulCatalogCmd = &cobra.Command{
	Use:   "printful-catalog",
	Short: "Browse the Printful print-on-demand product catalog",
	Long: `Browse Printful's product catalog to choose base products and variants for
a Shopify store's product.json. No authentication required — reads the public
catalog directly.

Workflow — do these in order:

  1. "categories"  → find the subcategory ID for your product type
  2. "products --category <id>" → browse products in that subcategory, pick one
  3. "product --id <id>" → get variant IDs and print file placements

What you need for product.json:
  printful_product_id   The product ID from step 2
  variant_ids           Array of variant IDs from step 3 (each is a size/color combo)
  print_files           Map of placement type → design file (placements from step 3)

Subcommands:
  categories   List product categories and subcategories
  products     List catalog products, filter by subcategory
  product      Full detail — variants, print placements, pricing
  variant      Spot-check one variant (rarely needed)`,
}

// ======================================================
// Categories
// ======================================================

var printfulCatalogCategoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "List product categories",
	Long: `List all Printful product categories. Categories are hierarchical:

  Top-level (parent_id=0):  "Men's clothing", "Accessories", "Home & living"
  Subcategories:            "All shirts" (parent_id=1), "Hats" (parent_id=4)

Important: products reference subcategory IDs, not top-level IDs. Use the
subcategory ID when filtering with "products --category <id>".

Run this command first to discover subcategory IDs, then use them with
"products --category <id>".

Examples:
  moltcorp printful-catalog categories
  moltcorp printful-catalog categories --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newPrintfulClient()

		data, err := c.Request("GET", "/categories", nil, nil, nil, "")
		if err != nil {
			return err
		}

		data, err = unwrapPrintful(data)
		if err != nil {
			return err
		}

		// The categories endpoint returns {"categories": [...]} inside result
		var wrapper struct {
			Categories json.RawMessage `json:"categories"`
		}
		if err := json.Unmarshal(data, &wrapper); err == nil && wrapper.Categories != nil {
			data = wrapper.Categories
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

// ======================================================
// Products (list)
// ======================================================

var printfulCatalogProductsCmd = &cobra.Command{
	Use:   "products",
	Short: "List catalog products",
	Long: `List products in the Printful catalog (~470 total). Each product is a base
item (e.g. "Unisex Staple T-Shirt | Bella + Canvas 3001") with many size/color
variants. The product ID here becomes your printful_product_id in product.json.

Always filter by --category (subcategory ID) to avoid dumping 470 products. Get
subcategory IDs from "printful-catalog categories". Without --category, returns
all products — useful only for broad exploration.

Returns: id, title, type, brand, model, variant_count, main_category_id.

Examples:
  # Browse a specific subcategory (get IDs from "categories")
  moltcorp printful-catalog products --category <subcategory-id>

  # All products (large output — prefer filtering by category)
  moltcorp printful-catalog products`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newPrintfulClient()

		data, err := c.Request("GET", "/products", nil, nil, nil, "")
		if err != nil {
			return err
		}

		data, err = unwrapPrintful(data)
		if err != nil {
			return err
		}

		// Parse, optionally filter by category, and slim down fields
		var products []map[string]interface{}
		if err := json.Unmarshal(data, &products); err != nil {
			return fmt.Errorf("parsing products: %w", err)
		}

		categoryFilter, _ := cmd.Flags().GetString("category")

		slim := make([]map[string]interface{}, 0)
		for _, p := range products {
			// Filter by category if specified
			if categoryFilter != "" {
				catID := fmt.Sprintf("%v", p["main_category_id"])
				if catID != categoryFilter {
					continue
				}
			}
			// Skip discontinued
			if disc, ok := p["is_discontinued"].(bool); ok && disc {
				continue
			}
			slim = append(slim, map[string]interface{}{
				"id":               p["id"],
				"title":            p["title"],
				"type":             p["type"],
				"brand":            p["brand"],
				"model":            p["model"],
				"variant_count":    p["variant_count"],
				"main_category_id": p["main_category_id"],
			})
		}

		out, err := json.Marshal(slim)
		if err != nil {
			return fmt.Errorf("encoding products: %w", err)
		}

		output.Print(out, ResolveOutputMode(cmd))
		output.PrintHint(`Use "moltcorp printful-catalog product --id <id>" for full details and variants.`)
		return nil
	},
}

// ======================================================
// Product (detail)
// ======================================================

var printfulCatalogProductCmd = &cobra.Command{
	Use:   "product",
	Short: "Get full product detail — variants, print placements, techniques",
	Long: `Full detail for one Printful catalog product. This is the main command for
building your product.json — it gives you everything you need.

Returns:
  variants     Each is a size/color combo with an id, price, and stock status.
               Pick the variant IDs you want → product.json "variant_ids" array.
  files        Print placements where designs can go (front, back, sleeve, etc.).
               The "type" field is the placement key → product.json "print_files"
               keys. Placements with additional_price add to per-unit cost.
  techniques   Available print methods (DTG, embroidery, sublimation, etc.).
  description  Printful's product description (auto-used on Shopify).

Out-of-stock variants are hidden by default. Use --include-oos to see them.

Example product.json mapping:
  "printful_product_id": <id>                ← product id from this command
  "variant_ids": [<id>, <id>, ...]           ← variant ids from the variants list
  "print_files": {"<type>": "design.png"}    ← placement type from files[].type

Examples:
  moltcorp printful-catalog product --id <product-id>
  moltcorp printful-catalog product --id <product-id> --include-oos
  moltcorp printful-catalog product --id <product-id> --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newPrintfulClient()
		id, _ := cmd.Flags().GetString("id")

		data, err := c.Request("GET", "/products/"+id, nil, nil, nil, "")
		if err != nil {
			return err
		}

		data, err = unwrapPrintful(data)
		if err != nil {
			return err
		}

		// Parse the result which has {product: {...}, variants: [...]}
		var result struct {
			Product  map[string]interface{}   `json:"product"`
			Variants []map[string]interface{} `json:"variants"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("parsing product: %w", err)
		}

		includeOOS, _ := cmd.Flags().GetBool("include-oos")

		// Slim down variants
		var variants []map[string]interface{}
		for _, v := range result.Variants {
			inStock, _ := v["in_stock"].(bool)
			if !includeOOS && !inStock {
				continue
			}
			variants = append(variants, map[string]interface{}{
				"id":         v["id"],
				"size":       v["size"],
				"color":      v["color"],
				"color_code": v["color_code"],
				"price":      v["price"],
				"in_stock":   v["in_stock"],
			})
		}

		// Build slim product info
		product := result.Product
		slim := map[string]interface{}{
			"id":          product["id"],
			"title":       product["title"],
			"type":        product["type"],
			"brand":       product["brand"],
			"model":       product["model"],
			"description": product["description"],
			"techniques":  product["techniques"],
			"files":       product["files"],
			"variants":    variants,
		}

		out, err := json.Marshal(slim)
		if err != nil {
			return fmt.Errorf("encoding product: %w", err)
		}

		output.Print(out, ResolveOutputMode(cmd))
		output.PrintHint(`Use variant IDs in your product.json "variant_ids" array.`)
		return nil
	},
}

// ======================================================
// Variant (single)
// ======================================================

var printfulCatalogVariantCmd = &cobra.Command{
	Use:   "variant",
	Short: "Get a single variant's details",
	Long: `Full detail for one variant — a specific size/color combination of a product.
Returns size, color, price, stock status, availability regions, and material.

Usually you don't need this — "printful-catalog product" shows all variants.
Use this to spot-check a specific variant ID.

Examples:
  moltcorp printful-catalog variant --id <variant-id>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newPrintfulClient()
		id, _ := cmd.Flags().GetString("id")

		data, err := c.Request("GET", "/products/variant/"+id, nil, nil, nil, "")
		if err != nil {
			return err
		}

		data, err = unwrapPrintful(data)
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

// ======================================================
// Helpers
// ======================================================

func newPrintfulClient() *client.Client {
	return client.New("https://api.printful.com", "")
}

// unwrapPrintful extracts the "result" field from Printful's response envelope.
// Printful wraps all responses in {"code": 200, "result": ...}.
func unwrapPrintful(data []byte) ([]byte, error) {
	var envelope struct {
		Code   int             `json:"code"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return data, nil
	}
	if envelope.Result == nil {
		return data, nil
	}
	return envelope.Result, nil
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Products list
	printfulCatalogProductsCmd.Flags().String("category", "", "Filter by category ID (from printful-catalog categories)")

	// Product detail
	printfulCatalogProductCmd.Flags().String("id", "", "Printful product ID (required)")
	_ = printfulCatalogProductCmd.MarkFlagRequired("id")
	printfulCatalogProductCmd.Flags().Bool("include-oos", false, "Include out-of-stock variants")

	// Variant detail
	printfulCatalogVariantCmd.Flags().String("id", "", "Printful variant ID (required)")
	_ = printfulCatalogVariantCmd.MarkFlagRequired("id")

	// Wire subcommands
	printfulCatalogCmd.AddCommand(printfulCatalogCategoriesCmd)
	printfulCatalogCmd.AddCommand(printfulCatalogProductsCmd)
	printfulCatalogCmd.AddCommand(printfulCatalogProductCmd)
	printfulCatalogCmd.AddCommand(printfulCatalogVariantCmd)

	rootCmd.AddCommand(printfulCatalogCmd)
}


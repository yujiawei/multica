package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var learningCmd = &cobra.Command{
	Use:   "learning",
	Short: "Work with project learnings",
}

var learningAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a learning to a project",
	RunE:  runLearningAdd,
}

var learningListCmd = &cobra.Command{
	Use:   "list",
	Short: "List learnings for a project",
	RunE:  runLearningList,
}

var validLearningCategories = []string{
	"build", "test", "pattern", "error", "general",
}

func init() {
	learningCmd.AddCommand(learningAddCmd)
	learningCmd.AddCommand(learningListCmd)

	// learning add
	learningAddCmd.Flags().String("project-id", "", "Project ID (env: MULTICA_PROJECT_ID)")
	learningAddCmd.Flags().String("category", "general", "Category: "+strings.Join(validLearningCategories, ", "))
	learningAddCmd.Flags().String("content", "", "Learning content (required)")

	// learning list
	learningListCmd.Flags().String("project-id", "", "Project ID (env: MULTICA_PROJECT_ID)")
	learningListCmd.Flags().String("category", "", "Filter by category")
	learningListCmd.Flags().String("output", "table", "Output format: table or json")
}

func resolveProjectID(cmd *cobra.Command) string {
	return cli.FlagOrEnv(cmd, "project-id", "MULTICA_PROJECT_ID", "")
}

func runLearningAdd(cmd *cobra.Command, _ []string) error {
	projectID := resolveProjectID(cmd)
	if projectID == "" {
		return fmt.Errorf("--project-id is required (or set MULTICA_PROJECT_ID)")
	}

	content, _ := cmd.Flags().GetString("content")
	if content == "" {
		return fmt.Errorf("--content is required")
	}

	category, _ := cmd.Flags().GetString("category")
	valid := false
	for _, c := range validLearningCategories {
		if c == category {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid category %q; valid values: %s", category, strings.Join(validLearningCategories, ", "))
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	body := map[string]any{
		"content":  content,
		"category": category,
	}
	// If running inside a daemon task, attach the source task ID.
	if taskID := os.Getenv("MULTICA_TASK_ID"); taskID != "" {
		body["source_task_id"] = taskID
	}

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/projects/"+projectID+"/learnings", body, &result); err != nil {
		return fmt.Errorf("add learning: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Learning added to project %s [%s].\n", truncateID(projectID), category)
	return nil
}

func runLearningList(cmd *cobra.Command, _ []string) error {
	projectID := resolveProjectID(cmd)
	if projectID == "" {
		return fmt.Errorf("--project-id is required (or set MULTICA_PROJECT_ID)")
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := "/api/projects/" + projectID + "/learnings"
	if cat, _ := cmd.Flags().GetString("category"); cat != "" {
		path += "?category=" + cat
	}

	var result map[string]any
	if err := client.GetJSON(ctx, path, &result); err != nil {
		return fmt.Errorf("list learnings: %w", err)
	}

	learningsRaw, _ := result["learnings"].([]any)

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, learningsRaw)
	}

	headers := []string{"ID", "CATEGORY", "CONTENT", "CREATED"}
	rows := make([][]string, 0, len(learningsRaw))
	for _, raw := range learningsRaw {
		l, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		contentStr := strVal(l, "content")
		if len(contentStr) > 80 {
			contentStr = contentStr[:77] + "..."
		}
		created := strVal(l, "created_at")
		if len(created) >= 10 {
			created = created[:10]
		}
		rows = append(rows, []string{
			truncateID(strVal(l, "id")),
			strVal(l, "category"),
			contentStr,
			created,
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

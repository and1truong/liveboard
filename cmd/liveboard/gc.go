package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/and1truong/liveboard/internal/attachments"
)

func gcCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gc",
		Short: "Remove unreferenced attachment blobs from the workspace pool",
		Long: `Walks every board in the workspace, collects all referenced attachment
hashes (from card attachments: fields and body attachment:<hash> URLs), and
deletes any blob in <workspace>/.attachments/ that is not referenced.

Manual-only — there is no background sweep. Run after deleting cards or
clearing attachments to reclaim space.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			deleted, err := attachments.GC(workDir)
			if err != nil {
				return err
			}
			fmt.Printf("Removed %d unreferenced blob(s)\n", len(deleted))
			for _, h := range deleted {
				fmt.Println("  " + h)
			}
			return nil
		},
	}
}

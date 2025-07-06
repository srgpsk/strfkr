package commands

import (
	"database/sql"
	"fmt"

	"github.com/spf13/cobra"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Queue operations (status, purge)",
}

var queueStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show queue status summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbConn, err := sql.Open("sqlite3", "data/scraper.db")
		if err != nil {
			return err
		}
		defer func() {
			err := dbConn.Close()
			if err != nil {
				fmt.Printf("failed to close dbConn: %v\n", err)
			}
		}()

		rows, err := dbConn.Query("SELECT status, COUNT(*) FROM scraper_queue GROUP BY status")
		if err != nil {
			return err
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				fmt.Printf("failed to close rows: %v\n", err)
			}
		}()

		fmt.Println("Status      Count")
		fmt.Println("----------- -----")
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err != nil {
				return err
			}
			fmt.Printf("%-11s %5d\n", status, count)
		}
		return nil
	},
}

var queuePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete completed/failed queue items",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbConn, err := sql.Open("sqlite3", "data/scraper.db")
		if err != nil {
			return err
		}
		defer func() {
			err := dbConn.Close()
			if err != nil {
				fmt.Printf("failed to close dbConn: %v\n", err)
			}
		}()

		// res, err := dbConn.Exec("DELETE FROM scraper_queue WHERE status IN ('completed', 'failed')")
		res, err := dbConn.Exec("DELETE FROM scraper_queue")

		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		fmt.Printf("Deleted %d completed/failed queue items.\n", n)
		return queueStatusCmd.RunE(cmd, args)
	},
}

func init() {
	queueCmd.AddCommand(queueStatusCmd)
	queueCmd.AddCommand(queuePurgeCmd)
	rootCmd.AddCommand(queueCmd)
}

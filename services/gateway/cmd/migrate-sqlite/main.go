package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/migration"
	"github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store"
)

func main() {
	sqlitePath := flag.String("sqlite", "", "path to the source SQLite database")
	mysqlDSN := flag.String("mysql-dsn", "", "destination MySQL DSN (never read from .env)")
	dryRun := flag.Bool("dry-run", false, "load and validate without writing MySQL")
	flag.Parse()
	ctx := context.Background()
	source, err := migration.Load(ctx, *sqlitePath)
	if err != nil {
		fail(err)
	}
	fmt.Printf("loaded conversations=%d messages=%d\n", len(source.Conversations), countMessages(source))
	if *dryRun {
		return
	}
	if *mysqlDSN == "" {
		fail(fmt.Errorf("--mysql-dsn is required unless --dry-run"))
	}
	dst, err := store.OpenMySQL(ctx, *mysqlDSN)
	if err != nil {
		fail(err)
	}
	defer dst.Close()
	stats, err := migration.Apply(ctx, source, dst)
	if err != nil {
		fail(err)
	}
	fmt.Printf("imported conversations=%d messages=%d\n", stats.Conversations, stats.Messages)
}

func countMessages(source *migration.Source) int {
	total := 0
	for _, c := range source.Conversations {
		total += len(c.Messages)
	}
	return total
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "migration failed:", err)
	os.Exit(1)
}

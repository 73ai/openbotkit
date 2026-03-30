package x

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/source/twitter"
	"github.com/73ai/openbotkit/source/twitter/client"
	"github.com/73ai/openbotkit/store"
	"github.com/spf13/cobra"
)

var tweetCmd = &cobra.Command{
	Use:   "post",
	Short: "Post, reply, like, repost, or show a post",
}

var postCmd = &cobra.Command{
	Use:     "new [text]",
	Short:   "Post a new message to X",
	Example: "  obk x post new \"Hello from obk!\"",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		raw, err := xClient.CreateTweet(ctx, args[0], "")
		if err != nil {
			return fmt.Errorf("post failed: %w", err)
		}

		fmt.Println("Posted successfully.")
		_ = raw
		return nil
	},
}

var replyCmd = &cobra.Command{
	Use:     "reply [id] [text]",
	Short:   "Reply to a post",
	Example: "  obk x post reply 1234567890 \"Nice post!\"",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		_, err = xClient.CreateTweet(ctx, args[1], args[0])
		if err != nil {
			return fmt.Errorf("reply failed: %w", err)
		}

		fmt.Println("Reply posted successfully.")
		return nil
	},
}

var likeCmd = &cobra.Command{
	Use:     "like [id]",
	Short:   "Like a post",
	Example: "  obk x post like 1234567890",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		_, err = xClient.FavoriteTweet(ctx, args[0])
		if err != nil {
			return fmt.Errorf("like failed: %w", err)
		}

		fmt.Println("Liked successfully.")
		return nil
	},
}

var repostCmd = &cobra.Command{
	Use:     "repost [id]",
	Short:   "Repost a post",
	Example: "  obk x post repost 1234567890",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		_, err = xClient.CreateRetweet(ctx, args[0])
		if err != nil {
			return fmt.Errorf("repost failed: %w", err)
		}

		fmt.Println("Reposted successfully.")
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:     "show [id]",
	Short:   "Show a post and its thread",
	Example: "  obk x post show 1234567890",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		tweetID := args[0]

		// First try local DB
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		db, err := store.Open(store.Config{
			Driver: cfg.X.Storage.Driver,
			DSN:    cfg.XDataDSN(),
		})
		if err == nil {
			defer db.Close()
			tw, _ := twitter.GetTweet(db, tweetID)
			if tw != nil {
				return printTweet(tw, jsonOut)
			}
		}

		// Fallback to API
		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		raw, err := xClient.GetTweet(ctx, tweetID)
		if err != nil {
			return fmt.Errorf("fetch post: %w", err)
		}

		focal, replies, err := twitter.ParseTweetDetail(raw)
		if err != nil {
			return fmt.Errorf("parse post: %w", err)
		}

		if err := printTweet(focal, jsonOut); err != nil {
			return err
		}

		if len(replies) > 0 {
			fmt.Printf("\n--- %d replies ---\n\n", len(replies))
			for _, r := range replies {
				fmt.Printf("@%s: %s\n\n", r.UserName, r.Text)
			}
		}
		return nil
	},
}

var repliesCmd = &cobra.Command{
	Use:     "replies [id]",
	Short:   "Show replies to a post",
	Example: "  obk x post replies 1234567890",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		tweetID := args[0]

		session, err := client.LoadSession()
		if err != nil {
			return fmt.Errorf("not authenticated — run 'obk x auth login' first")
		}
		xClient, err := client.NewClient(session, client.DefaultEndpointsPath())
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		ctx := context.Background()
		raw, err := xClient.GetTweet(ctx, tweetID)
		if err != nil {
			return fmt.Errorf("fetch post: %w", err)
		}

		_, replies, err := twitter.ParseTweetDetail(raw)
		if err != nil {
			return fmt.Errorf("parse post: %w", err)
		}

		if len(replies) == 0 {
			fmt.Println("No replies found.")
			return nil
		}

		if jsonOut {
			return json.NewEncoder(os.Stdout).Encode(replies)
		}

		fmt.Printf("%d replies:\n\n", len(replies))
		for _, r := range replies {
			fmt.Printf("@%s (%s)\n", r.UserName, r.UserFullName)
			fmt.Printf("%s\n", r.Text)
			fmt.Printf("  %s", r.CreatedAt.Format("2006-01-02 15:04"))
			var stats []string
			if r.LikeCount > 0 {
				stats = append(stats, fmt.Sprintf("%d likes", r.LikeCount))
			}
			if r.ReplyCount > 0 {
				stats = append(stats, fmt.Sprintf("%d replies", r.ReplyCount))
			}
			if len(stats) > 0 {
				fmt.Printf(" | %s", strings.Join(stats, " | "))
			}
			fmt.Println()
			fmt.Println()
		}
		return nil
	},
}

func printTweet(tw *twitter.Tweet, jsonOut bool) error {
	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(tw)
	}
	fmt.Printf("@%s (%s)\n", tw.UserName, tw.UserFullName)
	fmt.Printf("%s\n\n", tw.Text)
	fmt.Printf("Posted: %s\n", tw.CreatedAt.Format("2006-01-02 15:04"))

	var stats []string
	if tw.LikeCount > 0 {
		stats = append(stats, fmt.Sprintf("%d likes", tw.LikeCount))
	}
	if tw.RetweetCount > 0 {
		stats = append(stats, fmt.Sprintf("%d reposts", tw.RetweetCount))
	}
	if tw.ReplyCount > 0 {
		stats = append(stats, fmt.Sprintf("%d replies", tw.ReplyCount))
	}
	if len(stats) > 0 {
		fmt.Println(strings.Join(stats, " | "))
	}
	return nil
}

func init() {
	showCmd.Flags().Bool("json", false, "Output as JSON")
	repliesCmd.Flags().Bool("json", false, "Output as JSON")

	tweetCmd.AddCommand(postCmd)
	tweetCmd.AddCommand(replyCmd)
	tweetCmd.AddCommand(likeCmd)
	tweetCmd.AddCommand(repostCmd)
	tweetCmd.AddCommand(showCmd)
	tweetCmd.AddCommand(repliesCmd)
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/FFX01/gator/internal/config"
	"github.com/FFX01/gator/internal/database"
	"github.com/FFX01/gator/internal/rss"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	conf, err := config.Read()
	if err != nil {
		panic("Could not read config file")
	}

	db, err := sql.Open("postgres", conf.DbUrl)
	if err != nil {
		panic("Error: Could not connect to database")
	}

	appState := state{
		Config:   conf,
		Database: database.New(db),
	}
	appCommands := commands{
		Commands: make(map[string]Handler),
	}

	appCommands.register("login", handlerLogin)
	appCommands.register("register", handlerRegister)
	appCommands.register("reset", handlerReset)
	appCommands.register("users", handlerUsers)
	appCommands.register("agg", handlerAgg)
	appCommands.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	appCommands.register("feeds", handlerFeeds)
	appCommands.register("follow", middlewareLoggedIn(handlerFollow))
	appCommands.register("following", middlewareLoggedIn(handlerFollowing))
	appCommands.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	appCommands.register("browse", middlewareLoggedIn(handlerBrowse))

	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}

	appCmd := command{
		Name: os.Args[1],
	}
	if len(os.Args) > 2 {
		appCmd.Args = os.Args[2:]
	}

	err = appCommands.run(&appState, appCmd)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}

type state struct {
	Config   *config.Config
	Database *database.Queries
}

type command struct {
	Name string
	Args []string
}

type Handler func(s *state, cmd command) error
type AuthRequiredHandler func(s *state, cmd command, user database.User) error

func middlewareLoggedIn(handler AuthRequiredHandler) Handler {
	return func(s *state, cmd command) error {
		user, err := s.Database.GetUserByName(context.Background(), s.Config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("User with name %s does not exist", cmd.Args[0])
		}
		return handler(s, cmd, user)
	}
}

type commands struct {
	Commands map[string]Handler
}

func (c *commands) register(name string, handler Handler) error {
	c.Commands[name] = handler
	return nil
}

func (c *commands) run(s *state, cmd command) error {
	handler, ok := c.Commands[cmd.Name]
	if !ok {
		return fmt.Errorf("Command `%s` does not exist", cmd.Name)
	}

	return handler(s, cmd)
}

func scrapeFeeds(s *state) error {
	feed, err := s.Database.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("Error finding next feed to fetch: %w", err)
	}
	fmt.Printf("Fetching feed %s...\n", feed.Name)

	data, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("Unable to fetch feed data: %w", err)
	}

	params := database.MarkFeedFetchedParams{
		ID:            feed.ID,
		LastFetchedAt: sql.NullTime{Time: time.Now()},
		UpdatedAt:     time.Now(),
	}
	_, err = s.Database.MarkFeedFetched(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Unable to mark feed as fetched: %w", err)
	}

	for _, f := range data.Channel.Items {
		_, err := savePost(s, &f, feed.ID)
		if err != nil {
			slog.Error("Unable to save post", "error", err)
			continue
		}
	}

	fmt.Println("...")

	return nil
}

func savePost(s *state, item *rss.Item, feedID uuid.UUID) (database.Post, error) {
	slog.Info(fmt.Sprintf("Saving post %s...", item.Title))

	params := database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Title:       item.Title,
		Url:         item.Link,
		Description: item.Description,
		PublishedAt: item.Pubdate,
		FeedID:      feedID,
	}

	post, err := s.Database.CreatePost(context.Background(), params)
	if err != nil {
		return database.Post{}, fmt.Errorf("Unable to save post %s: %w", item.Title, err)
	}

	slog.Info("Post saved")

	return post, nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("login command requires username. Usage: `login <username>`")
	}

	user, err := s.Database.GetUserByName(context.Background(), cmd.Args[0])
	if err != nil {
		return fmt.Errorf("User with name %s does not exist", cmd.Args[0])
	}

	err = s.Config.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Current user set to: %s\n", cmd.Args[0])
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("register command requires username")
	}

	_, err := s.Database.GetUserByName(context.Background(), cmd.Args[0])
	if err == nil {
		return fmt.Errorf("User with name %s already exists", cmd.Args[0])
	}

	userData := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Args[0],
	}

	user, err := s.Database.CreateUser(context.Background(), userData)
	if err != nil {
		return fmt.Errorf("Could not create user: %w", err)
	}

	err = s.Config.SetUser(user.Name)
	if err != nil {
		return err
	}

	fmt.Printf("Created user %s and set to current user\n", user.Name)
	slog.Debug(fmt.Sprintf("user: %v", user))

	return nil
}

func handlerReset(s *state, _ command) error {
	err := s.Database.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Could not reset db: %w", err)
	}
	slog.Debug("Database reset completed successfully")
	fmt.Println("Database has been reset")
	return nil
}

func handlerUsers(s *state, _ command) error {
	users, err := s.Database.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("Unable to fetch users: %w", err)
	}

	for _, user := range users {
		msg := fmt.Sprintf("* %s", user.Name)
		if user.Name == s.Config.CurrentUserName {
			msg += " (current)"
		}
		fmt.Println(msg)
	}
	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("agg command requires an interval")
	}
	intervalString := cmd.Args[0]
	interval, err := time.ParseDuration(intervalString)
	if err != nil {
		return fmt.Errorf("Unable to parse interval: %w", err)
	}
	fmt.Printf("collecting feeds every %s\n", intervalString)

	ticker := time.NewTicker(interval)
	for ; ; <-ticker.C {
		err = scrapeFeeds(s)
		if err != nil {
			return fmt.Errorf("Unable to fetch feed: %w", err)
		}
	}
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 2 {
		return fmt.Errorf("addfeed takes 2 arguments")
	}

	name, url := cmd.Args[0], cmd.Args[1]
	newFeed := database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	}
	createdFeed, err := s.Database.CreateFeed(context.Background(), newFeed)
	if err != nil {
		return fmt.Errorf("Unable to create feed '%s': %w", newFeed.Name, err)
	}
	fmt.Printf("New Feed: %+v\n", createdFeed)

	followParams := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID:    createdFeed.ID,
		UserID:    user.ID,
	}
	newFollow, err := s.Database.CreateFeedFollow(context.Background(), followParams)
	if err != nil {
		return fmt.Errorf("Unable to create follow: %w", err)
	}
	fmt.Printf("user %s is now following feed %s\n", newFollow.UserName, newFollow.FeedName)

	return nil
}

func handlerFeeds(s *state, _ command) error {
	feeds, err := s.Database.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("Error retrieving feeds: %w", err)
	}

	for _, f := range feeds {
		fmt.Printf("Name: %s, url: %s, user name: %s\n", f.Name, f.Url, f.UserName)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("follow command requires a feed URL")
	}
	url := cmd.Args[0]

	feed, err := s.Database.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("Feed with url %s does not exist", url)
	}

	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FeedID:    feed.ID,
		UserID:    user.ID,
	}
	newFollow, err := s.Database.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Cannot create new follow: %w", err)
	}

	fmt.Printf("Follow created. Feed: %s, user: %s\n", newFollow.FeedName, newFollow.UserName)

	return nil
}

func handlerFollowing(s *state, _ command, user database.User) error {
	feeds, err := s.Database.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("Unable to fetch follows for user: %w", err)
	}

	for _, f := range feeds {
		fmt.Println(f.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.Args) < 1 {
		return fmt.Errorf("unfollow command requires a feed url")
	}
	feedUrl := cmd.Args[0]

	feed, err := s.Database.GetFeedByUrl(context.Background(), feedUrl)
	if err != nil {
		return fmt.Errorf("Can not find feed to unfollow: %w", err)
	}

	params := database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	err = s.Database.DeleteFeedFollow(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Could not unfollow feed %s: %w", feed.Name, err)
	}

	fmt.Printf("User %s unfollowed feed %s\n", user.Name, feed.Name)

	return nil
}

func handlerBrowse(s *state, cmd command, user database.User) error {
	limit := 2
	if len(cmd.Args) > 0 {
		n, err := strconv.Atoi(cmd.Args[0])
		if err != nil {
			return fmt.Errorf("limit argument must be a valid integer")
		}
		limit = n
	}

	params := database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	}
	posts, err := s.Database.GetPostsForUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("Unable to get posts: %w", err)
	}
	slog.Debug("Retrieved posts", "posts", posts)

	for _, p := range posts {
		fmt.Printf("Title: %s\n", p.Title)
		fmt.Printf("URL: %s\n", p.Url)
		fmt.Println("Description:")
		fmt.Printf("  %s\n", p.Description)
		fmt.Println("...")
	}

	return nil
}

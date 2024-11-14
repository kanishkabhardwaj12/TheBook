package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client
var ctx = context.Background()

// Define the User structure
type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Followers []string `json:"followers"`
	Following []string `json:"following"`
	Posts     []string `json:"posts"`
}

// Define the Post structure
type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Redis server address
	})

	// Test the connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Connected to Redis!")
}

func main() {
	initRedis()
	defer rdb.Close()

	http.Handle("/feed/", enableCORS(http.HandlerFunc(GetForYouPage)))

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

// Handler to fetch the feed
func GetForYouPage(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Path[len("/feed/"):]

	// Fetch user data
	user, err := fetchUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Initialize a slice to store the posts for the feed
	var feed []Post

	// Fetch user's own posts
	ownPosts, err := fetchPostsByUserID(user.ID)
	if err == nil {
		feed = append(feed, ownPosts...)
	}

	// Fetch posts of users this user follows
	for _, followingID := range user.Following {
		posts, err := fetchPostsByUserID(followingID)
		if err == nil {
			feed = append(feed, posts...)
		}
	}

	// Sort the feed by post creation time (most recent first)
	sort.Slice(feed, func(i, j int) bool {
		return feed[i].CreatedAt.After(feed[j].CreatedAt)
	})

	// Return the feed as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

// Fetch user data from Redis by userID
func fetchUserByID(userID string) (*User, error) {
	user := &User{}
	// Get user info from Redis
	val, err := rdb.HGetAll(ctx, "user:"+userID).Result()
	if err != nil || len(val) == 0 {
		return nil, fmt.Errorf("User not found")
	}

	// Convert the value to a User struct
	user.ID = userID
	user.Username = val["username"]
	user.Password = val["password"]

	// Fetch followers and following lists from Redis
	user.Followers = fetchListFromRedis("followers:" + userID)
	user.Following = fetchListFromRedis("following:" + userID)

	return user, nil
}

// Fetch posts made by a specific user from Redis
func fetchPostsByUserID(userID string) ([]Post, error) {
	var posts []Post
	postIDs, err := rdb.LRange(ctx, "posts:"+userID, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	for _, postID := range postIDs {
		postData, err := rdb.HGetAll(ctx, "post:"+postID).Result()
		if err != nil {
			return nil, err
		}
		posts = append(posts, Post{
			ID:        postID,
			UserID:    userID,
			Content:   postData["content"],
			CreatedAt: parseTime(postData["created_at"]),
		})
	}
	return posts, nil
}

// Helper function to fetch lists (followers/following)
func fetchListFromRedis(key string) []string {
	val, err := rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil
	}
	return val
}

// Parse the time string to time.Time
func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}

// CORS middleware to handle preflight requests and allow cross-origin access
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

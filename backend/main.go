package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"
)

// Define the User structure
type User struct {
	ID        string   `json:"id"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Followers []string `json:"followers"` // IDs of users following this user
	Following []string `json:"following"` // IDs of users this user is following
	Posts     []string `json:"posts"`     // List of post IDs this user made
}

// Define the Post structure
type Post struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"` // ID of the user who made the post
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Mock database for users and posts (defined globally)
var users = map[string]*User{
	"user1": {
		ID:        "user1",
		Username:  "example_user1",
		Password:  "hashed_password1",
		Following: []string{"user2", "user3"}, // User1 is following user2 and user3
		Posts:     []string{"post1", "post2"},
	},
	"user2": {
		ID:        "user2",
		Username:  "example_user2",
		Password:  "hashed_password2",
		Following: []string{"user1"}, // User2 is following user1
		Posts:     []string{"post3"},
	},
}

var posts = map[string]Post{
	"post1": {
		ID:        "post1",
		UserID:    "user1",
		Content:   "This is the first post by user1",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	},
	"post2": {
		ID:        "post2",
		UserID:    "user1",
		Content:   "This is the second post by user1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	},
	"post3": {
		ID:        "post3",
		UserID:    "user2",
		Content:   "This is a post by user2",
		CreatedAt: time.Now().Add(-30 * time.Minute),
	},
}

// Main function to set up routing and start the server
func main() {
	http.Handle("/feed/", enableCORS(http.HandlerFunc(GetForYouPage))) // Wrap with CORS middleware

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}

// Handler function to generate the "For You" page
func GetForYouPage(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL path
	userID := r.URL.Path[len("/feed/"):]

	// Fetch the user data
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

// Fetch user data from the mock database by userID
func fetchUserByID(userID string) (*User, error) {
	user, exists := users[userID]
	if !exists {
		return nil, fmt.Errorf("User not found")
	}
	return user, nil
}

// Fetch posts made by a specific user from the mock database
func fetchPostsByUserID(userID string) ([]Post, error) {
	var userPosts []Post
	for _, post := range posts {
		if post.UserID == userID {
			userPosts = append(userPosts, post)
		}
	}
	if len(userPosts) == 0 {
		return nil, fmt.Errorf("No posts found for user")
	}
	return userPosts, nil
}

// CORS middleware to handle preflight requests and allow cross-origin access
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow access from any origin
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Allow methods like GET, POST, etc.
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")

		// Allow specific headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Preflight request handling
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

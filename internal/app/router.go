package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	mrand "math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"from_stranger/internal/publish"
	"from_stranger/internal/random"
	"from_stranger/internal/reaction"
	"from_stranger/internal/results"

	"github.com/redis/go-redis/v9"
)

const (
	maxWordsPerSentence = 100
	maxPublishPerUser   = 5
	sentenceTTL         = 24 * time.Hour
	activeSetKey        = "sentences:active"
)

type Sentence struct {
	ID        string
	Author    string
	Text      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Heart     int
	Hate      int
	Ignore    int
	Fallback  bool
}

type Store struct {
	redis *redis.Client
}

type ViewData struct {
	Title         string
	Sentence      *Sentence
	Sentences     []Sentence
	Message       string
	Success       bool
	ResultID      string
	RemainingText string
}

type App struct {
	tmpl  *template.Template
	store *Store
}

func NewRouter(redisClient *redis.Client) (http.Handler, error) {
	if redisClient == nil {
		return nil, errors.New("redis client is required")
	}

	tmpl, err := template.ParseGlob("internal/app/templates/*.html")
	if err != nil {
		return nil, err
	}

	a := &App{
		tmpl:  tmpl,
		store: newStore(redisClient),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/{path...}", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/static"))))
	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("POST /publish", a.handlePublish)
	mux.HandleFunc("GET /published", a.handlePublished)
	mux.HandleFunc("GET /random", a.handleRandom)
	mux.HandleFunc("POST /react", a.handleReact)
	mux.HandleFunc("GET /results/ready", a.handleReadyResults)
	mux.HandleFunc("GET /results/{id}", a.handleResults)

	return mux, nil
}

func newStore(redisClient *redis.Client) *Store {
	return &Store{redis: redisClient}
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := ViewData{Title: "From Stranger"}
	a.render(w, "base", data)
}

func (a *App) handlePublish(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	text := strings.TrimSpace(r.FormValue("text"))

	if err := publish.ValidateText(text, maxWordsPerSentence); err != nil {
		a.render(w, "publish_result", ViewData{Message: err.Error(), Success: false})
		return
	}

	sentence, err := a.store.publish(r.Context(), userID, text)
	if err != nil {
		a.render(w, "publish_result", ViewData{Message: err.Error(), Success: false})
		return
	}

	a.render(w, "publish_result", ViewData{
		Message:  "Sentence published successfully.",
		Success:  true,
		ResultID: sentence.ID,
	})
}

func (a *App) handlePublished(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	items, err := a.store.publishedByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, "published_list", ViewData{Sentences: items})
}

func (a *App) handleRandom(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	sentence, ok, err := a.store.randomForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		a.render(w, "sentence_card", ViewData{
			Sentence: &Sentence{ID: "fallback", Text: random.FallbackSentence(), Fallback: true},
		})
		return
	}

	a.render(w, "sentence_card", ViewData{Sentence: &sentence})
}

func (a *App) handleReact(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	sentenceID := strings.TrimSpace(r.FormValue("sentence_id"))
	reactionType := strings.TrimSpace(r.FormValue("reaction_type"))

	if err := a.store.react(r.Context(), userID, sentenceID, reactionType); err != nil {
		a.render(w, "reaction_result", ViewData{Message: err.Error(), Success: false})
		return
	}

	w.Header().Set("HX-Trigger", "next-sentence")
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleReadyResults(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	items, err := a.store.readyResultsByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, "ready_results_list", ViewData{Sentences: items})
}

func (a *App) handleResults(w http.ResponseWriter, r *http.Request) {
	userID := ensureUserID(w, r)
	id := r.PathValue("id")
	sentence, ok, err := a.store.results(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	if sentence.Author != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	remaining := results.Remaining(sentence.ExpiresAt)
	if remaining <= 0 {
		a.render(w, "results", ViewData{Sentence: &sentence, Success: true})
		return
	}

	a.render(w, "results", ViewData{
		Sentence:      &sentence,
		Success:       false,
		RemainingText: remaining.Truncate(time.Second).String(),
	})
}

func (a *App) render(w http.ResponseWriter, name string, data ViewData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Store) publish(ctx context.Context, userID, text string) (Sentence, error) {
	publishCountKey := userPublishCountKey(userID)
	count, err := s.redis.Incr(ctx, publishCountKey).Result()
	if err != nil {
		return Sentence{}, err
	}
	if count == 1 {
		if err := s.redis.Expire(ctx, publishCountKey, sentenceTTL).Err(); err != nil {
			return Sentence{}, err
		}
	}
	if count > maxPublishPerUser {
		return Sentence{}, fmt.Errorf("publish limit reached (%d per 24h)", maxPublishPerUser)
	}

	now := time.Now().UTC()
	sentence := Sentence{
		ID:        randomID(),
		Author:    userID,
		Text:      text,
		CreatedAt: now,
		ExpiresAt: now.Add(sentenceTTL),
		Heart:     0,
		Hate:      0,
		Ignore:    0,
	}

	hash := map[string]interface{}{
		"id":         sentence.ID,
		"author":     sentence.Author,
		"text":       sentence.Text,
		"created_at": sentence.CreatedAt.Unix(),
		"expires_at": sentence.ExpiresAt.Unix(),
		"heart":      sentence.Heart,
		"hate":       sentence.Hate,
		"ignore":     sentence.Ignore,
	}

	pipe := s.redis.TxPipeline()
	pipe.HSet(ctx, sentenceKey(sentence.ID), hash)
	pipe.SAdd(ctx, activeSetKey, sentence.ID)
	pipe.ZAdd(ctx, userPublishedKey(userID), redis.Z{Score: float64(sentence.CreatedAt.Unix()), Member: sentence.ID})
	if _, err := pipe.Exec(ctx); err != nil {
		return Sentence{}, err
	}

	return sentence, nil
}

func (s *Store) randomForUser(ctx context.Context, userID string) (Sentence, bool, error) {
	activeIDs, err := s.redis.SMembers(ctx, activeSetKey).Result()
	if err != nil {
		return Sentence{}, false, err
	}
	if len(activeIDs) == 0 {
		return Sentence{}, false, nil
	}

	now := time.Now().UTC()
	candidates := make([]Sentence, 0, len(activeIDs))
	for _, id := range activeIDs {
		item, ok, err := s.getSentence(ctx, id)
		if err != nil {
			return Sentence{}, false, err
		}
		if !ok {
			_ = s.redis.SRem(ctx, activeSetKey, id).Err()
			continue
		}
		if now.After(item.ExpiresAt) {
			_ = s.redis.SRem(ctx, activeSetKey, id).Err()
			continue
		}
		if item.Author == userID {
			continue
		}
		candidates = append(candidates, item)
	}

	if len(candidates) == 0 {
		return Sentence{ID: "fallback", Text: random.FallbackSentence(), Fallback: true}, true, nil
	}

	mrand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	seenKey := userSeenKey(userID)
	seenIDs, err := s.redis.SMembers(ctx, seenKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return Sentence{}, false, err
	}
	seen := make(map[string]struct{}, len(seenIDs))
	for _, id := range seenIDs {
		seen[id] = struct{}{}
	}

	chosen := candidates[0]
	if _, exists := seen[chosen.ID]; exists && len(candidates) > 1 {
		second := candidates[1]
		if _, secondExists := seen[second.ID]; secondExists {
			return Sentence{ID: "fallback", Text: random.FallbackSentence(), Fallback: true}, true, nil
		}
		chosen = second
	}

	pipe := s.redis.TxPipeline()
	pipe.SAdd(ctx, seenKey, chosen.ID)
	pipe.Expire(ctx, seenKey, sentenceTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return Sentence{}, false, err
	}

	return chosen, true, nil
}

func (s *Store) react(ctx context.Context, userID, sentenceID, reactionType string) error {
	if !reaction.ValidType(reactionType) {
		return errors.New("invalid reaction type")
	}
	if sentenceID == "" || strings.HasPrefix(sentenceID, "fallback") {
		return nil
	}

	sentence, ok, err := s.getSentence(ctx, sentenceID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if time.Now().UTC().After(sentence.ExpiresAt) {
		return nil
	}

	reactedKey := userReactedKey(userID, sentenceID)
	result, err := s.redis.SetArgs(ctx, reactedKey, reactionType, redis.SetArgs{
		Mode: "NX",
		TTL:  sentenceTTL,
	}).Result()
	if err != nil {
		return err
	}
	if result != "OK" {
		return errors.New("already reacted")
	}

	if err := s.redis.HIncrBy(ctx, sentenceKey(sentenceID), reactionType, 1).Err(); err != nil {
		return err
	}

	return nil
}

func (s *Store) results(ctx context.Context, id string) (Sentence, bool, error) {
	return s.getSentence(ctx, id)
}

func (s *Store) publishedByUser(ctx context.Context, userID string) ([]Sentence, error) {
	ids, err := s.redis.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   userPublishedKey(userID),
		Start: 0,
		Stop:  -1,
	}).Result()
	if err != nil {
		return nil, err
	}

	items := make([]Sentence, 0, len(ids))
	for _, id := range ids {
		sentence, ok, err := s.getSentence(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		items = append(items, sentence)
	}

	return items, nil
}

func (s *Store) readyResultsByUser(ctx context.Context, userID string) ([]Sentence, error) {
	ids, err := s.redis.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   userPublishedKey(userID),
		Start: 0,
		Stop:  -1,
	}).Result()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	items := make([]Sentence, 0, len(ids))
	for _, id := range ids {
		sentence, ok, err := s.getSentence(ctx, id)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if now.Before(sentence.ExpiresAt) {
			continue
		}
		items = append(items, sentence)
	}

	return items, nil
}

func (s *Store) getSentence(ctx context.Context, id string) (Sentence, bool, error) {
	values, err := s.redis.HGetAll(ctx, sentenceKey(id)).Result()
	if err != nil {
		return Sentence{}, false, err
	}
	if len(values) == 0 {
		return Sentence{}, false, nil
	}

	createdUnix, err := parseInt64(values["created_at"])
	if err != nil {
		return Sentence{}, false, err
	}
	expiresUnix, err := parseInt64(values["expires_at"])
	if err != nil {
		return Sentence{}, false, err
	}
	heart, err := parseInt(values["heart"])
	if err != nil {
		return Sentence{}, false, err
	}
	hate, err := parseInt(values["hate"])
	if err != nil {
		return Sentence{}, false, err
	}
	ignoreCount, err := parseInt(values["ignore"])
	if err != nil {
		return Sentence{}, false, err
	}

	item := Sentence{
		ID:        values["id"],
		Author:    values["author"],
		Text:      values["text"],
		CreatedAt: time.Unix(createdUnix, 0).UTC(),
		ExpiresAt: time.Unix(expiresUnix, 0).UTC(),
		Heart:     heart,
		Hate:      hate,
		Ignore:    ignoreCount,
	}
	if item.ID == "" {
		item.ID = id
	}

	return item, true, nil
}

func parseInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func parseInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func sentenceKey(id string) string {
	return "sentence:" + id
}

func userSeenKey(userID string) string {
	return "user:" + userID + ":seen"
}

func userPublishCountKey(userID string) string {
	return "user:" + userID + ":publish_count"
}

func userPublishedKey(userID string) string {
	return "user:" + userID + ":published"
}

func userReactedKey(userID, sentenceID string) string {
	return "user:" + userID + ":reacted:" + sentenceID
}

func ensureUserID(w http.ResponseWriter, r *http.Request) string {
	const cookieName = "from_stranger_uid"

	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}

	id := randomID()
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int((24 * time.Hour).Seconds()),
		HttpOnly: true,
	})

	return id
}

func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

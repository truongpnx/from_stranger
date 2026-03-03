package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"from_stranger/internal/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()
	redisCfg := app.RedisConfigFromEnv()
	redisClient, err := app.ConnectRedis(ctx, redisCfg)
	if err != nil {
		log.Fatalf("redis connection failed (%s:%s): %v", redisCfg.Host, redisCfg.Port, err)
	}
	defer func() {
		_ = redisClient.Close()
	}()
	log.Printf("redis connected at %s:%s", redisCfg.Host, redisCfg.Port)

	router, err := app.NewRouter(redisClient)
	if err != nil {
		log.Fatalf("router init failed: %v", err)
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Printf("server listening on http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}

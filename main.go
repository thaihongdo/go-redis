package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type APIResponse struct {
	Cache bool              `json:"cache"`
	Data  []NominalResponse `json:"data"`
}

type NominalResponse struct {
	PlaceID     int      `json:"place_id"`
	License     string   `json:"license"`
	OsmType     string   `json:"osm_type"`
	OsmID       int      `json:"osm_id"`
	Boundingbox []string `json:"boundingbox"`
	Lat         string   `json:"lat"`
	Long        string   `json:"long"`
	DisplayName string   `json:"display_name"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
	Importance  float64  `json:"importance"`
	Icon        string   `json:"icon"`
}

type API struct {
	cache *redis.Client
}

func NewApi() *API {
	redisAddress := fmt.Sprintf("%s", os.Getenv("REDIS_URL"))

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: "",
		DB:       0,
	})
	return &API{
		cache: rdb,
	}
}

func main() {

	fmt.Println("starting server....")

	api := new(API)
	http.HandleFunc("/api", api.Handler)

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
}

func (a *API) Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In the handler...")

	q := r.URL.Query().Get("q")
	data, err := a.getData(r.Context(), q)
	if err != nil {
		fmt.Printf("Cal api fail... %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	resp := APIResponse{
		Cache: false,
		Data:  data,
	}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		fmt.Printf("Encode data fail...%v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (a *API) getData(ctx context.Context, q string) ([]NominalResponse, error) {
	val, err := a.cache.Get(ctx, q).Result()

	if err == redis.Nil {
		escapeQ := url.PathEscape(q)
		address := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json", escapeQ)

		resp, err := http.Get(address)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		data := make([]NominalResponse, 0)

		jsonData, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("marshal data fail...%v", err)
		}

		err = a.cache.Set(ctx, q, bytes.NewBuffer(jsonData).Bytes(), 5*time.Second).Err()
		if err != nil {
			fmt.Printf("set cache for data fail...%v", err)
		}

		json.NewDecoder(resp.Body).Decode(&data)

		return data, nil
	} else if err != nil {
		fmt.Println("error caching...")
		return nil, err
	} else {
		data := make([]NominalResponse, 0)
		json.Unmarshal(bytes.NewBufferString(val).Bytes(), &data)

		return data, nil
	}
}

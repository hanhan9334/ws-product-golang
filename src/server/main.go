package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type counters struct {
	sync.Mutex
	View  int `json:"view"`
	Click int `json:"click"`
}

//Context var for Redis
var ctx = context.Background()

var (
	c = counters{}

	content = []string{"sports", "entertainment", "business", "education"}
)

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	data := content[rand.Intn(len(content))]

	c.Lock()
	c.View++
	c.Unlock()

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(data)
	}
	uploadCounters(data, &c)

}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	c.Lock()
	c.Click++
	c.Unlock()

	return nil
}

func isInMinute(startTime time.Time, currentTime time.Time) bool {
	elapsed := currentTime.Sub(startTime)
	if elapsed < 60*time.Second {
		return true
	} else {
		return false
	}
}
func stringToTime(s string) time.Time {
	sec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(sec, 0)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {

	_user := r.Header.Get("user")

	currentTime := time.Now().Unix()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	val, err := rdb.Get(ctx, _user).Result()
	if err == redis.Nil {
		data := strconv.Itoa(int(currentTime))
		data += ",1"
		set, err := rdb.Set(ctx, _user, data, 1*time.Minute).Result()
		if err != nil {
			panic(err)
		}
		_ = set
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println(val)
		windowStartTime := stringToTime((strings.Split(val, ","))[0])
		calls, err := strconv.Atoi(strings.Split(val, ",")[1])
		_ = err

		fmt.Println(calls)
		if calls >= 5 {
			fmt.Fprint(w, "Too much request")
			w.WriteHeader(429)
			return
		} else {
			calls++

			value := (strings.Split(val, ","))[0] + "," + strconv.Itoa(calls)
			timeToExpire := int(currentTime) - int(windowStartTime.Unix())
			set, err := rdb.Set(ctx, _user, value, time.Duration(timeToExpire)).Result()
			if err != nil {
				panic(err)
			}
			_ = set

		}
	}
	fmt.Fprint(w, "Request sent")

}

func isAllowed() bool {
	return true
}

func uploadCounters(data string, c *counters) {

	current := time.Now()

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       1,
	})

	key := data + ":" + current.Format("2006-01-02 15:04:05")

	json, err := json.Marshal(counters{View: c.View, Click: c.Click})
	if err != nil {
		fmt.Println(err)
	}

	set, err := rdb.Set(ctx, key, json, 0).Result()
	if err != nil {
		panic(err)

	}
	_ = set

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println(key, val)

	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				//I tried to retrive all keys in Redis, store them into mock store and clear Redis cache.
				//But 
				//1. I counldn't debug it. 
				//2. I doubted that this goroutine should be declared in the main function.
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

}

func main() {
	http.HandleFunc("/", welcomeHandler)
	http.HandleFunc("/view/", viewHandler)
	http.HandleFunc("/stats/", statsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}

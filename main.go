package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

const INVALID_REQUEST = "invalid command"
const TRY_AGAIN = "try again"
const SUCCESSFUL = "command successfully executed"

var myCache ICache

type ICache interface {
	Set(input *setInput) error
	Get(key string) ([]byte, error)
	Push(key, value string)
	Pop(key string) (string, error)
}

type redisCache struct {
	client *redis.Client
}

type setInput struct {
	key       string
	value     string
	expiry    int64
	condition string
}

func (r *redisCache) Set(input *setInput) error {
	if condition := input.condition; condition == "NX" {
		return r.client.SetNX(input.key,
			input.value,
			time.Duration(time.Duration(input.expiry).Seconds()),
		).Err()
	} else if condition == "XX" {
		return r.client.SetXX(input.key,
			input.value,
			time.Duration(time.Duration(input.expiry).Seconds()),
		).Err()
	}
	return r.client.Set(input.key,
		input.value,
		time.Duration(time.Duration(input.expiry).Seconds()),
	).Err()
}

func (r *redisCache) Get(key string) ([]byte, error) {
	result, err := r.client.Get(key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}

	return result, err
}

func (r *redisCache) Push(key, value string) {
	stringInputs := strings.Split(value, " ")
	var inputs []interface{}
	for _, input := range stringInputs {
		inputs = append(inputs, input)
	}
	r.client.RPush(key, inputs...)
}

func (r *redisCache) Pop(key string) (string, error) {
	return r.client.RPop(key).Val(), r.client.RPop(key).Err()
}

func InitRedis() {
	myCache = &redisCache{
		client: redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "",
			DB:       0,
		}),
	}
}

func setValue(c *gin.Context) {
	req := c.Request
	if err := req.ParseForm(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": TRY_AGAIN,
		})
		return
	} else {
		var tim int64
		var err error
		if req.FormValue("expiry") == "" {
			tim = 0
			err = nil
		} else {
			tim, err = strconv.ParseInt(req.FormValue("expiry"), 10, 64)
		}

		if err != nil || tim < 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"err": INVALID_REQUEST,
			})
			return
		}
		log.Printf("key is %s\n", req.FormValue("key"))
		log.Printf("value is %s\n", req.FormValue("value"))
		log.Printf("expiry time is %d\n", tim)
		log.Printf("condition is %s\n", req.FormValue("condition"))

		if err := myCache.Set(&setInput{
			key:       req.FormValue("key"),
			value:     req.FormValue("value"),
			expiry:    tim,
			condition: req.FormValue("condition"),
		}); err != nil {
			log.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"err": TRY_AGAIN,
			})
			return
		}
		log.Println("successful")
		c.JSON(http.StatusOK, gin.H{
			"value": SUCCESSFUL,
		})
	}
}

func getValue(c *gin.Context) {
	req := c.Request

	if err := req.ParseForm(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": TRY_AGAIN,
		})
		return
	}

	key := req.FormValue("key")
	value, err := myCache.Get(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": TRY_AGAIN,
		})
		return
	}
	if value == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": "Key not found",
		})
		return
	}

	log.Println("successful")
	c.JSON(http.StatusOK, gin.H{
		"value": string(value),
	})

}

func qpush(c *gin.Context) {
	req := c.Request

	if err := req.ParseForm(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": TRY_AGAIN,
		})
		return
	}

	key := req.FormValue("key")
	myCache.Push(key, req.FormValue("value"))
	c.JSON(http.StatusOK, gin.H{
		"value": SUCCESSFUL,
	})
}

func qpop(c *gin.Context) {
	req := c.Request

	if err := req.ParseForm(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": TRY_AGAIN,
		})
		return
	}

	key := req.FormValue("key")
	result, err := myCache.Pop(key)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"err": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"value": result,
	})
}

func main() {

	InitRedis()

	router := gin.Default()

	router.StaticFile("/", "./static/homePage.html")

	router.StaticFile("/set", "./static/setForm.html")
	router.POST("/set", func(c *gin.Context) {
		setValue(c)
	})

	router.StaticFile("/get", "./static/getForm.html")
	router.POST("/get", func(c *gin.Context) {
		getValue(c)
	})

	router.StaticFile("/qpush", "./static/qpushForm.html")
	router.POST("/qpush", func(c *gin.Context) {
		qpush(c)
	})

	router.StaticFile("/qpop", "./static/qpopForm.html")
	router.POST("/qpop", func(c *gin.Context) {
		qpop(c)
	})

	router.Run(":3000")
}

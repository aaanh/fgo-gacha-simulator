package main

import (
	"database/sql"
	_ "fmt"
	"math/rand"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gofor-little/env"
	_ "github.com/mattn/go-sqlite3"
)

const SSR_RATE = 0.01
const SR_RATE = 0.07
const OTHER_RATE = 0.2

// Return list of Servants in JSON format
type Servant struct {
	CollectionNo int    `json:"collectionNo"`
	OriginalName string `json:"originalName"`
	Name         string `json:"name"`
	Rarity       int    `json:"rarity"`
	ClassName    string `json:"className"`
	AtkMax       int    `json:"atkMax"`
	HpMax        int    `json:"hpMax"`
	Attribute    string `json:"attribute"`
	Face         string `json:"face"`
	FacePath     string `json:"face_path"`
}

func DoSingleRoll(servants []Servant) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		roll := rand.Intn(100) + 1

		if roll == 1 {
			var filtered []Servant

			for _, sv := range servants {
				if sv.Rarity == 5 {
					filtered = append(filtered, sv)
				}
			}

			local_roll := rand.Intn(len(filtered))
			body := map[string]Servant{"roll": filtered[local_roll]}
			c.JSON(http.StatusOK, body)
		} else if roll > 1 && roll <= 4 {
			var filtered []Servant

			for _, sv := range servants {
				if sv.Rarity == 4 {
					filtered = append(filtered, sv)
				}
			}

			local_roll := rand.Intn(len(filtered))
			body := map[string]Servant{"roll": filtered[local_roll]}
			c.JSON(http.StatusOK, body)
		} else {
			var filtered []Servant

			for _, sv := range servants {
				if sv.Rarity <= 3 {
					filtered = append(filtered, sv)
				}
			}

			local_roll := rand.Intn(len(filtered))
			body := map[string]Servant{"roll": filtered[local_roll]}
			c.JSON(http.StatusOK, body)
		}
	}

	return gin.HandlerFunc(fn)
}

func DoMultiRoll(servants []Servant) gin.HandlerFunc {
	var guaranteed []Servant

	for _, sv := range servants {
		if sv.Rarity >= 4 {
			guaranteed = append(guaranteed, sv)
		}
	}

	fn := func(c *gin.Context) {
		var results []Servant
		for i := 0; i < 11; i++ {
			if i == 0 {
				local_roll := rand.Intn(len(guaranteed))
				results = append(results, guaranteed[local_roll])
			} else {
				roll := rand.Intn(100) + 1

				if roll == 1 {
					var filtered []Servant

					for _, sv := range servants {
						if sv.Rarity == 5 {
							filtered = append(filtered, sv)
						}
					}

					local_roll := rand.Intn(len(filtered))
					results = append(results, filtered[local_roll])
				} else if roll > 1 && roll <= 4 {
					var filtered []Servant

					for _, sv := range servants {
						if sv.Rarity == 4 {
							filtered = append(filtered, sv)
						}
					}

					local_roll := rand.Intn(len(filtered))
					results = append(results, filtered[local_roll])

				} else {
					var filtered []Servant

					for _, sv := range servants {
						if sv.Rarity <= 3 {
							filtered = append(filtered, sv)
						}
					}

					local_roll := rand.Intn(len(filtered))
					results = append(results, filtered[local_roll])

				}
			}
		}

		body := map[string][]Servant{"rolls": results}
		c.JSON(http.StatusOK, body)
	}

	return gin.HandlerFunc(fn)
}

func GetAllServants(servants []Servant) gin.HandlerFunc {
	fn := func(c *gin.Context) {
		body := map[string][]Servant{"servants": servants}
		c.JSON(http.StatusOK, body)
	}

	return gin.HandlerFunc(fn)
}

func GetServantByCollectionNo(db *sql.DB) gin.HandlerFunc {
	var sv Servant

	fn := func(c *gin.Context) {
		collectionNo := c.Param("collectionNo")
		res, err := db.Query("SELECT * FROM servants WHERE collectionNo = ?", collectionNo)

		// Handle query error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
			return
		}

		defer res.Close()

		if res.Next() {
			if err := res.Scan(&sv.CollectionNo, &sv.OriginalName, &sv.Name, &sv.Rarity, &sv.ClassName, &sv.AtkMax, &sv.HpMax, &sv.Attribute, &sv.Face, &sv.FacePath); err != nil {
				// Handle the error
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
				return
			}
		} else {
			// Handle the case where no row was found
			c.JSON(http.StatusNotFound, gin.H{"error": "Servant not found"})
			return
		}

		body := map[string]Servant{"servant": sv}
		c.JSON(http.StatusOK, body)
	}

	return gin.HandlerFunc(fn)
}

func main() {
	env.Load("./.env")

	SERVER_MODE := env.Get("SERVER_MODE", "debug")
	DATABASE_PATH := env.Get("DATABASE_PATH", "../database/sv_db.db")

	if SERVER_MODE == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		DATABASE_PATH = "../database/sv_db.db"
		gin.SetMode(gin.DebugMode)
	}

	// Connect to database
	db, _ := sql.Open("sqlite3", DATABASE_PATH)
	defer db.Close()
	rows, _ := db.Query("SELECT * FROM servants")
	defer rows.Close()

	var servants []Servant

	for rows.Next() {
		var sv Servant
		rows.Scan(&sv.CollectionNo, &sv.OriginalName, &sv.Name, &sv.Rarity, &sv.ClassName, &sv.AtkMax, &sv.HpMax, &sv.Attribute, &sv.Face, &sv.FacePath)
		// fmt.Print(servant)
		servants = append(servants, sv)
	}

	// Create router
	router := gin.Default()

	// CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000/servants", "http://localhost:3000", "https://fgo.aaanh.app", "https://fgo.aaanh.app/servants"} // You may want to restrict this to specific origins in production
	config.AllowMethods = []string{"GET"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization"}

	router.Use(cors.New(config))

	// Routes

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Roll actions
	router.GET("/roll/single", DoSingleRoll(servants))
	router.GET("/roll/multi", DoMultiRoll(servants))

	// Servant lookup
	router.GET("/servants/:collectionNo", GetServantByCollectionNo(db))
	router.GET("/servants", GetAllServants(servants))
	router.GET("/stats/total_servants", func(c *gin.Context) {
		c.JSON(http.StatusOK, len(servants))
	})

	// Start the server on port 8080
	router.Run(":8080")
}

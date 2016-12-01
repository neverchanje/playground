package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"gopkg.in/redis.v5"
)

const baseChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func generateRandomString(rnd *rand.Rand) string {
	str := make([]byte, 20)
	for i := 0; i < len(str); i++ {
		str[i] = baseChars[rnd.Intn(len(baseChars))]
	}
	return string(str)
}

var client *redis.Client

func insertRecords(num int, rnd *rand.Rand) {
	doForRecords(num, func(i int) {
		err := client.Set(generateRandomString(rnd),
			generateRandomString(rnd), 0).Err()
		if err != nil {
			panic(err)
		}
	})
}

func insertAndUpdateRecords(num int, rnd *rand.Rand) {
	doForRecords(num, func(i int) {
		err := client.Set(string(i%100), generateRandomString(rnd), 0).Err()
		if err != nil {
			panic(err)
		}
	})
}

func doForRecords(num int, op func(int)) {
	startTime := time.Now().UnixNano()
	k := 1
	for i := 0; i < num; i++ {
		if i > k*num/10 {
			fmt.Printf("%d percents completed\n", k*10)
			k++
		}
		op(i)
	}
	endedTime := time.Now().UnixNano()
	fmt.Printf("100 percents completed, costs %f seconds\n",
		time.Duration(endedTime-startTime).Seconds())
}

func main() {
	client = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:7999",
	})

	insertNum := flag.Int("insert", 0, "")
	insertAndUpdateNum := flag.Int("update", 0, "")
	clean := flag.Bool("clean", false, "")
	flag.Parse()

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	if *clean {
		err := client.FlushAll().Err()
		if err != nil {
			panic(err)
		}

		err = client.BgRewriteAOF().Err()
		if err != nil {
			panic(err)
		}
	}

	if *insertNum > 0 {
		insertRecords(*insertNum, rnd)
	} else if *insertAndUpdateNum > 0 {
		insertAndUpdateRecords(*insertAndUpdateNum, rnd)
	}
}

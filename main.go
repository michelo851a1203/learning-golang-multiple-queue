package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	maxThread      = 10 // 這個數量就是要開啟的 goroutine 最大數量，想要效能更好可以開越多，但相對記憶體消耗也越大
	input1ChanList = make([]chan string, maxThread)
	input2ChanList = make([]chan string, maxThread)
	outputChanList = make([]chan string, maxThread)
)

func CurrentServer(ctx *gin.Context) {
	dataCount := 10000 // 假資料，這個是進來時的資料筆數有多少
	inputList := make([]struct{}, dataCount)

	outputWaitGroup := sync.WaitGroup{}
	outputWaitGroup.Add(dataCount)

	currentCtx, cancel := context.WithCancel(context.Background())

	for index := range outputChanList {
		go func(wg *sync.WaitGroup, i int) {
			for {
				select {
				case result := <-outputChanList[i]:
					fmt.Println(result)
					// 提示處1*
					wg.Done()
				case <-currentCtx.Done():
					return
				default:
					time.Sleep(time.Millisecond)
				}
			}
		}(&outputWaitGroup, index)
	}

	go func() {
		for inputIndex := range inputList {
			allocatedThreadNumber := inputIndex % maxThread
			input1ChanList[allocatedThreadNumber] <- fmt.Sprintf("[input] - %d", inputIndex)
		}
	}()

	go func() {
		outputWaitGroup.Wait()
		cancel()
		fmt.Println("執行完畢...")
	}()

	// 如果這裡要用 ＳＳＥ 那就不用上面的 go routine 了
	// outputWaitGroup.Wait()
	// cancel()

	// 這裏未必要用 restful 可以在上面(提示處1*) 改成用 SSE 丟狀態
	ctx.JSON(http.StatusOK, gin.H{
		"message": "執行程序中",
	})
}

func ProcessOne(inputString string) string {
	// 隨機 sleep 0~2秒內的時間 假裝為第一次打 openai api 和處理字串的時間 如果要抄這裡記得改承你的邏輯
	delayTime := rand.Intn(2000) + 1
	time.Sleep(time.Duration(delayTime) * time.Millisecond)
	return fmt.Sprintf("%s ->第一階段時間 [%f] 秒,", inputString, float64(delayTime)/1000)
}

func ProcessTwo(inputString string) string {
	// 隨機 sleep 0~2秒內的時間 假裝為第二次打 openai api 和處理字串的時間 如果要抄這裡記得改承你的邏輯
	delayTime := rand.Intn(2000) + 1
	time.Sleep(time.Duration(delayTime) * time.Millisecond)
	return fmt.Sprintf("%s ->第二階段時間 [%f] 秒", inputString, float64(delayTime)/1000)
}

func main() {
	router := gin.Default()
	router.SetTrustedProxies(nil)

	router.GET("/", CurrentServer)

	for index := range input1ChanList {
		input1ChanList[index] = make(chan string)
		input2ChanList[index] = make(chan string)
		outputChanList[index] = make(chan string)

		// 這裡可以用 buffer channel，會讓你更快，但理論上記憶體會消耗更多，可以把上面那三行用這個試試
		// 注意：100 我只是隨便給的，給更大， queue 可容納的空間越大，也因此處理速度比較快，但相對消耗記憶體
		input1ChanList[index] = make(chan string, 100)
		input2ChanList[index] = make(chan string, 100)
		outputChanList[index] = make(chan string, 100)
	}

	for index := range input1ChanList {
		go func(in *chan string, i int) {
			for {
				select {
				case inputResult := <-*in:
					input2ChanList[i] <- ProcessOne(inputResult)
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(&input1ChanList[index], index)
	}

	for index := range input2ChanList {
		go func(in *chan string, i int) {
			for {
				select {
				case inputResult := <-*in:
					outputChanList[i] <- ProcessTwo(inputResult)
				default:
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(&input2ChanList[index], index)
	}

	router.Run(":8080")
}

package services

import (
	"context"
	"fmt"
)

func QueueJob(jobType, data string) {
	redisClient.LPush(context.Background(), "jobs", fmt.Sprintf("%s:%s", jobType, data))
}

func ProcessJobs() {
	for {
		job, _ := redisClient.BRPop(context.Background(), 0, "jobs").Result()
		if len(job) > 1 {
			fmt.Println("Processing job:", job[1]) //Simulate email or log
		}
	}
}

package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func ReadPodLogs(client *kubernetes.Clientset, namespace string, containerName string, follow *bool, sinceTime *time.Time, logLimit *int64) string {
	pod := GetPodContainingContainer(client, namespace, containerName)
	if pod != nil {
		podLogOptions := GetDefaultPodLogOptions()
		if logLimit != nil {
			podLogOptions.LimitBytes = logLimit
		}
		if follow != nil {
			podLogOptions.Follow = *follow
		}
		if sinceTime != nil {
			timeSince := metav1.NewTime(sinceTime.UTC())
			podLogOptions.SinceTime = &timeSince
		}
		podLogsRequest := client.CoreV1().Pods(namespace).GetLogs(pod.Name, podLogOptions)
		podLogs, err := podLogsRequest.Stream(context.Background()) // Pass a context for cancellation
		if err != nil {
			logger.Log("Failed to stream pod logs due to %v", err)
		}
		defer func(podLogs io.ReadCloser) {
			err = podLogs.Close()
			if err != nil {
				logger.Log("Failed to close pod log reader due to %v", err)
			}
		}(podLogs)
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			logger.Log("Failed to add pod logs to buffer due to: %v", err)
		}
		return buf.String()
	} else {
		return fmt.Sprintf("Could not find pod containing container with name '%s'", containerName)
	}
}

func GetDefaultPodLogOptions() *v1.PodLogOptions {
	logLimit := int64(50000000)
	sinceTime := metav1.NewTime(time.Now().Add(-1 * time.Minute).UTC())
	return &v1.PodLogOptions{
		Previous:   false,
		SinceTime:  &sinceTime,
		Timestamps: true,
		LimitBytes: &logLimit,
		Follow:     false,
	}
}

func GetPodContainingContainer(client *kubernetes.Clientset, namespace, containerName string) *v1.Pod {
	pods, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Log("Failed to list pods due to: %v", err)
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Name, containerName) {
				return &pod
			}
		}
	}
	return nil
}

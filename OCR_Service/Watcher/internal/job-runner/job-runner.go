package jobrunner

import (
	"context"
	"log/slog"
	"ocr/packages/queue"
	"os"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
var jobNamePrefix = "ocr-job-"

type JobRunner struct {
	clientSet         *kubernetes.Clientset
	rmqAddress        string
	publisherExchange string
	dockerImage       string
}

func NewJobRunner(kubeconfig *rest.Config, rmqAddress string, publisherexchange string, image string) (*JobRunner, error) {
	clientSet, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		logger.Error("Failed to create Kubernetes clientset", "error", err)
		return &JobRunner{}, err
	}
	return &JobRunner{clientSet: clientSet, rmqAddress: rmqAddress, publisherExchange: publisherexchange, dockerImage: image}, nil
}

func (j *JobRunner) CreateJob(ocrData queue.RmqMessage) error {
	var backoffLimit int32 = 2
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: jobNamePrefix,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "ocr-worker",
							Image: j.dockerImage,
							Env: []corev1.EnvVar{
								{
									Name:  "DOWNLOAD_URL",
									Value: ocrData.Download_link,
								},
								{
									Name:  "UPLOAD_URL",
									Value: ocrData.Upload_link,
								},
								{
									Name:  "JOBID",
									Value: ocrData.JobID,
								},
								{
									Name:  "DESCRIPTION",
									Value: ocrData.Description,
								},
								{
									Name:  "RABBITMQ_URL",
									Value: j.rmqAddress,
								},
								{
									Name:  "RABBITMQ_PUBLISHER_EXCHANGE",
									Value: j.publisherExchange,
								},
							},
						},
					},
				},
			},
		},
	}

	jobsClient := j.clientSet.BatchV1().Jobs("ocr-worker")
	result, err := jobsClient.Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		logger.Error("Failed to create job", "error", err)
		return err
	}
	logger.Info("Worker job was successfully created", "jobName", result.Name)
	return nil
}

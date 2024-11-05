package k8s

import (
	apps_v1 "k8s.io/api/apps/v1"
	batch_v1 "k8s.io/api/batch/v1"
	core_v1 "k8s.io/api/core/v1"
)

func getContainerImages(containers []core_v1.Container, filter ContainerFilter) []string {
	var images []string
	for _, c := range containers {
		if filter == nil || filter(c) {
			images = append(images, c.Image)
		}
	}

	return images
}

func getImagePullSecrets(imagePullSecrets []core_v1.LocalObjectReference) []string {
	var secrets []string
	for _, s := range imagePullSecrets {
		secrets = append(secrets, s.Name)
	}
	return secrets
}

// deployments

func getDeploymentIdentifier(d *apps_v1.Deployment) string {
	return "deployment/" + d.Namespace + "/" + d.Name
}

func updateDeploymentContainer(d *apps_v1.Deployment, index int, image string) {
	d.Spec.Template.Spec.Containers[index].Image = image
}

func updateDeploymentInitContainer(d *apps_v1.Deployment, index int, image string) {
	d.Spec.Template.Spec.InitContainers[index].Image = image
}

// stateful sets https://kubernetes.io/docs/tutorials/stateful-application/basic-stateful-set/

func getStatefulSetIdentifier(ss *apps_v1.StatefulSet) string {
	return "statefulset/" + ss.Namespace + "/" + ss.Name
}

func updateStatefulSetContainer(ss *apps_v1.StatefulSet, index int, image string) {
	ss.Spec.Template.Spec.Containers[index].Image = image
}

func updateStatefulSetInitContainer(ss *apps_v1.StatefulSet, index int, image string) {
	ss.Spec.Template.Spec.InitContainers[index].Image = image
}

// daemonsets

func getDaemonsetSetIdentifier(s *apps_v1.DaemonSet) string {
	return "daemonset/" + s.Namespace + "/" + s.Name
}

func updateDaemonsetSetContainer(s *apps_v1.DaemonSet, index int, image string) {
	s.Spec.Template.Spec.Containers[index].Image = image
}

func updateDaemonsetSetInitContainer(s *apps_v1.DaemonSet, index int, image string) {
	s.Spec.Template.Spec.InitContainers[index].Image = image
}

// cron

func getCronJobIdentifier(s *batch_v1.CronJob) string {
	return "cronjob/" + s.Namespace + "/" + s.Name
}

func updateCronJobContainer(s *batch_v1.CronJob, index int, image string) {
	s.Spec.JobTemplate.Spec.Template.Spec.Containers[index].Image = image
}

func updateCronJobInitContainer(s *batch_v1.CronJob, index int, image string) {
	s.Spec.JobTemplate.Spec.Template.Spec.InitContainers[index].Image = image
}

package mutation

import (
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	CaBundleVolume = "ca-bundle-volume"
	CaBundleCM     = "omega-bundle"
	CaMountPath    = "/etc/ssl/certs/ca-certificates.crt"
	CaSubPath      = "ca-certificates.crt"
	SSLCertEnvVar  = "SSL_CERT_FILE"
)

type InjectCA struct {
	Logger logrus.FieldLogger
}

var _ PodMutator = &InjectCA{}

func (ca InjectCA) Name() string {
	return "inject-ca-bundle"
}

func (ca InjectCA) Mutate(pod *corev1.Pod) (*corev1.Pod, error) {
	// Implement the logic to mutate the Pod
	ca.Logger = ca.Logger.WithField("mutate", ca.Name())
	ca.Logger.Info("Mutating pod...", pod.Namespace, pod.Name)

	mpod := pod.DeepCopy()
	if err := ca.InjectCAVolume(mpod); err != nil {
		return nil, err
	}
	if err := ca.InjectVolumeMount(mpod); err != nil {
		return nil, err
	}
	if err := ca.InjectEnv(mpod); err != nil {
		return nil, err
	}

	return mpod, nil
}

// CheckPodVolume checks if the pod has the volume and csi driver
func (ca InjectCA) CheckPodVolume(pod *corev1.Pod) (bool, bool) {
	ca.Logger.Info("Checking pod volumes:", pod.Namespace, pod.Name)
	VolNameExists := false
	ConfigMapExists := false
	for _, volume := range pod.Spec.Volumes {
		ca.Logger.Info("Checking if volume is ca-bundle-volume:", pod.Name)
		if volume.Name == CaBundleVolume {
			ca.Logger.Info("ca-bundle-volume name exists:", volume.Name)
			VolNameExists = true
		}
		ca.Logger.Info("Checking if volume is ConfigMap:", pod.Name)
		if volume.ConfigMap != nil {
			if volume.ConfigMap.Name == CaBundleCM {
				ca.Logger.Info("ConfigMap exists:", volume.Name)
				ConfigMapExists = true
			}
		}
	}
	return VolNameExists, ConfigMapExists
}

func (ca InjectCA) CheckContainerVolumeMount(container corev1.Container) bool {
	var caMountExists bool
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name != CaBundleVolume {
			continue
		}
		if volumeMount.MountPath == CaMountPath && volumeMount.SubPath == CaSubPath {
			caMountExists = true
			break
		}
	}
	return caMountExists
}

func (ca *InjectCA) InjectCAVolume(mpod *corev1.Pod) error {
	CAConfigMapVolume := corev1.Volume{
		Name: CaBundleVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: CaBundleCM,
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "root-certs.pem",
						Path: "ca-certificates.crt",
					},
				},
			},
		},
	}

	// Add the volume to the pod
	caVolExists, ConfigMapExists := ca.CheckPodVolume(mpod)
	ca.Logger.Info("caVolExists:", caVolExists, "ConfigMapExists:", ConfigMapExists)
	if !caVolExists && !ConfigMapExists {
		ca.Logger.Info("Injecting ca-bundle volume to pod:", mpod.Name)
		mpod.Spec.Volumes = append(mpod.Spec.Volumes, CAConfigMapVolume)
	}

	if caVolExists && !ConfigMapExists {
		ca.Logger.Info("caVol exists but ConfigMap does not exist")
		var updatedVolumes []corev1.Volume
		//ca.Logger.Debug("Updating ca-bundle volume in pod", mpod.Name, mpod.Namespace)
		for _, volume := range mpod.Spec.Volumes {

			if volume.Name != CaBundleVolume {
				updatedVolumes = append(updatedVolumes, volume)
			}
		}
		updatedVolumes = append(updatedVolumes, CAConfigMapVolume)
		mpod.Spec.Volumes = updatedVolumes
	} else {
		ca.Logger.Info("DID Not meet the condition caVolExists && !ConfigMapExists")
	}
	return nil
}

func (ca InjectCA) InjectVolumeMount(mpod *corev1.Pod) error {
	// Define the volume mount
	CAMount := corev1.VolumeMount{
		Name:      CaBundleVolume,
		MountPath: CaMountPath,
		SubPath:   CaSubPath,
		ReadOnly:  true,
	}

	//Inject the volume mount to all init-containers

	if mpod.Spec.InitContainers != nil {
		ca.Logger.Info("Injecting volume to init containers")
		for i := range mpod.Spec.InitContainers {
			initContainer := &mpod.Spec.InitContainers[i]
			//sc.Logger.Info("Checking volume in init container:", initContainer.Name)
			caMountExists := ca.CheckContainerVolumeMount(*initContainer)
			if !caMountExists {
				ca.Logger.Info("Injecting ca-bundle volume to init container:", initContainer.Name)
				initContainer.VolumeMounts = append(initContainer.VolumeMounts, CAMount)
			}
		}
	}

	//Inject the volume mount to all
	ca.Logger.Info("Injecting volume to containers")
	for i := range mpod.Spec.Containers {
		container := &mpod.Spec.Containers[i]
		ca.Logger.Info("Checking volume in container:", container.Name)
		caMountExists := ca.CheckContainerVolumeMount(*container)
		if !caMountExists {
			ca.Logger.Info("Injecting ca-bundle volume to container:", container.Name)
			container.VolumeMounts = append(container.VolumeMounts, CAMount)
		}
		ca.Logger.Info("container volume mounts:", container.VolumeMounts)
	}
	return nil
}

func (ca InjectCA) InjectEnv(mpod *corev1.Pod) error {
	if mpod.Spec.InitContainers != nil {
		if err := ca.CheckEnvVar(mpod.Spec.InitContainers); err != nil {
			return err
		}
	}
	if err := ca.CheckEnvVar(mpod.Spec.Containers); err != nil {
		return err
	}
	return nil
}

func (ca InjectCA) CheckEnvVar(containers []corev1.Container) error {
	for i := range containers {
		if containers[i].Env == nil {
			containers[i].Env = []corev1.EnvVar{
				{
					Name:  SSLCertEnvVar,
					Value: CaMountPath,
				},
			}
		} else {
			for j, envVar := range containers[i].Env {
				if envVar.Name == SSLCertEnvVar {
					if envVar.Value != CaMountPath {
						containers[i].Env[j].Value = CaMountPath
					}
					break
				}
			}
			containers[i].Env = append(containers[i].Env, corev1.EnvVar{
				Name:  SSLCertEnvVar,
				Value: CaMountPath,
			})
		}
	}
	return nil
}

package mutation

import (
	"context"
	"encoding/json"
	"fmt"

	"ca-bundle-injector/localtypes"

	"github.com/sirupsen/logrus"
	"github.com/wI2L/jsondiff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Mutator struct {
	Logger   *logrus.Entry
	BundleCm *localtypes.BundleConfigMapSpec
}

func NewMutator(logger *logrus.Entry) *Mutator {
	return &Mutator{Logger: logger}
}

type PodMutator interface {
	Mutate(*corev1.Pod) (*corev1.Pod, error)
	Name() string
}

func (m *Mutator) MutatePodPatch(pod *corev1.Pod) ([]byte, error) {
	// Implement the logic to mutate the Pod
	ctx := context.Background()

	var podName string
	if pod.ObjectMeta.Name != "" {
		podName = pod.ObjectMeta.Name
	} else {
		if pod.ObjectMeta.GenerateName != "" {
			podName = pod.ObjectMeta.GenerateName
		}
	}
	log := logrus.WithField("pod", podName)

	// Check if the ConfigMap Spec is valid in the Pod's namespace
	if err := m.checkConfigMapSpec(ctx, pod); err != nil {
		return nil, err
	}

	mutations := []PodMutator{
		InjectCA{Logger: log, BundleCm: m.BundleCm},
	}

	mpod := pod.DeepCopy()

	for _, mutation := range mutations {
		var err error
		mpod, err = mutation.Mutate(mpod)
		if err != nil {
			return nil, err
		}
	}

	//Create the patch diff
	patch, err := jsondiff.Compare(pod, mpod)
	if err != nil {
		return nil, err
	}

	//Patch bytes
	patchb, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	// return patch bytes
	return patchb, nil
}

func (m *Mutator) checkConfigMapSpec(ctx context.Context, pod *corev1.Pod) error {
	client, err := getClientset()
	if err != nil {
		return err
	}
	var cm *corev1.ConfigMap
	cm, err = client.CoreV1().ConfigMaps(pod.Namespace).Get(
		ctx,
		m.BundleCm.Name,
		metav1.GetOptions{},
	)
	if err != nil {
		m.Logger.Errorf("Failed to get ConfigMap %s: %v", m.BundleCm.Name, err)
		return err
	}

	for k := range cm.Data {
		m.Logger.Infof("DEBUG: key: %s", k)
		if k == m.BundleCm.Key {
			m.Logger.Infof("Successfully found key %s in ConfigMap %s", m.BundleCm.Key, m.BundleCm.Name)
			return nil
		}
	}
	m.Logger.Errorf("Key %s not found in ConfigMap %s", m.BundleCm.Key, m.BundleCm.Name)
	return fmt.Errorf("key %s not found in ConfigMap %s", m.BundleCm.Key, m.BundleCm.Name)
}

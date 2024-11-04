package k8s

import (
	"github.com/sirupsen/logrus"
)

type Translator struct {
	logrus.FieldLogger

	GenericResourceCache

	KeelSelector string
}

func (t *Translator) OnAdd(obj interface{}, isInInitialList bool) {
	gr, err := NewGenericResource(obj)
	if err != nil {
		t.Errorf("OnAdd failed to add resource %T: %#v", obj, obj)
		return
	}
	t.Debugf("added %s %s", gr.Kind(), gr.Name)
	t.GenericResourceCache.Add(gr)
}

func (t *Translator) OnUpdate(oldObj, newObj interface{}) {
	gr, err := NewGenericResource(newObj)
	if err != nil {
		t.Errorf("OnUpdate failed to update resource %T: %#v", newObj, newObj)
		return
	}
	t.Debugf("updated %s %s", gr.Kind(), gr.Name)
	t.GenericResourceCache.Add(gr)
}

func (t *Translator) OnDelete(obj interface{}) {
	gr, err := NewGenericResource(obj)
	if err != nil {
		t.Errorf("OnDelete failed to delete resource %T: %#v", obj, obj)
		return
	}
	t.Debugf("deleted %s %s", gr.Kind(), gr.Name)
	t.GenericResourceCache.Remove(gr.GetIdentifier())
}

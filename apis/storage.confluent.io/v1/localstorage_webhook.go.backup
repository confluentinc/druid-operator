/*

 */

package v1

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var localstoragelog = logf.Log.WithName("localstorage-resource")

func (r *LocalStorage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-storage-confluent-io-apache-org-v1-localstorage,mutating=true,failurePolicy=fail,groups=storage.confluent.io.apache.org,resources=localstorages,verbs=create;update,versions=v1,name=mlocalstorage.kb.io

var _ webhook.Defaulter = &LocalStorage{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LocalStorage) Default() {
	localstoragelog.Info("default", "name", r.Name)
	if r.Spec.ServiceAccountName == "" {
		r.Spec.ServiceAccountName = "druid-operator"
	}

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:verbs=create;update,path=/validate-storage-confluent-io-apache-org-v1-localstorage,mutating=false,failurePolicy=fail,groups=storage.confluent.io.apache.org,resources=localstorages,versions=v1,name=vlocalstorage.kb.io

var _ webhook.Validator = &LocalStorage{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LocalStorage) ValidateCreate() error {
	localstoragelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LocalStorage) ValidateUpdate(old runtime.Object) error {
	localstoragelog.Info("validate update", "name", r.Name)
	oldLS, ok := old.(*LocalStorage)
	if !ok {
		return errors.New("something wrong with updation of the object")
	}
	if oldLS.Spec.InstanceType != r.Spec.InstanceType {
		return errors.New("you cannot change the instance type at ths moment")
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LocalStorage) ValidateDelete() error {
	localstoragelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

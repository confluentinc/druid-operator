/*

 */

package controllers

import (
	"context"
	"os"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageconfluentiov1 "github.com/datainfrahq/druid-operator/apis/storage.confluent.io/v1"
)

// LocalStorageReconciler reconciles a LocalStorage object
type LocalStorageReconciler struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	ReconcileWait time.Duration
}

func NewLocalStorageReconciler(mgr ctrl.Manager) *LocalStorageReconciler {
	return &LocalStorageReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("LocalStorage"),
		Scheme:        mgr.GetScheme(),
		ReconcileWait: LookupReconcileTime(),
	}
}

//+kubebuilder:rbac:groups=storage.confluent.io.apache.org,resources=localstorages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.confluent.io.apache.org,resources=localstorages/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LocalStorage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.6.4/pkg/reconcile
func (r *LocalStorageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("localStorage", req.NamespacedName)

	// TODO(user): your logic here
	instance := &storageconfluentiov1.LocalStorage{}
	err := r.Get(
		ctx,
		types.NamespacedName{Name: req.Name, Namespace: req.Namespace},
		instance,
	)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if err := deployCluster(r.Client, instance); err != nil {
		return ctrl.Result{}, err
	} else {
		return ctrl.Result{RequeueAfter: r.ReconcileWait}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *LocalStorageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&storageconfluentiov1.LocalStorage{}).
		Complete(r)
}

func LookupReconcileTime() time.Duration {
	val, exists := os.LookupEnv("RECONCILE_WAIT")
	if !exists {
		return time.Second * 10
	} else {
		v, err := time.ParseDuration(val)
		if err != nil {
			logger.Error(err, err.Error())
			// Exit Program if not valid
			os.Exit(1)
		}
		return v
	}
}

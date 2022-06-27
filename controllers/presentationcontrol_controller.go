/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/Knetic/govaluate"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"math/big"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
	"time"

	architecturecurriculumv1alpha1 "github.com/jakobmoellersap/ac-sample-operator/api/v1alpha1"
)

// PresentationControlReconciler reconciles a PresentationControl object
type PresentationControlReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

var functions = map[string]govaluate.ExpressionFunction{
	"trunc": func(args ...interface{}) (interface{}, error) {
		return big.NewFloat(args[0].(float64)).SetMode(big.AwayFromZero).Text('f', int(args[1].(float64))), nil
	},
	"strlen": func(args ...interface{}) (interface{}, error) {
		length := len(args[0].(string))
		return (float64)(length), nil
	},
	"responseTime": func(args ...interface{}) (interface{}, error) {
		start := time.Now()
		url := args[0].(string)
		result, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = result.Body.Close()
		}()
		elapsed := time.Since(start).Seconds()
		return elapsed, nil
	},
}

//+kubebuilder:rbac:groups=architecture.curriculum.my.domain,resources=presentationcontrols,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=architecture.curriculum.my.domain,resources=presentationcontrols/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=architecture.curriculum.my.domain,resources=presentationcontrols/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PresentationControlReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	control := architecturecurriculumv1alpha1.PresentationControl{}
	if err := r.Client.Get(ctx, req.NamespacedName, &control); err != nil {
		logger.Info(req.NamespacedName.String() + " got deleted!")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if control.Generation == control.Status.ObservedGeneration {
		if control.Spec.Recalculate.Every == "" {
			return ctrl.Result{}, nil
		}

		// result has already been calculated once
		if control.Status.Result != "" {
			tThen, err := time.Parse(time.RFC3339, control.Status.ObservedAt)
			if err != nil {
				tThen = time.Now()
			}
			tDiff := time.Now().Sub(tThen)
			tRecalc, _ := time.ParseDuration(control.Spec.Recalculate.Every)
			if tDiff < tRecalc {
				return ctrl.Result{RequeueAfter: tDiff}, nil
			}
		}
	}

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(control.Spec.Formula, functions)
	if err != nil {
		control.Status.Result = err.Error()
		return ctrl.Result{RequeueAfter: 3 * time.Second}, r.Client.Status().Update(ctx, &control)
	}

	result, err := expression.Eval(r.GetParameters(ctx, control.Spec.Parameters))
	if err != nil {
		control.Status.Result = err.Error()
		return ctrl.Result{RequeueAfter: 3 * time.Second}, r.Client.Status().Update(ctx, &control)
	} else {
		control.Status.Result = fmt.Sprintf("%v", result)
		control.Status.ObservedAt = time.Now().Format(time.RFC3339)
		r.Recorder.Event(&control, "Normal", "Calculation",
			fmt.Sprintf("%s = %s | %s", expression.String(), control.Status.Result, control.Spec.Parameters),
		)
	}

	if control.Generation != control.Status.ObservedGeneration {
		control.Status.ObservedGeneration = control.Generation
	}

	if control.Spec.Recalculate.Every != "" {
		duration, err := time.ParseDuration(control.Spec.Recalculate.Every)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: duration}, r.Client.Status().Update(ctx, &control)
	} else {
		return ctrl.Result{}, r.Client.Status().Update(ctx, &control)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PresentationControlReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&architecturecurriculumv1alpha1.PresentationControl{}).
		Complete(r)
}

func (r *PresentationControlReconciler) GetParameters(ctx context.Context, parameters architecturecurriculumv1alpha1.Parameters) govaluate.Parameters {
	params := govaluate.MapParameters{}
	for key, param := range parameters {
		switch param.Type {
		case architecturecurriculumv1alpha1.ParameterTypeNumber:
			params[key], _ = strconv.Atoi(param.Value)
		case architecturecurriculumv1alpha1.ParameterTypeString:
			params[key] = param.Value
		case architecturecurriculumv1alpha1.ParameterTypeSecret:
			split := strings.Split(param.Value, "/")
			params[key] = r.GetSecretParameter(ctx, split[1], split[0])
		default:
			params[key], _ = strconv.Atoi(param.Value)
		}

	}
	return params
}

func (r *PresentationControlReconciler) GetSecretParameter(ctx context.Context, name, namespace string) string {
	key := client.ObjectKey{Namespace: namespace, Name: name}
	logger := log.FromContext(ctx).WithName("secret-parameter")
	secret := &v1.Secret{}
	if err := r.Get(ctx, key, secret); err != nil {
		logger.Error(err, "error fetching secret")
	}
	return string(secret.Data["value"])
}

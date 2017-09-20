// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/alexellis/faas/gateway/requests"
	v1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

// MakeDeleteHandler delete a function
func MakeDeleteHandler(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, _ := ioutil.ReadAll(r.Body)

		request := requests.DeleteFunctionRequest{}
		err := json.Unmarshal(body, &request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(request.FunctionName) == 0 {
			w.WriteHeader(http.StatusBadRequest)
		}

		getOpts := metav1.GetOptions{}

		// This makes sure we don't delete non-labelled deployments
		deployment, findDeployErr := clientset.Extensions().Deployments(functionNamespace).Get(request.FunctionName, getOpts)

		if findDeployErr != nil {
			if errors.IsNotFound(err) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}

			w.Write([]byte(findDeployErr.Error()))
			return
		}

		if isFunction(deployment) {
			deleteFunction(clientset, request, w)

		} else {
			w.WriteHeader(http.StatusBadRequest)

			w.Write([]byte("Not a function: " + request.FunctionName))
			return
		}
	}
}

func isFunction(deployment *v1beta1.Deployment) bool {
	if deployment != nil {
		if _, found := deployment.Labels["faas_function"]; found {
			return true
		}
	}
	return false
}

func deleteFunction(clientset *kubernetes.Clientset, request requests.DeleteFunctionRequest, w http.ResponseWriter) {
	opts := &metav1.DeleteOptions{}

	if deployErr := clientset.Extensions().Deployments(functionNamespace).Delete(request.FunctionName, opts); deployErr != nil {
		if errors.IsNotFound(deployErr) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(deployErr.Error()))
		return
	}

	if svcErr := clientset.Core().Services(functionNamespace).Delete(request.FunctionName, opts); svcErr != nil {
		if errors.IsNotFound(svcErr) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.Write([]byte(svcErr.Error()))
		return
	}
}

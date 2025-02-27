package controllers

import (
	"bytes"
	"context"
	//"flag"
	"io"
	"io/ioutil"
	//"path/filepath"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	//"k8s.io/client-go/tools/clientcmd"
	//"k8s.io/client-go/util/homedir"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"k8s.io/client-go/rest"
)

type GitRepositoryWatcher struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *GitRepositoryWatcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	// get source object
	var repository sourcev1.GitRepository
	if err := r.Get(ctx, req.NamespacedName, &repository); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Nuevo Brand Brand New revision detected", "revision", repository.Status.Artifact.Revision)

	//loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	//configOverrides := &clientcmd.ConfigOverrides{}
	//kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	//config, err := kubeConfig.ClientConfig()

	config, err := rest.InClusterConfig()

	if err != nil {
		panic(err.Error())
	}

	filename := "/home/vagrant/applier/run-clone.yaml"

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err.Error())
	}
	//log.Printf("%q \n", string(b))

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
		//log.Error(err)
	}

	dd, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
		//log.Error(err)
	}

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			panic(err.Error())
			//log.Error(err)
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(c.Discovery())
		if err != nil {
			panic(err.Error())
			//log.Error(err)
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			panic(err.Error())
			//log.Error(err)
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace("default")
			}
			dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dd.Resource(mapping.Resource)
		}

		if _, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{}); err != nil {
			panic(err.Error())
			//log.Error(err)
		}
	}
	if err != io.EOF {
		panic(err.Error())
		//log.Error("eof ", err)
	}

	return ctrl.Result{}, nil
}

func (r *GitRepositoryWatcher) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sourcev1.GitRepository{}, builder.WithPredicates(GitRepositoryRevisionChangePredicate{})).
		Complete(r)
}

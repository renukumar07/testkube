package triggers

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/google/go-cmp/cmp"
	testtriggers_v1 "github.com/kubeshop/testkube-operator/apis/testtriggers/v1"
	"github.com/kubeshop/testkube-operator/pkg/informers/externalversions"
	"github.com/kubeshop/testkube-operator/pkg/validation/tests/v1/testtrigger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (s *Service) runWatcher(ctx context.Context) {
	f := informers.NewSharedInformerFactory(s.clientset, 0)
	podInformer := f.Core().V1().Pods()
	deploymentInformer := f.Apps().V1().Deployments()
	daemonsetInformer := f.Apps().V1().DaemonSets()
	statefulsetInformer := f.Apps().V1().StatefulSets()
	serviceInformer := f.Core().V1().Services()
	ingressInformer := f.Networking().V1().Ingresses()
	clusterEventInformer := f.Core().V1().Events()

	testkubeInformerFactory := externalversions.NewSharedInformerFactory(s.testKubeClientset, 0)
	testTriggerInformer := testkubeInformerFactory.Tests().V1().TestTriggers()

	podInformer.Informer().AddEventHandler(s.podEventHandler(ctx))
	deploymentInformer.Informer().AddEventHandler(s.deploymentEventHandler(ctx))
	daemonsetInformer.Informer().AddEventHandler(s.daemonSetEventHandler(ctx))
	statefulsetInformer.Informer().AddEventHandler(s.statefulSetEventHandler(ctx))
	serviceInformer.Informer().AddEventHandler(s.serviceEventHandler(ctx))
	ingressInformer.Informer().AddEventHandler(s.ingressEventHandler(ctx))
	clusterEventInformer.Informer().AddEventHandler(s.clusterEventEventHandler(ctx))
	testTriggerInformer.Informer().AddEventHandler(s.testTriggerEventHandler())

	s.logger.Debugf("trigger service: starting pod informer")
	go podInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting deployment informer")
	go deploymentInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting daemonset informer")
	go daemonsetInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting statefulset informer")
	go statefulsetInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting service informer")
	go serviceInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting ingress informer")
	go ingressInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting cluster event informer")
	go clusterEventInformer.Informer().Run(ctx.Done())
	s.logger.Debugf("trigger service: starting test trigger informer")
	go testTriggerInformer.Informer().Run(ctx.Done())

}

func (s *Service) podEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process create pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(pod.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: pod %s/%s was created in the past",
					pod.Namespace, pod.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s created", pod.Namespace, pod.Name)
			event := newPodEvent(testtrigger.EventCreated, pod)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create pod event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				s.logger.Errorf("failed to process delete pod event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: pod %s/%s deleted", pod.Namespace, pod.Name)
			event := newPodEvent(testtrigger.EventDeleted, pod)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete pod event: %v", err)
			}
		},
	}
}

func (s *Service) deploymentEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(deployment.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: deployment %s/%s was created in the past",
					deployment.Namespace, deployment.Name,
				)
				return
			}
			s.logger.Debugf("emiting event: deployment %s/%s created", deployment.Namespace, deployment.Name)
			event := newDeploymentEvent(testtrigger.EventCreated, deployment, nil)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create deployment event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDeployment, ok := oldObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for old object due to it being an unexpected type, received type %+v",
					oldDeployment,
				)
				return
			}
			newDeployment, ok := newObj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf(
					"failed to process update deployment event for new object due to it being an unexpected type, received type %+v",
					newDeployment,
				)
				return
			}
			if cmp.Equal(oldDeployment.Spec, newDeployment.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: deployment specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: deployment %s/%s updated",
				newDeployment.Namespace, newDeployment.Name,
			)
			causes := diffDeployments(oldDeployment, newDeployment)
			event := newDeploymentEvent(testtrigger.EventModified, newDeployment, causes)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update deployment event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsv1.Deployment)
			if !ok {
				s.logger.Errorf("failed to process create deployment event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: deployment %s/%s deleted", deployment.Namespace, deployment.Name)
			event := newDeploymentEvent(testtrigger.EventDeleted, deployment, nil)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete deployment event: %v", err)
			}
		},
	}
}

func (s *Service) statefulSetEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			statefulset, ok := obj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf("failed to process create statefulset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(statefulset.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: statefulset %s/%s was created in the past",
					statefulset.Namespace, statefulset.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: statefulset %s/%s created", statefulset.Namespace, statefulset.Name)
			event := newStatefulSetEvent(testtrigger.EventCreated, statefulset)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create statefulset event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldStatefulSet, ok := oldObj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update statefulset event for old object due to it being an unexpected type, received type %+v",
					oldStatefulSet,
				)
				return
			}
			newStatefulSet, ok := newObj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update statefulset event for new object due to it being an unexpected type, received type %+v",
					newStatefulSet,
				)
				return
			}
			if cmp.Equal(oldStatefulSet.Spec, newStatefulSet.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: statefulset specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: statefulset %s/%s updated",
				newStatefulSet.Namespace, newStatefulSet.Name,
			)
			event := newStatefulSetEvent(testtrigger.EventModified, newStatefulSet)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update statefulset event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			statefulset, ok := obj.(*appsv1.StatefulSet)
			if !ok {
				s.logger.Errorf("failed to process delete statefulset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: statefulset %s/%s deleted", statefulset.Namespace, statefulset.Name)
			event := newStatefulSetEvent(testtrigger.EventDeleted, statefulset)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete statefulset event: %v", err)
			}
		},
	}
}

func (s *Service) daemonSetEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			daemonset, ok := obj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf("failed to process create daemonset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(daemonset.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: daemonset %s/%s was created in the past",
					daemonset.Namespace, daemonset.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: daemonset %s/%s created", daemonset.Namespace, daemonset.Name)
			event := newDaemonSetEvent(testtrigger.EventCreated, daemonset)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create daemonset event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDaemonSet, ok := oldObj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update daemonset event for old object due to it being an unexpected type, received type %+v",
					oldDaemonSet,
				)
				return
			}
			newDaemonSet, ok := newObj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf(
					"failed to process update daemonset event for new object due to it being an unexpected type, received type %+v",
					newDaemonSet,
				)
				return
			}
			if cmp.Equal(oldDaemonSet.Spec, newDaemonSet.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: daemonset specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: daemonset %s/%s updated",
				newDaemonSet.Namespace, newDaemonSet.Name,
			)
			event := newDaemonSetEvent(testtrigger.EventModified, newDaemonSet)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update daemonset event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			daemonset, ok := obj.(*appsv1.DaemonSet)
			if !ok {
				s.logger.Errorf("failed to process delete daemonset event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: daemonset %s/%s deleted", daemonset.Namespace, daemonset.Name)
			event := newDaemonSetEvent(testtrigger.EventDeleted, daemonset)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete daemonset event: %v", err)
			}
		},
	}
}

func (s *Service) serviceEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			service, ok := obj.(*corev1.Service)
			if !ok {
				s.logger.Errorf("failed to process create service event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(service.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: service %s/%s was created in the past",
					service.Namespace, service.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: service %s/%s created", service.Namespace, service.Name)
			event := newServiceEvent(testtrigger.EventCreated, service)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create service event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldService, ok := oldObj.(*corev1.Service)
			if !ok {
				s.logger.Errorf(
					"failed to process update service event for old object due to it being an unexpected type, received type %+v",
					oldService,
				)
				return
			}
			newService, ok := newObj.(*corev1.Service)
			if !ok {
				s.logger.Errorf(
					"failed to process update service event for new object due to it being an unexpected type, received type %+v",
					newService,
				)
				return
			}
			if cmp.Equal(oldService.Spec, newService.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: service specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: service %s/%s updated",
				newService.Namespace, newService.Name,
			)
			event := newServiceEvent(testtrigger.EventModified, newService)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update service event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			service, ok := obj.(*corev1.Service)
			if !ok {
				s.logger.Errorf("failed to process delete service event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: service %s/%s deleted", service.Namespace, service.Name)
			event := newServiceEvent(testtrigger.EventDeleted, service)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete service event: %v", err)
			}
		},
	}
}

func (s *Service) ingressEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ingress, ok := obj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf("failed to process create ingress event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(ingress.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: ingress %s/%s was created in the past",
					ingress.Namespace, ingress.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: ingress %s/%s created", ingress.Namespace, ingress.Name)
			event := newIngressEvent(testtrigger.EventCreated, ingress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create ingress event: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldIngress, ok := oldObj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf(
					"failed to process update ingress event for old object due to it being an unexpected type, received type %+v",
					oldIngress,
				)
				return
			}
			newIngress, ok := newObj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf(
					"failed to process update ingress event for new object due to it being an unexpected type, received type %+v",
					newIngress,
				)
				return
			}
			if cmp.Equal(oldIngress.Spec, newIngress.Spec) {
				s.logger.Debugf("trigger service: watcher component: no-op update trigger: ingress specs are equal")
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: emiting event: ingress %s/%s updated",
				oldIngress.Namespace, newIngress.Name,
			)
			event := newIngressEvent(testtrigger.EventModified, newIngress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching update ingress event: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			ingress, ok := obj.(*networkingv1.Ingress)
			if !ok {
				s.logger.Errorf("failed to process delete ingress event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: ingress %s/%s deleted", ingress.Namespace, ingress.Name)
			event := newIngressEvent(testtrigger.EventDeleted, ingress)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching delete ingress event: %v", err)
			}
		},
	}
}

func (s *Service) clusterEventEventHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			clusterEvent, ok := obj.(*corev1.Event)
			if !ok {
				s.logger.Errorf("failed to process create cluster event event due to it being an unexpected type, received type %+v", obj)
				return
			}
			if inPast(clusterEvent.CreationTimestamp.Time, s.started) {
				s.logger.Debugf(
					"trigger service: watcher component: no-op create trigger: cluster event %s/%s was created in the past",
					clusterEvent.Namespace, clusterEvent.Name,
				)
				return
			}
			s.logger.Debugf("trigger service: watcher component: emiting event: cluster event %s/%s created", clusterEvent.Namespace, clusterEvent.Name)
			event := NewClusterEventEvent(testtrigger.EventCreated, clusterEvent)
			if err := s.match(ctx, event); err != nil {
				s.logger.Errorf("event matcher returned an error while matching create cluster event event: %v", err)
			}
		},
	}
}

func (s *Service) testTriggerEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.logger.Errorf("failed to process create testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: adding testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.addTrigger(t)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			t, ok := newObj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.logger.Errorf(
					"failed to process update testtrigger event for new testtrigger due to it being an unexpected type, received type %+v",
					newObj,
				)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: updating testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.updateTrigger(t)
		},
		DeleteFunc: func(obj interface{}) {
			t, ok := obj.(*testtriggers_v1.TestTrigger)
			if !ok {
				s.logger.Errorf("failed to process delete testtrigger event due to it being an unexpected type, received type %+v", obj)
				return
			}
			s.logger.Debugf(
				"trigger service: watcher component: deleting testtrigger %s/%s for resource %s on event %s",
				t.Namespace, t.Name, t.Spec.Resource, t.Spec.Event,
			)
			s.removeTrigger(t)
		},
	}
}

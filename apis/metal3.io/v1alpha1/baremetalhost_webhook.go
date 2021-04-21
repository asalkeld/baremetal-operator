package v1alpha1

// +kubebuilder:webhook:verbs=create;update,path=/validate-metal3-io-v1alpha1-baremetalhost,mutating=false,groups=metal3.io,resources=baremetalhosts,versions=v1alpha1,name=vbaremetalhost.kb.io

// see https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html

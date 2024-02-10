# Goobernetes

Kubernetes objects as _code_, because templates are annoying.

`main.go` is a progam that prints the required k8s manifests for an example app
in a given deployment environment. It uses the standard k8s SDK to define
objects and some helpers to print as YAML.

There is not much code to be shared really, the point of the program is to
enable working on definitions as plain old code instead of as YAML and various
DSLs to handle differences between instances of the app. It's all hard coded in
the program, edit the code to change your configuration.

## Usage

```
❯ make
go build -o generate main.go
./generate --env=local > k8s.local.yaml
./generate --env=production > k8s.production.yaml
```

<Open main.go, make some config changes, save>

```
❯ make
go build -o generate main.go
./generate --env=local > k8s.local.yaml
./generate --env=production > k8s.production.yaml

❯ git diff *yaml
diff --git a/k8s.production.yaml b/k8s.production.yaml
index 72a2c36..c6211ba 100644
--- a/k8s.production.yaml
+++ b/k8s.production.yaml
@@ -4,7 +4,7 @@ kind: Deployment
 metadata:
   name: echo
 spec:
-  replicas: 3
+  replicas: 4
   selector:
     matchLabels:
       app: echo
```

## Why?

* I find YAML patching harder to reason about than calling functions
* Important dependencies are easier to see in code than spread out templates
* There are more tools around to analyze your code

You still get the same YAML out in the end, for executing any other validators
or linters.

## Experimental

All objects are printed through a pipe to `kubectl-neat` for removing crud.
This is a bit hacky but seemed like the easiest way.

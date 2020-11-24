package cptcconnectivity

// nop is unused variable that just enables Go compiler to find something to
// compile from this package when all other files have build constraing
// `k8srequired`. This prevents `go list` and therefore `pkger` to fail when
// enumerating resources in source tree.
var nop = "enable go list in this package for pkger"

// "Current" renderer — used for draft documents that have no pin.
// At v1.0.0 this is an alias of the v1.0.0 bundle. When the first
// renderer version bump happens, rewire this file to point at the
// new snapshot (and copy the previous one into a frozen directory).

export * from "./v1.0.0/index";

# W5 Flag-Flip Verification

Staging flip UTC: YYYY-MM-DDTHH:MM:SSZ
Prod flip UTC:    YYYY-MM-DDTHH:MM:SSZ
Operator:         @handle

## Staging — API replicas
| Replica ID | Image tag | Flag value (from /api/v1/feature-flags) |
|------------|-----------|------------------------------------------|
| pod/api-staging-1 | vX.Y.Z | true |

## Staging — Worker replicas
| Replica ID | Image tag | Flag value (printenv) |
|------------|-----------|------------------------|
| pod/worker-staging-1 | vX.Y.Z | true |

## Staging — Frontend replicas
| Replica ID | Image tag | printenv value | /config.json value |
|------------|-----------|-----------------|---------------------|
| pod/web-staging-1 | vX.Y.Z | true (or baked) | true |

## Production — API replicas
| Replica ID | Image tag | Flag value |
|------------|-----------|------------|
| pod/api-prod-1 | vX.Y.Z | true |

## Production — Worker replicas
| Replica ID | Image tag | Flag value |
|------------|-----------|------------|
| pod/worker-prod-1 | vX.Y.Z | true |

## Production — Frontend replicas
| Replica ID | Image tag | printenv value | /config.json value |
|------------|-----------|-----------------|---------------------|
| pod/web-prod-1 | vX.Y.Z | true (or baked) | true |

## Attestation — ALL THREE WORKLOAD CLASSES
- [ ] Every staging API replica returned `true`.
- [ ] Every staging worker replica has `METALDOCS_DOCX_V2_ENABLED=true`.
- [ ] Every staging frontend replica serves `/config.json` with the flag `true`.
- [ ] Every production API replica returned `true`.
- [ ] Every production worker replica has `METALDOCS_DOCX_V2_ENABLED=true`.
- [ ] Every production frontend replica serves `/config.json` with the flag `true`.
- [ ] Image tag is identical across replicas within each class (no mid-flip rolling deploy).
- [ ] Admin sign-off: @handle — YYYY-MM-DD
- [ ] SRE sign-off: @handle — YYYY-MM-DD

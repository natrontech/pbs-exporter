// The predicateType field must match this string
predicateType: "https://slsa.dev/provenance/v0.2"

predicate: {
  // This condition verifies that the builder is the builder we expect and trust.
  builder: {
    id: =~"^https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@refs/tags/v[0-9]+.[0-9]+.[0-9]+$"
  }
  invocation: {
    configSource: {
      // This condition verifies the entrypoint of the workflow.
      entryPoint: ".github/workflows/release.yml"

      // This condition verifies that the image was generated from the source repository we expect.
      uri: =~"^git\\+https://github.com/natrontech/pbs-exporter@refs/tags/v[0-9]+.[0-9]+.[0-9]+(-rc.[0-9]+)?$"
    }
  }
}

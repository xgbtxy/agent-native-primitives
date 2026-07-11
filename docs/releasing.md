# Release process

Releases are tag-driven and contain one Windows package and one macOS Universal package.

```bash
git tag -a v0.1.0 -m "tooltruth v0.1.0"
git push origin v0.1.0
```

The release workflow:

1. reruns race tests and `go vet`;
2. builds Windows x86_64;
3. builds macOS x86_64 and arm64, then combines them with `lipo`;
4. packages the binary with the license and README;
5. generates `SHA256SUMS.txt`;
6. emits GitHub build-provenance attestations;
7. publishes the GitHub Release only after every preceding job succeeds.

The CLI version is injected from the tag at link time. Do not create a tag that does not match the intended public version.

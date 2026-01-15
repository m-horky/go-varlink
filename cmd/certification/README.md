# Certification

This package contains certification as per [the specification](https://varlink.org/Language-Bindings#how-to-test-new-language-bindings).

By default, it runs on GitHub actions (see `.github/workflows/certification.yaml`), but it is possible to run it locally as well.

## Client certification

```bash
git clone https://github.com/varlink/python.git
PYTHONPATH=python/ python3 -m varlink.tests.test_certification --varlink=tcp:127.0.0.1:12345 &
PID=$!
go run ./cmd/certification client -protocol tcp -socket 127.0.0.1:12345
kill $PID >/dev/null || true
```

## Server certification

```bash
git clone https://github.com/varlink/python.git
go run cmd/certification/*.go server -protocol tcp -socket 127.0.0.1:12345 &
PID=$!
sleep 2
cd python
python3 -m varlink.tests.test_certification --client --varlink=tcp:127.0.0.1:12345
cd -
kill $PID >/dev/null || true
```

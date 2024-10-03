There is an issue in streamlit that affects Mantium. When streamlit version > 1.35.0, the dashboard becomes slower than < 1.35.0. This issue is being tracked in [this issue](https://github.com/streamlit/streamlit/issues/9456) in the streamlit github repository.

To counter this issue, it's necessary to make some changes to the streamlit code, build the frontend, and replace the installed streamlit library files:

1. Clone the [streamlit repository](https://github.com/streamlit/streamlit) and checkout to any version.
2. Make the changes in the streamlit code as mentioned in the issue.
3. Install the frontend dependencies:

```
cd frontend/
yarn install
cd ..
```

5. Remove the python protobuf part from the `Makefile` file:

```
.PHONY: protobuf
# Recompile Protobufs for Python and the frontend.
protobuf: check-protoc

    <-- COMMENT/REMOVE THIS PART -->
	protoc \
		--proto_path=proto \
		--python_out=lib \
		--mypy_out=lib \
		proto/streamlit/proto/*.proto
    <-- -->

	@# JS protobuf generation. The --es6 flag generates a proper es6 module.
	cd frontend/ ; ( \
		echo "/* eslint-disable */" ; \
		echo ; \
		yarn --silent pbjs \
			../proto/streamlit/proto/*.proto \
			--path=proto -t static-module --wrap es6 \
	) > ./lib/src/proto.js

	@# Typescript type declarations for our generated protobufs
	cd frontend/ ; ( \
		echo "/* eslint-disable */" ; \
		echo ; \
		yarn --silent pbts ./lib/src/proto.js \
	) > ./lib/src/proto.d.ts
```

6. Build the frontend protobuf files:

```
make protobuf
```

7. Build the frontend:

```
cd frontend/
yarn build
```

8. Replace the installed streamlit library files with the built frontend files:

```
rm -rf /path/to/lib/python3.8/site-packages/streamlit/static
cp -r /path/to/streamlit/frontend/app/build /path/to/lib/python3.8/site-packages/streamlit/static
```

# Lab 4: Proof-of-Work Blockchain

## Downloading Dependencies
```
go mod tidy
```

## Building
The code does not include a main.go file so no need to build it.

## Testing
```
make test
```
This command will run all the tests defined in the Makefile.

## Notes
Please note that the success of the tests is closely related to the computing power of your CPU. The tests involve mining blocks, which require significant computational resources. If you encounter test failures, it may be due to the target difficulty being too high for your system to complete the mining process within the specified timeout.

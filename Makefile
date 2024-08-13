# Run all tests
build:

test:
	cd src && go clean -testcache && go test -v blockchain/tests

doc:
	cd src/blockchain && go doc -u -all > blockchain-doc.txt
	cd src/miner && go doc -u -all > miner-doc.txt
	cd src/tracker && go doc -u -all > tracker-doc.txt
	cd src/user && go doc -u -all > user-doc.txt
	cd src/tests && go doc -u -all > tests-doc.txt
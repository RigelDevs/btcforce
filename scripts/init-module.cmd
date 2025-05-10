@echo off
REM scripts/init-module.cmd

echo === Initializing Go Module ===

cd ..
go mod init btcforce

echo.
echo === Adding Dependencies ===

go get github.com/btcsuite/btcd/btcec/v2
go get github.com/btcsuite/btcd/btcutil
go get github.com/btcsuite/btcd/chaincfg
go get github.com/btcsuite/btcd/chaincfg/chainhash
go get github.com/joho/godotenv
go get github.com/cockroachdb/pebble

echo.
echo === Tidying Module ===

go mod tidy

echo.
echo === Module Initialized Successfully ===

echo.
echo Current go.mod:
type go.mod
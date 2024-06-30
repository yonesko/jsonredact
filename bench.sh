git checkout $1

go test -v -benchmem -bench . -run ^$ -count=1 > $(git rev-parse --abbrev-ref HEAD).txt

git checkout $2

go test -v -benchmem -bench . -run ^$ -count=1 > $(git rev-parse --abbrev-ref HEAD).txt

benchstat $1.txt $2.txt
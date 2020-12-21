package blockchain

import (
	"log"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/incognitochain/incognito-chain/common"
)

var (
	randomNumber = int64(1000)
	candidates   = []string{
		"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",
	}
	candidates1 = candidates
	candidates2 = []string{
		"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",
	}
	candidates3        = append(candidates1, candidates2...)
	assignedCandidates = make(map[string]byte)
	priorityKeys       = []string{
		"002861fd7eede6f3e28394a6ccc0b559e696509151dc75d9e5630356e23da90f",
		"1ecbbbd5076fbbc574637893692de5d38b59de23c44c063da835bd44be943a97",
		"2c86c4a23fb3245ebda2fb6d70c6487804672ab5c5038e70613ebe4562903a85",
		"44b83685be993e0b2a689a3c15b5fd26f2b27a3d40015bbbecfbfe6282efb925",
		"8eda59aa90e707ce8afc3eca7473aabcc5a505c4e3ec9c4248a211f733963c8f",
		"945605ea416e86ca6aa855fbaae0b4335c60cb470e1c94fbb12ce681d06d7c43",
		"a4b23abd534a82e6a5969f1505e7a3dd91d22ff842571d328ac746f073b84ac9",
		"a6b16be39ead5680aa799a213e9bf22dac6a1f16598ba8813a79fb8150dd7961",
		"a9c5558021c0b8277e0291258ad7d42ba0a9e868af59fb0ea205cb4a53442830",
		"b43a5a6e87d98bdf219766c84734df0760416682aa75f9243a9148dfeb2f9e67",
		"e7191ee68362219eabdeff08178adcab6690f94e28dee1e525249433bdea5abf",
	}
	expectedShuffledCandidates = []string{
		"121VhftSAygpEJZ6i9jGk9dTsGQVZNEnYLWhjXjomneDCT9XC6zGjeTW8ENTJVPooNnQBRuRdpwe9DoNVdLF4j3iy6Ld2p1eNLK8ek1bNSNrHFVjtsHaQVpcoHBy6nGghA9y7pr6ne6W7MRdAXLPHEDgpAodnRVxFSc2zyUbA48XKQVRtSVUneKjcdhDHP1hY9gC1EYLWtqn4weLzHKA84w77rHFhwwV2sbseHDqsmtQa4k5aiTCcXoUdmMpEuvadtSmKR33wciA3FNr5h125ce9ge1eSFPznvXNsCy7sA1rc4YJxipgwGoDDWSftUfnh5vY37WbwwNLhmMRvxtjHP7WBkSrLrKdamr8TdUu93cQyrykYRxbCben7pK1N75NknsSqLSihBnMTLo8gcFDGaDtQixPiJxMkefZ3qfHHwdvaL27",
		"121VhftSAygpEJZ6i9jGk9dnzbXovyLBfFHs6cuMcjFEAB6FphEtuXPXCCNga16A1LkSHHSgLq9rFGQ496VnxHxDL8Ar6duhJxpEdcr75rkRFBzEvPNTkRYVbVcBUMtWd5PugsB6QjYt5CJQyUtzzbctC15AeX64aHrK1QwHJjFz8hnMz3eh8P8SPqKQ6zXhByqHm7YrcCY5uKZUn7CqM9RTwwJaqUhtqsHKyozBkfw26XhfkwvFh4vvL6McE5Ty1ztgiygUp8tt7haPEjVNGCqnwpPEB76oPVFTiKGePY1XqHy6aAFvuUBdmnNFEXrQPnh5xz5ULpq6PMJjVFvpu5kXcnaJbFBRZJcsJm1y69n56zUFKSg4LiNwqymf84U2SiPKT12cFehJedwfJkBMwnGpDAfaEZZvqcNRD9nVRUG4no9X",
		"121VhftSAygpEJZ6i9jGk9dPK5DE41KmqUhp2EFJXyyHNrQomEh7QW7icEh2zymM5953C2HN2kYYCbepkL37Qny6mgyxGiAxxqEEkCHnqK4FsNBDbGwojMmVCgURwmS1oRqpfoFYhKwFUTzjyYduxBNegdgUw9VEYXJe3xSMBib24oBRhjbmK7LZ5qxRqKMsV7Lp49rPi2QGiizqiVKSsaAD2QMHEaC5ZBsGJvtRRMdGarHfe6yLtWAvcPLpsuSkYHfNni5Nh61LT1LwYLqSwDqNVKC3ew6RfRvmFfFTPhBGUcbhRKog6djipwKW9RZ7jvpgSiwifLriiz8h25ziR3Guh7k1cgyYm6TxQHrrLBRUgXKGvLkeAEi9ThrCsuQnbE61re6UmjwCcbPd9j53fzVmm4JJ19kMsyGUL2m7obSsdxvM",
		"121VhftSAygpEJZ6i9jGk9a4wsKYVdTBhFsvZ4jwhzaWjFsondoUtyYpqF3hjWmPEUvMJF1Wh9NMhRVGaLYW8JtmS2JcP3Zq1L2AvYjYJaWm5CRqoYPz3DUEeJFSbUDdZPcp5YAL3FizX3J192zR3kWA967p6cZUgVP6e9LbHBdQBFGCQYRYKw8DNRCUzophTmoFeFwjWJF3EuXAWX2gv5ASj2Nem9YytNtjhZHCRS7Vz8pvgLdMJVa7B9fpS6oBV7nS4LWdeZc8NPC9VGKbTa1MCy7YkXfjbHfKpowEArnB9CLaLpWmSiaZruTiRxtZQZkU9z3YCMZW2dW5SHmwMGEseu6WwPPqgLz32tazKzNzHgiAJp561pxfCm7HF4r6VxtTqPKe8gjwLfDZqw3X2ew6rQ8Vo2csnjWrSQJAxYuMeinW",
		"121VhftSAygpEJZ6i9jGk9a1qL1ZjK3QhzqpDT4uJop7tnEfahvCVRVbKPsuH94uGMqj1a1nLQSAcQypiUP3yc1s3t4jCRae8Kf6VXJRyHngt9X9iT2yJexKgRhnjzTfJYU4VkXV3w3xwiFCp5pGnDdocA8aq1SUCVMwfKAnhxjDHQDDWMMaxSLvjU56ZqaNZtsCQGTaySZLxc2RmCgMUeC5JQkF4t3P2NbTaZzz9JreLtZqSL3DVLacPwXy2enu3QMQYcPbZRVbz2Rxuf8e3hHYwjbAqvDJayCk9iasJnmZKP2gVRQhHUcX7cED4U22TMzi2rE6FoVMebThoDB2Dp21BW21qthS26Rkxe7UbxTesss9Xk1LSQsh8tRV1yxyGJ3DgtY2csBicRmT5PxQ3j9FwdSViKX82M1u9qv7ZcQSnvrJ",
		"121VhftSAygpEJZ6i9jGk9dXXyF2XfP9G56ZoZ8hAtLMc3i33FWSqa3FJgMkc7pTs5qdaeJHAFfJpwjoGWazHWzknPSh8d319L9xMoz2TsLCefqPeP8Kqf24C8fuY9RCTCvnAmecnXL6SJyiVTP6Vjjhcvvdk8cQHVUSnnXxbuufgRckyw9Mc7VrgpG3qzBfeYCfWkurDmdVnyjh7jPsZQM1sBjRFiNSjQQo7HQgaMi2YQp9WGpE4kJfF265eqXySrT8BycKLnjunED9B1TU6WNs2e9aFB2u82tMzwoRwTHWbAgc9rixwM3UiAAnhiEhx56nkidGRsqo2LR5AQASHEUP6aHtnz1wcwovEPthXXChDhRuDAa5PHvfz57LQpWZANf9HV26J5wYFysJb4bN9vRcxTJdijRzDAt5QbBJs5DJo64N",
		"121VhftSAygpEJZ6i9jGk9a5oQvgecAms7BtyyjGxpySigmDAdo8af26UKNmXYjNUhFVp4NN6RpRJFTGn57w7evPi3HkaF8ToXsCJ7ceaZ5p6hrCHLN6tKm5sjBEo3yusZZayurNMsrLGRhBE2i8Xhxkns8uN8c9WY5kcwSVsyD9f7399fMzRqtMB7TjyE4ad2KWDmteZZWuZB79jYB5wHWcxRyUxYaQ761gT8oMJ6FKr7wdPYAeuJ7Pai71Xi9YdPUDNQ1dZ7Uq1m7wxavKKax14Tuf9onu9oZDTeat7SNK1PDxjvf2uwkEmcHAp7qzp8c7igm8X6VjC6685gdThcdEHPxiwDsu3UxQyXK1fqwSHDHx3Ff7w5xKeDK8zJNghCLBbZ3HowQbT7hAKqu3N5puMKw5cjQJndA4trRw5yuzXxbf",
		"121VhftSAygpEJZ6i9jGk9eKW2e2edvTfmZXDVdt8qTxwzuUVFHUtpf9JGKZN9damEuDoy381bwjwq6g5D4M6Zn2KUh2giSEktUca7nbvoM7L23aq9XXtsmyQKvVBseCVvUNmyERNHRZpzvzNCn6gwnzyMR58uBcibUBwV5441jYR18RxwwyKh2w8S6ogEmdrERAxdMYdxPXwj43Ve5aHnZtT8ZfV6vPPKnPmgyM95Bpw5ep1HmWoZvtF4s5WkqbCoaAYoBd94Bcysx4wzVQbvmU1SCD5hCpF3nDtE9n7G3SnNAerDE3DWiZt5GjpLtSpQyAjbaQekqpBnyKVqbsnPiDh5EynFqzETZLkt9p9hvAK2EuzQXBVZXpFwbwoiiw918LhZ2nw11xy2eQF6hGXG5GzaYYiBHmA4Fyd8rywPGA1QH4",
		"121VhftSAygpEJZ6i9jGk9a5Kfw77TTKCB5FhKUBU1JJKrvogDS3g9JhfQXY4PP9xEzfnTRB43MhcpxmbR7qempyNsRu7k3oY59xphaP843bUnZWk17LiedGSaDUfc7xEB2jNt94rpm1FXF9hjPRijDqeqyyBhFV3uyqhmfCdnH1xxYpJW8XLk45Jhpf5vGZpy2qFn3vVUtmxk2TMdW68AsBvkT2PkFGQDqiWYRMBXStV1Npzxu3CUKamd74ZXA66tSY7rP1QE4vSFDCLX23rJE9tjVZuty74Edbin4ZzgG2PhLqvv7s3gpNzaY6oaaxRSbon7JWujnF3uv8o6DvdraGkWiJa1VhXrJjVDzNs5WVNtETGMPy58uUPkemv8oto6yCJzxiDgjUR9Yjy3mfnY3eNWV89BocWjNkBT4Y9JU8HMD8",
		"121VhftSAygpEJZ6i9jGk9ZogHDZ369maVpypAooJmsxv3QK9apuYCZxe5iM61B1W8CgRRNuQuBEwBpjMM1bMBeL1buAa2LpokfP8FCKA2gfiLn9QCLew61SQYeTAgRG9rqouR5zph6ECxDbU7qYTytd5f8QFw2jedB9Y7C9eBCRgg6XHiKgaRK7WFN9iNYB3L4KGYGQCJZUTW7zLNxX2uQrpR3sPsERnsQYSfar2dLhH2d7R8Avywn1aZPH2hDLp1ytjv8kgbUM74fT7Dton9dCVm3AV4zgc39eX9j42kx2U7mXWeG7MKDZUBiAVAfWyxvecjpMt5juzuXvzRmiN5qkZJ8QVFAQ2XLSpszFZtuPJET5pSbfYn1StqvALbGKM5jU4n6oSGuQdFmSehMbkjZwZJ5oNBf1X77ax9hXigt3ejqh",
		"121VhftSAygpEJZ6i9jGk9drax2iDha73FtSVnju8AYxEHLxLqrgcB5ocJPiJ3BBRcRgZ1TmTQxnEsSpSm3wEdaRd98Y7YEHBwrMsQdaPsA66MJeTxy9ZDpyAD82sWfYzHNA7Q8pjpBvCrvxKHQTz6NBRZXspvCtxozStN6mJMJWoMUyMBccZLgRMTN7dDXArcJVPtTVQWqjT15DToLbzY3qdnc1vdZDTq916qNdQ9PbCVwbswdqtdCxEwCoYo9uLS9gdkvJaJdU1wNuYFYvFgiAQFa6mgjNZWiDnLyYBtVX3VyfVGe4K8fRgG9bgj15ZG7UypBoQTjxxJJHDmMy23VHV3qSDr8bjLnhLVYgHmkpuHfhxFX2B9KXXhkc4XMgxxyC83HWaz2XvS1eNuTMVbKUd3tjCBkZQJszBDsKa5R7gJqH",
	}
	expectedRemainCandidates = []string{
		"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",
	}
	expectedAssignedCandidates = map[byte][]string{
		byte(0): []string{"121VhftSAygpEJZ6i9jGk9dPK5DE41KmqUhp2EFJXyyHNrQomEh7QW7icEh2zymM5953C2HN2kYYCbepkL37Qny6mgyxGiAxxqEEkCHnqK4FsNBDbGwojMmVCgURwmS1oRqpfoFYhKwFUTzjyYduxBNegdgUw9VEYXJe3xSMBib24oBRhjbmK7LZ5qxRqKMsV7Lp49rPi2QGiizqiVKSsaAD2QMHEaC5ZBsGJvtRRMdGarHfe6yLtWAvcPLpsuSkYHfNni5Nh61LT1LwYLqSwDqNVKC3ew6RfRvmFfFTPhBGUcbhRKog6djipwKW9RZ7jvpgSiwifLriiz8h25ziR3Guh7k1cgyYm6TxQHrrLBRUgXKGvLkeAEi9ThrCsuQnbE61re6UmjwCcbPd9j53fzVmm4JJ19kMsyGUL2m7obSsdxvM", "121VhftSAygpEJZ6i9jGk9a1qL1ZjK3QhzqpDT4uJop7tnEfahvCVRVbKPsuH94uGMqj1a1nLQSAcQypiUP3yc1s3t4jCRae8Kf6VXJRyHngt9X9iT2yJexKgRhnjzTfJYU4VkXV3w3xwiFCp5pGnDdocA8aq1SUCVMwfKAnhxjDHQDDWMMaxSLvjU56ZqaNZtsCQGTaySZLxc2RmCgMUeC5JQkF4t3P2NbTaZzz9JreLtZqSL3DVLacPwXy2enu3QMQYcPbZRVbz2Rxuf8e3hHYwjbAqvDJayCk9iasJnmZKP2gVRQhHUcX7cED4U22TMzi2rE6FoVMebThoDB2Dp21BW21qthS26Rkxe7UbxTesss9Xk1LSQsh8tRV1yxyGJ3DgtY2csBicRmT5PxQ3j9FwdSViKX82M1u9qv7ZcQSnvrJ", "121VhftSAygpEJZ6i9jGk9a5oQvgecAms7BtyyjGxpySigmDAdo8af26UKNmXYjNUhFVp4NN6RpRJFTGn57w7evPi3HkaF8ToXsCJ7ceaZ5p6hrCHLN6tKm5sjBEo3yusZZayurNMsrLGRhBE2i8Xhxkns8uN8c9WY5kcwSVsyD9f7399fMzRqtMB7TjyE4ad2KWDmteZZWuZB79jYB5wHWcxRyUxYaQ761gT8oMJ6FKr7wdPYAeuJ7Pai71Xi9YdPUDNQ1dZ7Uq1m7wxavKKax14Tuf9onu9oZDTeat7SNK1PDxjvf2uwkEmcHAp7qzp8c7igm8X6VjC6685gdThcdEHPxiwDsu3UxQyXK1fqwSHDHx3Ff7w5xKeDK8zJNghCLBbZ3HowQbT7hAKqu3N5puMKw5cjQJndA4trRw5yuzXxbf"},
		byte(1): []string{
			"121VhftSAygpEJZ6i9jGk9dTsGQVZNEnYLWhjXjomneDCT9XC6zGjeTW8ENTJVPooNnQBRuRdpwe9DoNVdLF4j3iy6Ld2p1eNLK8ek1bNSNrHFVjtsHaQVpcoHBy6nGghA9y7pr6ne6W7MRdAXLPHEDgpAodnRVxFSc2zyUbA48XKQVRtSVUneKjcdhDHP1hY9gC1EYLWtqn4weLzHKA84w77rHFhwwV2sbseHDqsmtQa4k5aiTCcXoUdmMpEuvadtSmKR33wciA3FNr5h125ce9ge1eSFPznvXNsCy7sA1rc4YJxipgwGoDDWSftUfnh5vY37WbwwNLhmMRvxtjHP7WBkSrLrKdamr8TdUu93cQyrykYRxbCben7pK1N75NknsSqLSihBnMTLo8gcFDGaDtQixPiJxMkefZ3qfHHwdvaL27",
			"121VhftSAygpEJZ6i9jGk9dnzbXovyLBfFHs6cuMcjFEAB6FphEtuXPXCCNga16A1LkSHHSgLq9rFGQ496VnxHxDL8Ar6duhJxpEdcr75rkRFBzEvPNTkRYVbVcBUMtWd5PugsB6QjYt5CJQyUtzzbctC15AeX64aHrK1QwHJjFz8hnMz3eh8P8SPqKQ6zXhByqHm7YrcCY5uKZUn7CqM9RTwwJaqUhtqsHKyozBkfw26XhfkwvFh4vvL6McE5Ty1ztgiygUp8tt7haPEjVNGCqnwpPEB76oPVFTiKGePY1XqHy6aAFvuUBdmnNFEXrQPnh5xz5ULpq6PMJjVFvpu5kXcnaJbFBRZJcsJm1y69n56zUFKSg4LiNwqymf84U2SiPKT12cFehJedwfJkBMwnGpDAfaEZZvqcNRD9nVRUG4no9X",
			"121VhftSAygpEJZ6i9jGk9a4wsKYVdTBhFsvZ4jwhzaWjFsondoUtyYpqF3hjWmPEUvMJF1Wh9NMhRVGaLYW8JtmS2JcP3Zq1L2AvYjYJaWm5CRqoYPz3DUEeJFSbUDdZPcp5YAL3FizX3J192zR3kWA967p6cZUgVP6e9LbHBdQBFGCQYRYKw8DNRCUzophTmoFeFwjWJF3EuXAWX2gv5ASj2Nem9YytNtjhZHCRS7Vz8pvgLdMJVa7B9fpS6oBV7nS4LWdeZc8NPC9VGKbTa1MCy7YkXfjbHfKpowEArnB9CLaLpWmSiaZruTiRxtZQZkU9z3YCMZW2dW5SHmwMGEseu6WwPPqgLz32tazKzNzHgiAJp561pxfCm7HF4r6VxtTqPKe8gjwLfDZqw3X2ew6rQ8Vo2csnjWrSQJAxYuMeinW",
		},
	}
)
var _ = func() (_ struct{}) {
	assignedCandidates["121VhftSAygpEJZ6i9jGk9a5Kfw77TTKCB5FhKUBU1JJKrvogDS3g9JhfQXY4PP9xEzfnTRB43MhcpxmbR7qempyNsRu7k3oY59xphaP843bUnZWk17LiedGSaDUfc7xEB2jNt94rpm1FXF9hjPRijDqeqyyBhFV3uyqhmfCdnH1xxYpJW8XLk45Jhpf5vGZpy2qFn3vVUtmxk2TMdW68AsBvkT2PkFGQDqiWYRMBXStV1Npzxu3CUKamd74ZXA66tSY7rP1QE4vSFDCLX23rJE9tjVZuty74Edbin4ZzgG2PhLqvv7s3gpNzaY6oaaxRSbon7JWujnF3uv8o6DvdraGkWiJa1VhXrJjVDzNs5WVNtETGMPy58uUPkemv8oto6yCJzxiDgjUR9Yjy3mfnY3eNWV89BocWjNkBT4Y9JU8HMD8"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9dnzbXovyLBfFHs6cuMcjFEAB6FphEtuXPXCCNga16A1LkSHHSgLq9rFGQ496VnxHxDL8Ar6duhJxpEdcr75rkRFBzEvPNTkRYVbVcBUMtWd5PugsB6QjYt5CJQyUtzzbctC15AeX64aHrK1QwHJjFz8hnMz3eh8P8SPqKQ6zXhByqHm7YrcCY5uKZUn7CqM9RTwwJaqUhtqsHKyozBkfw26XhfkwvFh4vvL6McE5Ty1ztgiygUp8tt7haPEjVNGCqnwpPEB76oPVFTiKGePY1XqHy6aAFvuUBdmnNFEXrQPnh5xz5ULpq6PMJjVFvpu5kXcnaJbFBRZJcsJm1y69n56zUFKSg4LiNwqymf84U2SiPKT12cFehJedwfJkBMwnGpDAfaEZZvqcNRD9nVRUG4no9X"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9ZogHDZ369maVpypAooJmsxv3QK9apuYCZxe5iM61B1W8CgRRNuQuBEwBpjMM1bMBeL1buAa2LpokfP8FCKA2gfiLn9QCLew61SQYeTAgRG9rqouR5zph6ECxDbU7qYTytd5f8QFw2jedB9Y7C9eBCRgg6XHiKgaRK7WFN9iNYB3L4KGYGQCJZUTW7zLNxX2uQrpR3sPsERnsQYSfar2dLhH2d7R8Avywn1aZPH2hDLp1ytjv8kgbUM74fT7Dton9dCVm3AV4zgc39eX9j42kx2U7mXWeG7MKDZUBiAVAfWyxvecjpMt5juzuXvzRmiN5qkZJ8QVFAQ2XLSpszFZtuPJET5pSbfYn1StqvALbGKM5jU4n6oSGuQdFmSehMbkjZwZJ5oNBf1X77ax9hXigt3ejqh"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9dTsGQVZNEnYLWhjXjomneDCT9XC6zGjeTW8ENTJVPooNnQBRuRdpwe9DoNVdLF4j3iy6Ld2p1eNLK8ek1bNSNrHFVjtsHaQVpcoHBy6nGghA9y7pr6ne6W7MRdAXLPHEDgpAodnRVxFSc2zyUbA48XKQVRtSVUneKjcdhDHP1hY9gC1EYLWtqn4weLzHKA84w77rHFhwwV2sbseHDqsmtQa4k5aiTCcXoUdmMpEuvadtSmKR33wciA3FNr5h125ce9ge1eSFPznvXNsCy7sA1rc4YJxipgwGoDDWSftUfnh5vY37WbwwNLhmMRvxtjHP7WBkSrLrKdamr8TdUu93cQyrykYRxbCben7pK1N75NknsSqLSihBnMTLo8gcFDGaDtQixPiJxMkefZ3qfHHwdvaL27"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9dXXyF2XfP9G56ZoZ8hAtLMc3i33FWSqa3FJgMkc7pTs5qdaeJHAFfJpwjoGWazHWzknPSh8d319L9xMoz2TsLCefqPeP8Kqf24C8fuY9RCTCvnAmecnXL6SJyiVTP6Vjjhcvvdk8cQHVUSnnXxbuufgRckyw9Mc7VrgpG3qzBfeYCfWkurDmdVnyjh7jPsZQM1sBjRFiNSjQQo7HQgaMi2YQp9WGpE4kJfF265eqXySrT8BycKLnjunED9B1TU6WNs2e9aFB2u82tMzwoRwTHWbAgc9rixwM3UiAAnhiEhx56nkidGRsqo2LR5AQASHEUP6aHtnz1wcwovEPthXXChDhRuDAa5PHvfz57LQpWZANf9HV26J5wYFysJb4bN9vRcxTJdijRzDAt5QbBJs5DJo64N"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9eKW2e2edvTfmZXDVdt8qTxwzuUVFHUtpf9JGKZN9damEuDoy381bwjwq6g5D4M6Zn2KUh2giSEktUca7nbvoM7L23aq9XXtsmyQKvVBseCVvUNmyERNHRZpzvzNCn6gwnzyMR58uBcibUBwV5441jYR18RxwwyKh2w8S6ogEmdrERAxdMYdxPXwj43Ve5aHnZtT8ZfV6vPPKnPmgyM95Bpw5ep1HmWoZvtF4s5WkqbCoaAYoBd94Bcysx4wzVQbvmU1SCD5hCpF3nDtE9n7G3SnNAerDE3DWiZt5GjpLtSpQyAjbaQekqpBnyKVqbsnPiDh5EynFqzETZLkt9p9hvAK2EuzQXBVZXpFwbwoiiw918LhZ2nw11xy2eQF6hGXG5GzaYYiBHmA4Fyd8rywPGA1QH4"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9a4wsKYVdTBhFsvZ4jwhzaWjFsondoUtyYpqF3hjWmPEUvMJF1Wh9NMhRVGaLYW8JtmS2JcP3Zq1L2AvYjYJaWm5CRqoYPz3DUEeJFSbUDdZPcp5YAL3FizX3J192zR3kWA967p6cZUgVP6e9LbHBdQBFGCQYRYKw8DNRCUzophTmoFeFwjWJF3EuXAWX2gv5ASj2Nem9YytNtjhZHCRS7Vz8pvgLdMJVa7B9fpS6oBV7nS4LWdeZc8NPC9VGKbTa1MCy7YkXfjbHfKpowEArnB9CLaLpWmSiaZruTiRxtZQZkU9z3YCMZW2dW5SHmwMGEseu6WwPPqgLz32tazKzNzHgiAJp561pxfCm7HF4r6VxtTqPKe8gjwLfDZqw3X2ew6rQ8Vo2csnjWrSQJAxYuMeinW"] = 1
	assignedCandidates["121VhftSAygpEJZ6i9jGk9a1qL1ZjK3QhzqpDT4uJop7tnEfahvCVRVbKPsuH94uGMqj1a1nLQSAcQypiUP3yc1s3t4jCRae8Kf6VXJRyHngt9X9iT2yJexKgRhnjzTfJYU4VkXV3w3xwiFCp5pGnDdocA8aq1SUCVMwfKAnhxjDHQDDWMMaxSLvjU56ZqaNZtsCQGTaySZLxc2RmCgMUeC5JQkF4t3P2NbTaZzz9JreLtZqSL3DVLacPwXy2enu3QMQYcPbZRVbz2Rxuf8e3hHYwjbAqvDJayCk9iasJnmZKP2gVRQhHUcX7cED4U22TMzi2rE6FoVMebThoDB2Dp21BW21qthS26Rkxe7UbxTesss9Xk1LSQsh8tRV1yxyGJ3DgtY2csBicRmT5PxQ3j9FwdSViKX82M1u9qv7ZcQSnvrJ"] = 0
	assignedCandidates["121VhftSAygpEJZ6i9jGk9dPK5DE41KmqUhp2EFJXyyHNrQomEh7QW7icEh2zymM5953C2HN2kYYCbepkL37Qny6mgyxGiAxxqEEkCHnqK4FsNBDbGwojMmVCgURwmS1oRqpfoFYhKwFUTzjyYduxBNegdgUw9VEYXJe3xSMBib24oBRhjbmK7LZ5qxRqKMsV7Lp49rPi2QGiizqiVKSsaAD2QMHEaC5ZBsGJvtRRMdGarHfe6yLtWAvcPLpsuSkYHfNni5Nh61LT1LwYLqSwDqNVKC3ew6RfRvmFfFTPhBGUcbhRKog6djipwKW9RZ7jvpgSiwifLriiz8h25ziR3Guh7k1cgyYm6TxQHrrLBRUgXKGvLkeAEi9ThrCsuQnbE61re6UmjwCcbPd9j53fzVmm4JJ19kMsyGUL2m7obSsdxvM"] = 0
	assignedCandidates["121VhftSAygpEJZ6i9jGk9drax2iDha73FtSVnju8AYxEHLxLqrgcB5ocJPiJ3BBRcRgZ1TmTQxnEsSpSm3wEdaRd98Y7YEHBwrMsQdaPsA66MJeTxy9ZDpyAD82sWfYzHNA7Q8pjpBvCrvxKHQTz6NBRZXspvCtxozStN6mJMJWoMUyMBccZLgRMTN7dDXArcJVPtTVQWqjT15DToLbzY3qdnc1vdZDTq916qNdQ9PbCVwbswdqtdCxEwCoYo9uLS9gdkvJaJdU1wNuYFYvFgiAQFa6mgjNZWiDnLyYBtVX3VyfVGe4K8fRgG9bgj15ZG7UypBoQTjxxJJHDmMy23VHV3qSDr8bjLnhLVYgHmkpuHfhxFX2B9KXXhkc4XMgxxyC83HWaz2XvS1eNuTMVbKUd3tjCBkZQJszBDsKa5R7gJqH"] = 0
	assignedCandidates["121VhftSAygpEJZ6i9jGk9a5oQvgecAms7BtyyjGxpySigmDAdo8af26UKNmXYjNUhFVp4NN6RpRJFTGn57w7evPi3HkaF8ToXsCJ7ceaZ5p6hrCHLN6tKm5sjBEo3yusZZayurNMsrLGRhBE2i8Xhxkns8uN8c9WY5kcwSVsyD9f7399fMzRqtMB7TjyE4ad2KWDmteZZWuZB79jYB5wHWcxRyUxYaQ761gT8oMJ6FKr7wdPYAeuJ7Pai71Xi9YdPUDNQ1dZ7Uq1m7wxavKKax14Tuf9onu9oZDTeat7SNK1PDxjvf2uwkEmcHAp7qzp8c7igm8X6VjC6685gdThcdEHPxiwDsu3UxQyXK1fqwSHDHx3Ff7w5xKeDK8zJNghCLBbZ3HowQbT7hAKqu3N5puMKw5cjQJndA4trRw5yuzXxbf"] = 0
	Logger.Init(common.NewBackend(nil).Logger("test", true))
	return
}()

func TestUtils(t *testing.T) {
	temp := []string{}
	shardIDs := make(map[string]byte)
	m := make(map[string]string)
	shuffledCandidates := []string{}
	for _, candidate := range candidates {
		seed := strconv.Itoa(int(randomNumber)) + candidate
		hash := common.HashH([]byte(seed)).String()
		temp = append(temp, hash)
		m[hash] = candidate
		shardID := calculateCandidateShardID(candidate, randomNumber, 2)
		shardIDs[candidate] = shardID
	}
	sort.Strings(temp)
	for _, key := range temp {
		shuffledCandidates = append(shuffledCandidates, m[key])
	}
}

func TestShuffleShardCandidate(t *testing.T) {
	candidates1 := candidates1
	randomNumber1 := int64(1000)
	shuffledCandiates1 := shuffleShardCandidate(candidates1, randomNumber1)

	candidates2 := candidates2
	randomNumber2 := int64(1000)
	shuffledCandiates2 := shuffleShardCandidate(candidates2, randomNumber2)

	candidates3 := candidates3
	randomNumber3 := int64(2)
	shuffledCandiates3 := shuffleShardCandidate(candidates3, randomNumber3)

	candidates4 := candidates2
	randomNumber4 := int64(45868549000)
	shuffledCandiates4 := shuffleShardCandidate(candidates4, randomNumber4)

	candidates5 := candidates3
	randomNumber5 := int64(9223372036854775807)
	shuffledCandiates5 := shuffleShardCandidate(candidates5, randomNumber5)

	repeatTest := func(candidatesTest []string, randomNumberTest int64, shuffledCandiatesExpect []string) {
		count := 100
		for i := 0; i < count; i++ {
			shuffledCandiates := shuffleShardCandidate(candidatesTest, randomNumberTest)
			if !reflect.DeepEqual(shuffledCandiates, shuffledCandiatesExpect) {
				log.Fatalf("Expect shuffled candidates to be %+v \n but get %+v", shuffledCandiatesExpect, shuffledCandiates)
			}
		}
	}

	t.Log("Testcase 1")
	repeatTest(candidates1, randomNumber1, shuffledCandiates1)
	t.Log("Testcase 1 done")

	t.Log("Testcase 2")
	repeatTest(candidates2, randomNumber2, shuffledCandiates2)
	t.Log("Testcase 2 done")

	t.Log("Testcase 3")
	repeatTest(candidates3, randomNumber3, shuffledCandiates3)
	t.Log("Testcase 3 done")

	t.Log("Testcase 4")
	repeatTest(candidates4, randomNumber4, shuffledCandiates4)
	t.Log("Testcase 4 done")

	t.Log("Testcase 5")
	repeatTest(candidates5, randomNumber5, shuffledCandiates5)
	t.Log("Testcase 5 done")
}

func TestAssignShardCandidate(t *testing.T) {
	cloneNumberOfPendingValidatorMap := func(src map[byte]int) map[byte]int {
		dst := make(map[byte]int)
		// Copy from the original map to the target map
		for key, value := range src {
			dst[key] = value
		}
		return dst
	}
	candidates1 := candidates1
	numberOfPendingValidator1 := make(map[byte]int)
	numberOfPendingValidator1[0] = 0
	numberOfPendingValidator1[1] = 0
	numberOfPendingValidator1Clone := cloneNumberOfPendingValidatorMap(numberOfPendingValidator1)
	testnetAssignOffset1 := 3
	activeShards1 := 2
	randomNumber1 := int64(1000)
	remainCandidates1, newAssignCandidates1 := assignShardCandidate(candidates1, numberOfPendingValidator1Clone, randomNumber1, testnetAssignOffset1, activeShards1)

	candidates2 := candidates2
	numberOfPendingValidator2 := make(map[byte]int)
	numberOfPendingValidator2[0] = 0
	numberOfPendingValidator2[1] = 0
	numberOfPendingValidator2Clone := cloneNumberOfPendingValidatorMap(numberOfPendingValidator2)
	testnetAssignOffset2 := 3
	activeShards2 := 8
	randomNumber2 := int64(4503)
	remainCandidates2, newAssignCandidates2 := assignShardCandidate(candidates2, numberOfPendingValidator2Clone, randomNumber2, testnetAssignOffset2, activeShards2)

	candidates3 := candidates2
	numberOfPendingValidator3 := make(map[byte]int)
	numberOfPendingValidator3[0] = 0
	numberOfPendingValidator3[1] = 0
	numberOfPendingValidator3Clone := cloneNumberOfPendingValidatorMap(numberOfPendingValidator3)
	testnetAssignOffset3 := 4
	activeShards3 := 8
	randomNumber3 := int64(9223372036854775807)
	remainCandidates3, newAssignCandidates3 := assignShardCandidate(candidates3, numberOfPendingValidator3Clone, randomNumber3, testnetAssignOffset3, activeShards3)

	candidates4 := candidates3
	numberOfPendingValidator4 := make(map[byte]int)
	numberOfPendingValidator4[0] = 0
	numberOfPendingValidator4[1] = 0
	numberOfPendingValidator4Clone := cloneNumberOfPendingValidatorMap(numberOfPendingValidator4)
	testnetAssignOffset4 := 5
	activeShards4 := 128
	randomNumber4 := int64(9223372036854775807)
	remainCandidates4, newAssignCandidates4 := assignShardCandidate(candidates4, numberOfPendingValidator4Clone, randomNumber4, testnetAssignOffset4, activeShards4)

	candidates5 := candidates3
	numberOfPendingValidator5 := make(map[byte]int)
	numberOfPendingValidator5[0] = 0
	numberOfPendingValidator5[1] = 0
	numberOfPendingValidator5Clone := cloneNumberOfPendingValidatorMap(numberOfPendingValidator5)
	testnetAssignOffset5 := 4
	activeShards5 := 9
	randomNumber5 := int64(45868549000)
	remainCandidates5, newAssignCandidates5 := assignShardCandidate(candidates5, numberOfPendingValidator5Clone, randomNumber5, testnetAssignOffset5, activeShards5)

	repeatTest := func(candidatesTest []string, numberOfPendingValidatorTest map[byte]int, randomNumberTest int64, testnetAssignOffsetTest int, activeShardsTest int, remainCandidatesExpect []string, newAssignCandidatesExpect map[byte][]string) {
		count := 100
		for i := 0; i < count; i++ {
			numberOfPendingValidatorTestClone := cloneNumberOfPendingValidatorMap(numberOfPendingValidatorTest)
			remainCandidates, newAssignCandidates := assignShardCandidate(candidatesTest, numberOfPendingValidatorTestClone, randomNumberTest, testnetAssignOffsetTest, activeShardsTest)
			if !reflect.DeepEqual(remainCandidatesExpect, remainCandidates) {
				t.Fatalf("Expected remain candidate to be %+v \n but get %+v", remainCandidates, remainCandidatesExpect)
			}
			if !reflect.DeepEqual(newAssignCandidatesExpect, newAssignCandidates) {
				t.Fatalf("Expected assign candidate to be %+v \n but get %+v", newAssignCandidates, newAssignCandidatesExpect)
			}
		}
	}

	t.Log("Testcase 1")
	repeatTest(candidates1, numberOfPendingValidator1, randomNumber1, testnetAssignOffset1, activeShards1, remainCandidates1, newAssignCandidates1)
	t.Log("Testcase 1 done")

	t.Log("Testcase 2")
	repeatTest(candidates2, numberOfPendingValidator2, randomNumber2, testnetAssignOffset2, activeShards2, remainCandidates2, newAssignCandidates2)
	t.Log("Testcase 2 done")

	t.Log("Testcase 3")
	repeatTest(candidates3, numberOfPendingValidator3, randomNumber3, testnetAssignOffset3, activeShards3, remainCandidates3, newAssignCandidates3)
	t.Log("Testcase 3 done")

	t.Log("Testcase 4")
	repeatTest(candidates4, numberOfPendingValidator4, randomNumber4, testnetAssignOffset4, activeShards4, remainCandidates4, newAssignCandidates4)
	t.Log("Testcase 4 done")

	t.Log("Testcase 5")
	repeatTest(candidates5, numberOfPendingValidator5, randomNumber5, testnetAssignOffset5, activeShards5, remainCandidates5, newAssignCandidates5)
	t.Log("Testcase 5 done")

}

func Test_swap(t *testing.T) {
	type args struct {
		badPendingValidators  []string
		goodPendingValidators []string
		currentGoodProducers  []string
		currentBadProducers   []string
		maxCommittee          int
		offset                int
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want1   []string
		want2   []string
		want3   []string
		wantErr bool
	}{
		{
			name: "swap case 1",
			args: args{
				goodPendingValidators: []string{"val12"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          3,
				offset:                0,
			},
			want:    []string{"val12"},
			want1:   []string{"val1", "val2", "val3"},
			want2:   nil,
			want3:   []string{},
			wantErr: false,
		},
		{
			name: "swap case 2",
			args: args{
				goodPendingValidators: []string{"val12"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          3,
				offset:                4,
			},
			want:    []string{"val12"},
			want1:   []string{"val1", "val2", "val3"},
			want2:   nil,
			want3:   []string{},
			wantErr: true,
		},
		{
			name: "swap case 3",
			args: args{
				goodPendingValidators: []string{"val12", "val22", "val32"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          6,
				offset:                3,
			},
			want:    []string{},
			want1:   []string{"val1", "val2", "val3", "val12", "val22", "val32"},
			want2:   nil,
			want3:   []string{"val12", "val22", "val32"},
			wantErr: false,
		},
		{
			name: "swap case 4",
			args: args{
				goodPendingValidators: []string{"val12", "val22", "val32", "val42"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          4,
				offset:                3,
			},
			want:    []string{"val42"},
			want1:   []string{"val3", "val12", "val22", "val32"},
			want2:   []string{"val1", "val2"},
			want3:   []string{"val12", "val22", "val32"},
			wantErr: false,
		},
		{
			name: "swap case 5",
			args: args{
				goodPendingValidators: []string{"val12", "val22", "val32"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          3,
				offset:                4,
			},
			want:    []string{"val12", "val22", "val32"},
			want1:   []string{"val1", "val2", "val3"},
			want2:   nil,
			want3:   []string{},
			wantErr: true,
		},
		{
			name: "swap case 6",
			args: args{
				goodPendingValidators: []string{"val12", "val22", "val32", "val42"},
				currentGoodProducers:  []string{"val1", "val2", "val3"},
				maxCommittee:          3,
				offset:                3,
			},
			want:    []string{"val42"},
			want1:   []string{"val12", "val22", "val32"},
			want2:   []string{"val1", "val2", "val3"},
			want3:   []string{"val12", "val22", "val32"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := swap(tt.args.badPendingValidators, tt.args.goodPendingValidators, tt.args.currentGoodProducers, tt.args.currentBadProducers, tt.args.maxCommittee, tt.args.offset)
			if (err != nil) != tt.wantErr {
				t.Errorf("swap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("swap() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("swap() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("swap() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("swap() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func TestSwapValidator(t *testing.T) {
	type args struct {
		pendingValidators  []string
		currentValidators  []string
		maxCommittee       int
		minCommittee       int
		offset             int
		producersBlackList map[string]uint8
		swapOffset         int
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want1   []string
		want2   []string
		want3   []string
		wantErr bool
	}{
		{
			name: "swap case 1",
			args: args{
				pendingValidators: []string{"val12"},
				currentValidators: []string{"val1", "val2", "val3"},
				maxCommittee:      3,
				minCommittee:      1,
				offset:            1,
				swapOffset:        1,
			},
			want:    []string{},
			want1:   []string{"val2", "val3", "val12"},
			want2:   []string{"val1"},
			want3:   []string{"val12"},
			wantErr: false,
		},
		{
			name: "swap case 2",
			args: args{
				pendingValidators: []string{"val12"},
				currentValidators: []string{"val1", "val2", "val3"},
				maxCommittee:      4,
				minCommittee:      1,
				offset:            3,
				swapOffset:        1,
			},
			want:    []string{},
			want1:   []string{"val1", "val2", "val3", "val12"},
			want2:   []string{},
			want3:   []string{"val12"},
			wantErr: false,
		},
		{
			name: "swap case 3",
			args: args{
				pendingValidators: []string{"val12", "val22"},
				currentValidators: []string{"val1", "val2", "val3"},
				maxCommittee:      3,
				minCommittee:      1,
				offset:            1,
				swapOffset:        1,
			},
			want:    []string{"val22"},
			want1:   []string{"val2", "val3", "val12"},
			want2:   []string{"val1"},
			want3:   []string{"val12"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3, err := SwapValidator(tt.args.pendingValidators, tt.args.currentValidators, tt.args.maxCommittee, tt.args.minCommittee, tt.args.offset, tt.args.producersBlackList, tt.args.swapOffset)
			if (err != nil) != tt.wantErr {
				t.Errorf("SwapValidator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SwapValidator() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("SwapValidator() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("SwapValidator() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("SwapValidator() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func TestRemoveValidator(t *testing.T) {
	type args struct {
		validators        []string
		removedValidators []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "error case",
			args: args{
				validators:        []string{"val1", "val2", "val3"},
				removedValidators: []string{"val1", "val2", "val3", "val4"},
			},
			want:    []string{"val1", "val2", "val3"},
			wantErr: true,
		},
		{
			name: "happy case 1",
			args: args{
				validators:        []string{"val1", "val2", "val3"},
				removedValidators: []string{"val1"},
			},
			want:    []string{"val2", "val3"},
			wantErr: false,
		},
		{
			name: "happy case 2",
			args: args{
				validators:        []string{"val1", "val2", "val3"},
				removedValidators: []string{"val2"},
			},
			want:    []string{"val1", "val3"},
			wantErr: false,
		},
		{
			name: "happy case 3",
			args: args{
				validators:        []string{"val1", "val2", "val3"},
				removedValidators: []string{"val3"},
			},
			want:    []string{"val1", "val2"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RemoveValidator(tt.args.validators, tt.args.removedValidators)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveValidator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveValidator() got = %v, want %v", got, tt.want)
			}
		})
	}
}

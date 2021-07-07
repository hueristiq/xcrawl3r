package exts

import (
	"regexp"
)

// S3finder is a
func S3finder(source string) (buckets []string, err error) {
	s3Regex, err := regexp.Compile(`(?i)[a-z0-9.-]+\.s3\.amazonaws\.com|[a-z0-9.-]+\.s3-[a-z0-9-]\.amazonaws\.com|[a-z0-9.-]+\.s3-website[.-](eu|ap|us|ca|sa|cn)|//s3\.amazonaws\.com/[a-z0-9._-]+|//s3-[a-z0-9-]+\.amazonaws\.com/[a-z0-9._-]+`)
	if err != nil {
		return buckets, err
	}

	for _, match := range s3Regex.FindAllStringSubmatch(source, -1) {
		buckets = append(buckets, decodeChars(match[0]))
	}

	return buckets, nil
}

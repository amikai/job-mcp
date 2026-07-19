# geo and industry take slugs from ?get=locations / ?get=industries;
# tag is free text matched against title and description.
GET https://jobicy.com/api/v2/remote-jobs?count=3&geo=usa&industry=dev&tag=golang

HTTP 200

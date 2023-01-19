# .repodata Parser

The `repodata` module is designed to take parameters specifying how to filter a list of HashiCorp github repos. It does this by grabbing the repo data itself through a go package called [go-github](https://github.com/google/go-github) that allows the module to utilize the github cli to make the required api call. The filtering and compiling is then done within the module.

An example invocation of the module might look like the following:

```sh
copywrite report
```

Because there are no specific filters stated in this call, the default filters, namely Name, License and HTMLURL, will be used on the data

An example invocation of the module with custom filters might look like the following:

```sh
copywrite parse --fields Name,Language,License,UpdatedAt
```

## Compatiblity

The module currently supports data types of *string,*License and *Timestamp only. More can and will be implemented in the future. All data types are listed [here](https://github.com/google/go-github/blob/0b5813fe43cc374cacb2e7492861af7d12199377/github/repos.go#L270:~:text=type-,Repository,-struct%20%7B). The module also only supports a csv as an output file, but it is designed to allow for other output files to be implemented as well.

# AuthCheck

AuthCheck is a powerful authentication testing tool designed to identify potential authentication bypass vulnerabilities by comparing HTTP responses across different authentication methods.

## Features

- Compare responses between:
  - Cookies vs No Cookies
  - Two different Cookie headers
  - Bearer token vs No Bearer token
  - Two different Bearer tokens
- Concurrent request processing
- Progress bar visualization
- Automatic retry mechanism
- Detailed response comparison including:
  - Response status codes
  - Response body sizes
  - Side-by-side comparison
- Filters out static files (.js, .map, .svg)

## Installation

```bash
go install github.com/fractalized-cyber/authcheck@latest
```

## Usage

```bash
authcheck [options] -f <file_with_endpoints>
```

### Options

- `-f <file>`        File containing endpoints (one per line)
- `-version`         Show version information
- `-mode <number>`   Operation mode (1-4):
  - 1: Cookies -> No Cookies
  - 2: Compare Two Cookies
  - 3: Bearer Token -> No Bearer Token
  - 4: Compare Two Bearer Tokens
- `-c1 <cookie>`     First cookie header
- `-c2 <cookie>`     Second cookie header (for mode 2)
- `-t1 <token>`      First bearer token
- `-t2 <token>`      Second bearer token (for mode 4)

### Examples

Compare with/without cookie:
```bash
authcheck -f endpoints.txt -mode 1 -c1 "session=abc123"
```

Compare two different cookies:
```bash
authcheck -f endpoints.txt -mode 2 -c1 "session=abc123" -c2 "session=xyz789"
```

Compare with/without bearer token:
```bash
authcheck -f endpoints.txt -mode 3 -t1 "eyJ0eXAi..."
```

Compare two different bearer tokens:
```bash
authcheck -f endpoints.txt -mode 4 -t1 "eyJ0eXAi..." -t2 "eyKhbGci..."
```

## Output

The tool reports endpoints where both requests return HTTP 200 status codes with the same size response.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

- [Fractalized Cyber](https://github.com/fractalized-cyber) 

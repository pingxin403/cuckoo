# API Generated Code

This directory contains all protobuf-generated code for the Cuckoo project, organized by programming language.

## Directory Structure

```
api/gen/
├── go/                    # Go generated code
│   ├── go.mod
│   ├── authpb/
│   ├── userpb/
│   ├── impb/
│   └── ...
├── java/                  # Java generated code
│   └── com/pingxin403/cuckoo/
│       ├── authpb/
│       ├── userpb/
│       └── ...
└── typescript/            # TypeScript generated code
    ├── package.json
    ├── tsconfig.json
    ├── authpb/
    ├── userpb/
    └── ...
```

## Usage

### Go Services

Import the generated code in your Go services:

```go
import (
    "github.com/pingxin403/cuckoo/api/gen/go/authpb"
    "github.com/pingxin403/cuckoo/api/gen/go/userpb"
    "github.com/pingxin403/cuckoo/api/gen/go/impb"
)
```

Add to your `go.mod`:

```go
require (
    github.com/pingxin403/cuckoo/api/gen/go v0.0.0-00010101000000-000000000000
)

replace (
    github.com/pingxin403/cuckoo/api/gen/go => ../../api/gen/go
)
```

### Java Services

Add to your `build.gradle`:

```gradle
dependencies {
    implementation files('../../api/gen/java')
}
```

Import in your Java code:

```java
import com.pingxin403.cuckoo.authpb.*;
import com.pingxin403.cuckoo.userpb.*;
```

### TypeScript Services

Add to your `package.json`:

```json
{
  "dependencies": {
    "@cuckoo/api-gen": "file:../../api/gen/typescript"
  }
}
```

Import in your TypeScript code:

```typescript
import { AuthService } from '@cuckoo/api-gen/authpb/auth';
import { UserService } from '@cuckoo/api-gen/userpb/user';
```

## Generating Code

To regenerate all protobuf code:

```bash
make proto
```

To generate code for a specific language:

```bash
make gen-proto-go
make gen-proto-java
make gen-proto-typescript
```

## Verification

To verify that generated code is up to date:

```bash
make verify-proto
```

This command will:
1. Regenerate all code
2. Check if there are any differences with git
3. Fail if code is out of date

## Notes

- All generated code is committed to git
- Never manually edit generated code
- Always run `make proto` after modifying `.proto` files
- CI will verify that generated code is up to date

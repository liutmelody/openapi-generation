# TestGroup.Tag3

## Overview

### Available Operations

* [post_test](#post_test) - Post Test2

## post_test

This is a test endpoint.
It has a description.

### Example Usage

<!-- UsageSnippet language="python" operationID="postTest2" method="post" path="/test2" -->
```python
# Synchronous Example
from datetime import date
from decimal import Decimal
import os
import speakeasy.new_openapi
from speakeasy.new_openapi import SDK
from speakeasy.new_openapi.utils import parse_datetime


with SDK(
    deprecated_query_param1="some example query param",
    deprecated_query_param2="some example query param",
    security=speakeasy.new_openapi.Security(
        my_api_key=speakeasy.new_openapi.MyAPIKey(
            my_api_key=os.getenv("SPEAKEASY_MY_API_KEY", ""),
        ),
    ),
) as sdk:

    res = sdk.test_group.tag3.post_test(obj=speakeasy.new_openapi.ExhaustiveObject(
        str_="example",
        bool_=True,
        integer=999999,
        int32=1,
        num=1.1,
        float32=8499.3,
        date_=date.fromisoformat("2020-01-01"),
        date_time=parse_datetime("2020-01-01T00:00:00Z"),
        anything="<value>",
        bool_opt=True,
        int_opt_null=999999,
        num_opt_null=1.1,
        int_enum=3,
        int32_enum=69,
        bigint=593288,
        decimal_str=Decimal("7028.3"),
        obj=speakeasy.new_openapi.SimpleObject(
            str_="example",
        ),
        map={

        },
        arr=[],
        any="<value>",
        nullable_int_enum=3,
        nullable_string_enum="Second",
        color="green",
        icon="tick",
        hero_width=480,
    ), type_="type1")

    assert res is not None

    # Handle response
    print(res)
```

</br>

An Async SDK client can also be used to make asynchronous requests by importing it and asyncio.

```python
# Asynchronous Example
import asyncio
from datetime import date
from decimal import Decimal
import os
import speakeasy.new_openapi
from speakeasy.new_openapi import AsyncSDK
from speakeasy.new_openapi.utils import parse_datetime

async def main():

    async with AsyncSDK(
        deprecated_query_param1="some example query param",
        deprecated_query_param2="some example query param",
        security=speakeasy.new_openapi.Security(
            my_api_key=speakeasy.new_openapi.MyAPIKey(
                my_api_key=os.getenv("SPEAKEASY_MY_API_KEY", ""),
            ),
        ),
    ) as sdk:

        res = await sdk.test_group.tag3.post_test(obj=speakeasy.new_openapi.ExhaustiveObject(
            str_="example",
            bool_=True,
            integer=999999,
            int32=1,
            num=1.1,
            float32=8499.3,
            date_=date.fromisoformat("2020-01-01"),
            date_time=parse_datetime("2020-01-01T00:00:00Z"),
            anything="<value>",
            bool_opt=True,
            int_opt_null=999999,
            num_opt_null=1.1,
            int_enum=3,
            int32_enum=69,
            bigint=593288,
            decimal_str=Decimal("7028.3"),
            obj=speakeasy.new_openapi.SimpleObject(
                str_="example",
            ),
            map={

            },
            arr=[],
            any="<value>",
            nullable_int_enum=3,
            nullable_string_enum="Second",
            color="green",
            icon="tick",
            hero_width=480,
        ), type_="type1")

        assert res is not None

        # Handle response
        print(res)

asyncio.run(main())
```

### Parameters

| Parameter                                                                                                                                                 | Type                                                                                                                                                      | Required                                                                                                                                                  | Description                                                                                                                                               | Example                                                                                                                                                   |
| --------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `obj`                                                                                                                                                     | [models.ExhaustiveObject](../../models/exhaustiveobject.md)                                                                                               | :heavy_check_mark:                                                                                                                                        | A simple object that uses all our supported primitive types and enums and has optional properties.<br/><br/>[A link to the external docs.](https://speakeasy.com) |                                                                                                                                                           |
| `deprecated_query_param1`                                                                                                                                 | *Optional[str]*                                                                                                                                           | :heavy_minus_sign:                                                                                                                                        | :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.                                      | some example query param                                                                                                                                  |
| `deprecated_query_param2`                                                                                                                                 | *Optional[str]*                                                                                                                                           | :heavy_minus_sign:                                                                                                                                        | :warning: **DEPRECATED**: This will be removed in a future release, please migrate away from it as soon as possible.                                      | some example query param                                                                                                                                  |
| `type_`                                                                                                                                                   | [Optional[models.Type]](../../models/type.md)                                                                                                             | :heavy_minus_sign:                                                                                                                                        | N/A                                                                                                                                                       | type1                                                                                                                                                     |
| `retries`                                                                                                                                                 | [Optional[utils.RetryConfig]](../../models/utils/retryconfig.md)                                                                                          | :heavy_minus_sign:                                                                                                                                        | Configuration to override the default retry behavior of the client.                                                                                       |                                                                                                                                                           |
| `server_url`                                                                                                                                              | *Optional[str]*                                                                                                                                           | :heavy_minus_sign:                                                                                                                                        | An optional server URL to use.                                                                                                                            | http://localhost:8080                                                                                                                                     |

### Response

**[bytes](../../models/.md)**

### Errors

| Error Type                     | Status Code                    | Content Type                   |
| ------------------------------ | ------------------------------ | ------------------------------ |
| models.BadRequestResponseError | 400                            | application/json               |
| models.ErrorsError             | 404                            | application/json               |
| models.Test2ResponseError      | 500                            | application/json               |
| models.SDKError                | 4XX, 5XX                       | \*/\*                          |